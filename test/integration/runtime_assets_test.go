package integration_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/plan"
)

// TestRuntimeAssets_Config verifies that runtime_assets in a config file
// are parsed correctly and validated.
func TestRuntimeAssets_Config(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "config-runtime-assets"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config, want non-nil")
	}
	if len(cfg.RuntimeAssets) != 2 {
		t.Fatalf("RuntimeAssets len = %d, want 2", len(cfg.RuntimeAssets))
	}
	if cfg.RuntimeAssets[0].Src != "configs/app.yaml" {
		t.Errorf("RuntimeAssets[0].Src = %q, want %q", cfg.RuntimeAssets[0].Src, "configs/app.yaml")
	}
	if cfg.RuntimeAssets[0].Dst != "/app/configs/app.yaml" {
		t.Errorf("RuntimeAssets[0].Dst = %q, want %q", cfg.RuntimeAssets[0].Dst, "/app/configs/app.yaml")
	}
	if cfg.RuntimeAssets[1].Src != "static/index.html" {
		t.Errorf("RuntimeAssets[1].Src = %q, want %q", cfg.RuntimeAssets[1].Src, "static/index.html")
	}
	if cfg.RuntimeAssets[1].Dst != "/app/static/index.html" {
		t.Errorf("RuntimeAssets[1].Dst = %q, want %q", cfg.RuntimeAssets[1].Dst, "/app/static/index.html")
	}
}

// TestRuntimeAssets_PlanEmitsCOPYInRuntimeStage verifies that WithRuntimeAssets
// injects 2 COPY steps into the runtime stage in declaration order,
// after framework CopyFrom steps and before CMD/ENTRYPOINT.
func TestRuntimeAssets_PlanEmitsCOPYInRuntimeStage(t *testing.T) {
	fw := &core.Framework{
		Name:         "go",
		BuildCommand: "go build -o app .",
		Port:         8080,
	}
	assets := []core.AssetCopy{
		{Src: "configs/app.yaml", Dst: "/app/configs/app.yaml"},
		{Src: "static/index.html", Dst: "/app/static/index.html"},
	}
	p, err := plan.Plan(fw, plan.WithRuntimeAssets(assets))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Runtime stage is the last stage.
	last := p.Stages[len(p.Stages)-1]

	// Collect COPY steps (not CopyFrom) from the runtime stage.
	var copySteps []core.Step
	for _, s := range last.Steps {
		if s.Type == core.StepCopy {
			copySteps = append(copySteps, s)
		}
	}
	if len(copySteps) < 2 {
		t.Fatalf("runtime stage has %d StepCopy steps, want at least 2; steps: %v", len(copySteps), last.Steps)
	}

	// The last 2 COPY steps must be the user assets in declaration order.
	got0 := copySteps[len(copySteps)-2]
	got1 := copySteps[len(copySteps)-1]

	if len(got0.Args) < 2 || got0.Args[0] != "configs/app.yaml" || got0.Args[1] != "/app/configs/app.yaml" {
		t.Errorf("asset COPY[0] args = %v, want [configs/app.yaml /app/configs/app.yaml]", got0.Args)
	}
	if len(got1.Args) < 2 || got1.Args[0] != "static/index.html" || got1.Args[1] != "/app/static/index.html" {
		t.Errorf("asset COPY[1] args = %v, want [static/index.html /app/static/index.html]", got1.Args)
	}

	// Verify CMD comes after the asset COPY steps.
	assetCopyIdx := -1
	cmdIdx := -1
	for i, s := range last.Steps {
		if s.Type == core.StepCopy && len(s.Args) >= 2 && s.Args[0] == "configs/app.yaml" {
			assetCopyIdx = i
		}
		if s.Type == core.StepCmd {
			cmdIdx = i
		}
	}
	if assetCopyIdx < 0 {
		t.Fatal("asset COPY step not found in runtime stage")
	}
	if cmdIdx >= 0 && cmdIdx <= assetCopyIdx {
		t.Errorf("CMD (idx %d) appears before asset COPY (idx %d); want CMD after", cmdIdx, assetCopyIdx)
	}
}

// TestRuntimeAssets_DockerfileOutput verifies the Dockerfile text contains
// the expected COPY lines for user assets.
func TestRuntimeAssets_DockerfileOutput(t *testing.T) {
	fw := &core.Framework{
		Name:         "go",
		BuildCommand: "go build -o app .",
		Port:         8080,
	}
	assets := []core.AssetCopy{
		{Src: "configs/app.yaml", Dst: "/app/configs/app.yaml"},
		{Src: "static/index.html", Dst: "/app/static/index.html"},
	}
	df, err := docksmith.GenerateDockerfile(fw, plan.WithRuntimeAssets(assets))
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}

	if !strings.Contains(df, "COPY configs/app.yaml /app/configs/app.yaml") {
		t.Errorf("Dockerfile missing COPY for configs/app.yaml:\n%s", df)
	}
	if !strings.Contains(df, "COPY static/index.html /app/static/index.html") {
		t.Errorf("Dockerfile missing COPY for static/index.html:\n%s", df)
	}
}

// TestRuntimeAssets_Chown verifies that AssetCopy.Chown causes COPY --chown=<value>
// to be emitted in the Dockerfile, and that missing chown leaves plain COPY.
func TestRuntimeAssets_Chown(t *testing.T) {
	fw := &core.Framework{
		Name:         "go",
		BuildCommand: "go build -o app .",
		Port:         8080,
	}
	assets := []core.AssetCopy{
		{Src: "atlas.hcl", Dst: "/app/atlas.hcl", Chown: "permanu:permanu"},
		{Src: "sql", Dst: "/app/sql", Chown: "1000:1000"},
		{Src: "migrations", Dst: "/app/migrations"},
	}
	df, err := docksmith.GenerateDockerfile(fw, plan.WithRuntimeAssets(assets))
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}

	if !strings.Contains(df, "COPY --chown=permanu:permanu atlas.hcl /app/atlas.hcl") {
		t.Errorf("Dockerfile missing COPY --chown=permanu:permanu:\n%s", df)
	}
	if !strings.Contains(df, "COPY --chown=1000:1000 sql /app/sql") {
		t.Errorf("Dockerfile missing COPY --chown=1000:1000:\n%s", df)
	}
	// No chown: plain COPY, no --chown flag.
	if !strings.Contains(df, "COPY migrations /app/migrations") {
		t.Errorf("Dockerfile missing plain COPY migrations:\n%s", df)
	}
	if strings.Contains(df, "COPY --chown") && strings.Contains(df, "migrations /app/migrations") {
		// Make sure the plain COPY line doesn't have --chown.
		for _, line := range strings.Split(df, "\n") {
			if strings.Contains(line, "migrations /app/migrations") && strings.Contains(line, "--chown") {
				t.Errorf("migrations COPY should be plain but got: %s", line)
			}
		}
	}
}

// TestRuntimeAssets_ChownConfig verifies chown roundtrips through config parse.
func TestRuntimeAssets_ChownConfig(t *testing.T) {
	toml := `runtime="go"
[start]
command="./app"
[[runtime_assets]]
src="atlas.hcl"
dst="/app/atlas.hcl"
chown="permanu:permanu"
[[runtime_assets]]
src="sql"
dst="/app/sql"
chown="1000:1000"
`
	cfg, err := config.ParseConfig("docksmith.toml", []byte(toml))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.RuntimeAssets[0].Chown != "permanu:permanu" {
		t.Errorf("RuntimeAssets[0].Chown = %q, want %q", cfg.RuntimeAssets[0].Chown, "permanu:permanu")
	}
	if cfg.RuntimeAssets[1].Chown != "1000:1000" {
		t.Errorf("RuntimeAssets[1].Chown = %q, want %q", cfg.RuntimeAssets[1].Chown, "1000:1000")
	}
}

// TestRuntimeAssets_ChownValidation verifies invalid chown values are rejected.
func TestRuntimeAssets_ChownValidation(t *testing.T) {
	cases := []struct {
		name  string
		chown string
	}{
		{name: "shell injection semicolon", chown: "foo;bar"},
		{name: "space in chown", chown: "foo bar"},
		{name: "dollar sign", chown: "$USER"},
		{name: "slash", chown: "foo/bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			toml := "runtime=\"go\"\n[start]\ncommand=\"./app\"\n[[runtime_assets]]\nsrc=\"config.yaml\"\ndst=\"/app/config.yaml\"\nchown=\"" + tc.chown + "\""
			cfg, parseErr := config.ParseConfig("docksmith.toml", []byte(toml))
			if parseErr != nil {
				return // parse-time failure is also acceptable
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected validation error for chown=%q, got nil", tc.chown)
			}
		})
	}
}

// TestRuntimeAssets_Validation verifies parse-time validation rejects
// bad src/dst combinations.
func TestRuntimeAssets_Validation(t *testing.T) {
	cases := []struct {
		name string
		toml string
	}{
		{
			name: "src with dotdot",
			toml: "runtime=\"go\"\n[start]\ncommand=\"./app\"\n[[runtime_assets]]\nsrc=\"../etc/passwd\"\ndst=\"/app/config\"",
		},
		{
			name: "src absolute",
			toml: "runtime=\"go\"\n[start]\ncommand=\"./app\"\n[[runtime_assets]]\nsrc=\"/etc/passwd\"\ndst=\"/app/config\"",
		},
		{
			name: "dst relative",
			toml: "runtime=\"go\"\n[start]\ncommand=\"./app\"\n[[runtime_assets]]\nsrc=\"config.yaml\"\ndst=\"app/config\"",
		},
		{
			name: "src empty",
			toml: "runtime=\"go\"\n[start]\ncommand=\"./app\"\n[[runtime_assets]]\nsrc=\"\"\ndst=\"/app/config\"",
		},
		{
			name: "dst empty",
			toml: "runtime=\"go\"\n[start]\ncommand=\"./app\"\n[[runtime_assets]]\nsrc=\"config.yaml\"\ndst=\"\"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, parseErr := config.ParseConfig("docksmith.toml", []byte(tc.toml))
			if parseErr != nil {
				// Some inputs may fail at parse time; that's acceptable.
				return
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected validation error for %q, got nil", tc.name)
			}
		})
	}
}
