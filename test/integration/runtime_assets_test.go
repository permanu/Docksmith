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
