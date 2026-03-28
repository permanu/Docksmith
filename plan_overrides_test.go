package docksmith

import (
	"fmt"
	"strings"
	"testing"
)

func goFrameworkForOverride() *Framework {
	return &Framework{
		Name:         "go-gin",
		GoVersion:    "1.22",
		Port:         8080,
		BuildCommand: "go build -o server .",
		StartCommand: "./server",
	}
}

func TestPlanWithUserOverride_AddsUserStep(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithUser("nonroot"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, step := range runtime.Steps {
		if step.Type == StepUser && len(step.Args) > 0 && step.Args[0] == "nonroot" {
			return
		}
	}
	t.Error("expected USER nonroot step in runtime stage")
}

func TestPlanWithUserOverride_EmptyString_RemovesExistingUser(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithUser(""))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, step := range runtime.Steps {
		if step.Type == StepUser {
			t.Errorf("WithUser(\"\") should remove USER step, found args=%v", step.Args)
		}
	}
}

func TestPlanWithHealthcheckDisabled_RemovesHealthcheck(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithHealthcheckDisabled())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type == StepHealthcheck {
				t.Error("healthcheck should be removed when disabled")
			}
		}
	}
}

func TestPlanWithHealthcheck_AddsStep(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithHealthcheck("curl -f http://localhost:8080/health"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, step := range runtime.Steps {
		if step.Type == StepHealthcheck {
			return
		}
	}
	t.Error("expected HEALTHCHECK step in runtime stage")
}

func TestPlanWithExposeOverride_ChangesPort(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithExpose(9000))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Expose != 9000 {
		t.Errorf("Expose = %d, want 9000", plan.Expose)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, step := range runtime.Steps {
		if step.Type == StepExpose {
			if len(step.Args) == 0 || step.Args[0] != "9000" {
				t.Errorf("EXPOSE step args = %v, want [9000]", step.Args)
			}
			return
		}
	}
	t.Error("expected EXPOSE step in runtime stage")
}

func TestPlanWithRuntimeImage_OverridesBaseImage(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithRuntimeImage("gcr.io/distroless/static:nonroot"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	if runtime.From != "gcr.io/distroless/static:nonroot" {
		t.Errorf("runtime stage From = %q, want distroless", runtime.From)
	}
}

func TestPlanWithExtraEnv_AddsEnvSteps(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithExtraEnv(map[string]string{
		"PORT": "8080",
		"ENV":  "production",
	}))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	envKeys := map[string]bool{}
	for _, step := range runtime.Steps {
		if step.Type == StepEnv && len(step.Args) == 2 {
			envKeys[step.Args[0]] = true
		}
	}
	if !envKeys["PORT"] || !envKeys["ENV"] {
		t.Errorf("extra env steps missing, found keys: %v", envKeys)
	}
}

func TestPlanWithEntrypoint_ReplacesCmd(t *testing.T) {
	plan, err := Plan(goFrameworkForOverride(), WithEntrypoint("/bin/sh", "-c", "./server"))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, step := range runtime.Steps {
		if step.Type == StepEntrypoint {
			return
		}
	}
	t.Error("expected ENTRYPOINT step in runtime stage")
}

func TestPlanWithNoOptions_SameAsBefore(t *testing.T) {
	fw := goFrameworkForOverride()
	planWithout, err := Plan(fw)
	if err != nil {
		t.Fatalf("Plan without opts: %v", err)
	}
	planWith, err := Plan(fw)
	if err != nil {
		t.Fatalf("Plan with empty opts: %v", err)
	}
	if len(planWithout.Stages) != len(planWith.Stages) {
		t.Errorf("stage count differs: %d vs %d", len(planWithout.Stages), len(planWith.Stages))
	}
}

func TestGenerateDockerfileWithOptions_ExposeOverride(t *testing.T) {
	fw := goFrameworkForOverride()
	dockerfile, err := GenerateDockerfile(fw, WithExpose(9090))
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}
	if !strings.Contains(dockerfile, "EXPOSE 9090") {
		t.Errorf("dockerfile should contain EXPOSE 9090, got:\n%s", dockerfile)
	}
}

func TestBuildWithOptions_PassesPlanOptions(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, dir+"/docksmith.yaml", "runtime: go\nversion: \"1.22\"\nstart:\n  command: ./server\n")
	mustWriteFile(t, dir+"/go.mod", fmt.Sprintf("module testapp\n\ngo 1.22\n"))

	dockerfile, _, err := Build(dir, WithExpose(7777))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(dockerfile, "EXPOSE 7777") {
		t.Errorf("dockerfile should contain EXPOSE 7777")
	}
}
