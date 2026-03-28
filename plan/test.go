package plan

import (
	"encoding/json"
	"testing"

	"github.com/permanu/docksmith/core"
)

func TestBuildPlanJSONRoundTrip(t *testing.T) {
	plan := core.BuildPlan{
		Framework:    "nextjs",
		Expose:       3000,
		Dockerignore: []string{"node_modules", ".next"},
		Stages: []core.Stage{
			{
				Name: "deps",
				From: "node:22-alpine",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{Type: core.StepCopy, Args: []string{"package.json", "package-lock.json", "./"}},
					{
						Type:       core.StepRun,
						Args:       []string{"npm ci"},
						CacheMount: &core.CacheMount{Target: "/root/.npm"},
					},
				},
			},
			{
				Name: "build",
				From: "deps",
				Steps: []core.Step{
					{Type: core.StepCopy, Args: []string{".", "."}},
					{Type: core.StepEnv, Args: []string{"NODE_ENV", "production"}},
					{Type: core.StepRun, Args: []string{"npm run build"}},
				},
			},
			{
				Name: "runtime",
				From: "node:22-alpine",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{
						Type:     core.StepCopyFrom,
						CopyFrom: &core.CopyFrom{Stage: "build", Src: ".next", Dst: ".next"},
						Link:     true,
					},
					{Type: core.StepExpose, Args: []string{"3000"}},
					{Type: core.StepCmd, Args: []string{"node", "server.js"}},
				},
			},
		},
	}

	data, err := json.Marshal(&plan)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got core.BuildPlan
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Framework != plan.Framework {
		t.Errorf("Framework: got %q, want %q", got.Framework, plan.Framework)
	}
	if got.Expose != plan.Expose {
		t.Errorf("Expose: got %d, want %d", got.Expose, plan.Expose)
	}
	if len(got.Dockerignore) != len(plan.Dockerignore) {
		t.Fatalf("Dockerignore len: got %d, want %d", len(got.Dockerignore), len(plan.Dockerignore))
	}
	if len(got.Stages) != len(plan.Stages) {
		t.Fatalf("Stages len: got %d, want %d", len(got.Stages), len(plan.Stages))
	}

	deps := got.Stages[0]
	if deps.Name != "deps" {
		t.Errorf("stage 0 name: got %q, want %q", deps.Name, "deps")
	}
	if deps.Steps[2].CacheMount == nil {
		t.Fatal("expected cache mount on step 2, got nil")
	}
	if deps.Steps[2].CacheMount.Target != "/root/.npm" {
		t.Errorf("cache mount target: got %q, want %q", deps.Steps[2].CacheMount.Target, "/root/.npm")
	}

	runtime := got.Stages[2]
	step := runtime.Steps[1]
	if step.CopyFrom == nil {
		t.Fatal("expected copy_from on runtime step 1, got nil")
	}
	if step.CopyFrom.Stage != "build" {
		t.Errorf("copy_from stage: got %q, want %q", step.CopyFrom.Stage, "build")
	}
	if !step.Link {
		t.Error("expected link=true on COPY --link step")
	}
}

func TestBuildPlanDockerignoreOmittedWhenEmpty(t *testing.T) {
	plan := core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name:  "build",
				From:  "golang:1.26-alpine",
				Steps: []core.Step{{Type: core.StepRun, Args: []string{"go build -o app ."}}},
			},
		},
	}

	data, err := json.Marshal(&plan)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["dockerignore"]; ok {
		t.Error("expected dockerignore to be omitted when empty, but it was present")
	}
}

func TestStepTypesCoverAllExpected(t *testing.T) {
	types := []core.StepType{
		core.StepWorkdir,
		core.StepCopy,
		core.StepCopyFrom,
		core.StepRun,
		core.StepEnv,
		core.StepArg,
		core.StepExpose,
		core.StepCmd,
		core.StepEntrypoint,
		core.StepUser,
		core.StepHealthcheck,
	}

	seen := map[core.StepType]bool{}
	for _, st := range types {
		if seen[st] {
			t.Errorf("duplicate StepType value: %d", int(st))
		}
		if st == 0 {
			t.Errorf("StepType must not be zero (uninitialized): %d", int(st))
		}
		seen[st] = true
	}
}

func TestBuildPlanSecretMountRoundTrip(t *testing.T) {
	plan := core.BuildPlan{
		Framework: "python",
		Expose:    8000,
		Stages: []core.Stage{
			{
				Name: "build",
				From: "python:3.12-slim",
				Steps: []core.Step{
					{
						Type: core.StepRun,
						Args: []string{"pip install -r requirements.txt"},
						SecretMount: &core.SecretMount{
							ID:     "pip-conf",
							Target: "/root/.pip/pip.conf",
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(&plan)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got core.BuildPlan
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sm := got.Stages[0].Steps[0].SecretMount
	if sm == nil {
		t.Fatal("expected secret_mount after round-trip, got nil")
	}
	if sm.ID != "pip-conf" {
		t.Errorf("secret mount id: got %q, want %q", sm.ID, "pip-conf")
	}
}
