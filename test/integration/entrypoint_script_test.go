package integration_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/plan"
)

// ---------------------------------------------------------------------------
// WithEntrypointScript — plan option
// ---------------------------------------------------------------------------

func TestWithEntrypointScript_CopiesAndSetsEntrypoint(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "go-gin",
		GoVersion:    "1.23",
		Port:         8080,
		BuildCommand: "go build -o server .",
		StartCommand: "./server",
	}
	p, err := docksmith.Plan(fw, docksmith.WithEntrypointScript("entrypoint.sh"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := p.Stages[len(p.Stages)-1]

	var foundCopy, foundEntrypoint bool
	for _, step := range runtime.Steps {
		if step.Type == docksmith.StepCopy {
			joined := strings.Join(step.Args, " ")
			if strings.Contains(joined, "entrypoint.sh") && strings.Contains(joined, "/entrypoint.sh") {
				foundCopy = true
			}
		}
		if step.Type == docksmith.StepEntrypoint {
			if len(step.Args) == 1 && step.Args[0] == "/entrypoint.sh" {
				foundEntrypoint = true
			}
		}
	}

	if !foundCopy {
		t.Errorf("expected COPY step for entrypoint.sh in runtime stage; steps: %+v", runtime.Steps)
	}
	if !foundEntrypoint {
		t.Errorf("expected ENTRYPOINT [\"/entrypoint.sh\"] in runtime stage; steps: %+v", runtime.Steps)
	}
}

func TestWithEntrypointScript_CopyHasChmod(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "go-gin",
		GoVersion:    "1.23",
		Port:         8080,
		BuildCommand: "go build -o server .",
		StartCommand: "./server",
	}
	p, err := docksmith.Plan(fw, docksmith.WithEntrypointScript("scripts/run.sh"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := p.Stages[len(p.Stages)-1]

	df := docksmith.EmitDockerfile(p)
	if !strings.Contains(df, "--chmod=755") {
		t.Errorf("expected --chmod=755 in emitted Dockerfile; got:\n%s", df)
	}
	_ = runtime
}

func TestWithEntrypointScript_CopyBeforeEntrypoint(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "go-gin",
		GoVersion:    "1.23",
		Port:         8080,
		BuildCommand: "go build -o server .",
		StartCommand: "./server",
	}
	p, err := docksmith.Plan(fw, docksmith.WithEntrypointScript("entrypoint.sh"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := p.Stages[len(p.Stages)-1]

	copyIdx, entrypointIdx := -1, -1
	for i, step := range runtime.Steps {
		if step.Type == docksmith.StepCopy {
			joined := strings.Join(step.Args, " ")
			if strings.Contains(joined, "entrypoint.sh") {
				copyIdx = i
			}
		}
		if step.Type == docksmith.StepEntrypoint {
			entrypointIdx = i
		}
	}

	if copyIdx == -1 {
		t.Fatal("COPY step for entrypoint.sh not found")
	}
	if entrypointIdx == -1 {
		t.Fatal("ENTRYPOINT step not found")
	}
	if copyIdx >= entrypointIdx {
		t.Errorf("COPY (idx=%d) must appear before ENTRYPOINT (idx=%d)", copyIdx, entrypointIdx)
	}
}

func TestWithEntrypointScript_BypassesTini(t *testing.T) {
	// When an entrypoint script is set, tini must not also be wired as entrypoint.
	// Node/Python normally get tini; this test verifies the script takes over.
	fw := &docksmith.Framework{
		Name:           "express",
		NodeVersion:    "22",
		PackageManager: "npm",
		Port:           3000,
		StartCommand:   "node server.js",
	}
	p, err := docksmith.Plan(fw, docksmith.WithEntrypointScript("entrypoint.sh"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := p.Stages[len(p.Stages)-1]

	var entrypoints []docksmith.Step
	for _, step := range runtime.Steps {
		if step.Type == docksmith.StepEntrypoint {
			entrypoints = append(entrypoints, step)
		}
	}
	if len(entrypoints) != 1 {
		t.Fatalf("expected exactly 1 ENTRYPOINT step, got %d: %+v", len(entrypoints), entrypoints)
	}
	if entrypoints[0].Args[0] != "/entrypoint.sh" {
		t.Errorf("ENTRYPOINT should be /entrypoint.sh, got %v", entrypoints[0].Args)
	}
}

func TestWithEntrypointScript_DockerfileOutput(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "go-gin",
		GoVersion:    "1.23",
		Port:         8080,
		BuildCommand: "go build -o server .",
		StartCommand: "./server",
	}
	p, err := docksmith.Plan(fw, docksmith.WithEntrypointScript("entrypoint.sh"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	df := docksmith.EmitDockerfile(p)

	assertContains(t, df, "COPY --chmod=755 entrypoint.sh /entrypoint.sh")
	assertContains(t, df, `ENTRYPOINT ["/entrypoint.sh"]`)
	// CMD must still be present (Start.Command passed as args to entrypoint)
	assertContains(t, df, "CMD")
}

// ---------------------------------------------------------------------------
// Config round-trip via fixture
// ---------------------------------------------------------------------------

func TestLoadConfig_CustomEntrypoint_Fixture(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "custom-entrypoint"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Start.EntrypointScript != "entrypoint.sh" {
		t.Errorf("EntrypointScript = %q, want entrypoint.sh", cfg.Start.EntrypointScript)
	}
}

func TestBuild_CustomEntrypoint_Fixture(t *testing.T) {
	// Load config manually and convert to plan options — this is the path the
	// CLI and public API take when a docksmith.toml is present.
	dir := filepath.Join("../../testdata", "fixtures", "custom-entrypoint")
	cfg, err := docksmith.LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	opts, err := docksmith.ConfigToPlanOptions(cfg)
	if err != nil {
		t.Fatalf("ConfigToPlanOptions: %v", err)
	}
	df, _, err := docksmith.Build(dir, opts...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	assertContains(t, df, "COPY --chmod=755 entrypoint.sh /entrypoint.sh")
	assertContains(t, df, `ENTRYPOINT ["/entrypoint.sh"]`)
}

// ---------------------------------------------------------------------------
// Validation — parse-time rejections
// ---------------------------------------------------------------------------

func TestValidateEntrypointScript_AbsolutePath_Rejected(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "go",
		Start: docksmith.StartConfig{
			Command:          "./server",
			EntrypointScript: "/etc/entrypoint.sh",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for absolute entrypoint_script path, got nil")
	}
}

func TestValidateEntrypointScript_PathTraversal_Rejected(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "go",
		Start: docksmith.StartConfig{
			Command:          "./server",
			EntrypointScript: "../outside/entrypoint.sh",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for path traversal in entrypoint_script, got nil")
	}
}

func TestValidateEntrypointScript_RelativePath_Accepted(t *testing.T) {
	cfg := &docksmith.Config{
		Runtime: "go",
		Start: docksmith.StartConfig{
			Command:          "./server",
			EntrypointScript: "scripts/entrypoint.sh",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error for relative entrypoint_script path, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PlanConfig field is set by WithEntrypointScript
// ---------------------------------------------------------------------------

func TestWithEntrypointScript_PlanConfigField(t *testing.T) {
	opts := []docksmith.PlanOption{docksmith.WithEntrypointScript("run.sh")}
	cfg := plan.ResolvePlanConfig(opts)
	if cfg.EntrypointScript != "run.sh" {
		t.Errorf("EntrypointScript = %q, want run.sh", cfg.EntrypointScript)
	}
}
