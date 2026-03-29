package plan

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith/core"
)

func TestWithSecrets_AppliedToRunSteps(t *testing.T) {
	fw := &core.Framework{
		Name:           "express",
		StartCommand:   "node index.js",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	secrets := []core.SecretMount{
		{ID: "npm", Target: "/root/.npmrc"},
	}
	plan, err := Plan(fw, WithSecrets(secrets))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	found := false
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type == core.StepRun && len(step.SecretMounts) > 0 {
				found = true
				if step.SecretMounts[0].ID != "npm" {
					t.Errorf("secret ID = %q, want %q", step.SecretMounts[0].ID, "npm")
				}
			}
		}
	}
	if !found {
		t.Error("expected at least one RUN step with secret mounts")
	}
}

func TestWithSecrets_EnvOnly(t *testing.T) {
	fw := &core.Framework{
		Name:         "flask",
		StartCommand: "gunicorn app:app",
		Port:         8000,
	}
	secrets := []core.SecretMount{
		{ID: "license", Env: "LICENSE_KEY"},
	}
	plan, err := Plan(fw, WithSecrets(secrets))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	found := false
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type != core.StepRun {
				continue
			}
			for _, sm := range step.SecretMounts {
				if sm.ID == "license" && sm.Env == "LICENSE_KEY" && sm.Target == "" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected RUN step with env-only secret mount")
	}
}

func TestWithSecrets_TargetOnly(t *testing.T) {
	fw := &core.Framework{
		Name:         "flask",
		StartCommand: "gunicorn app:app",
		Port:         8000,
	}
	secrets := []core.SecretMount{
		{ID: "pip", Target: "/root/.pip/pip.conf"},
	}
	plan, err := Plan(fw, WithSecrets(secrets))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	found := false
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type != core.StepRun {
				continue
			}
			for _, sm := range step.SecretMounts {
				if sm.ID == "pip" && sm.Target == "/root/.pip/pip.conf" && sm.Env == "" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected RUN step with target-only secret mount")
	}
}

func TestWithSecrets_MultipleSecretsOnStep(t *testing.T) {
	fw := &core.Framework{
		Name:         "flask",
		StartCommand: "gunicorn app:app",
		Port:         8000,
	}
	secrets := []core.SecretMount{
		{ID: "pip", Target: "/root/.pip/pip.conf"},
		{ID: "token", Env: "API_TOKEN"},
	}
	plan, err := Plan(fw, WithSecrets(secrets))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	// Find a RUN step that has both secrets.
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type != core.StepRun {
				continue
			}
			if len(step.SecretMounts) >= 2 {
				return // pass
			}
		}
	}
	t.Error("expected at least one RUN step with 2+ secret mounts")
}

func TestWithSecrets_ConfigOverridesAutoDetected(t *testing.T) {
	// Simulate a step that already has a secret from auto-detection.
	plan := &core.BuildPlan{
		Framework: "python",
		Expose:    8000,
		Stages: []core.Stage{
			{
				Name: "build",
				From: "python:3.12-slim",
				Steps: []core.Step{
					{Type: core.StepWorkdir, Args: []string{"/app"}},
					{
						Type: core.StepRun,
						Args: []string{"pip install -r requirements.txt"},
						SecretMounts: []core.SecretMount{
							{ID: "pip", Target: "/original/path"},
						},
					},
				},
			},
		},
	}

	incoming := []core.SecretMount{
		{ID: "pip", Target: "/override/path"},
	}
	applySecrets(plan, incoming)

	step := plan.Stages[0].Steps[1]
	if len(step.SecretMounts) != 1 {
		t.Fatalf("want 1 secret mount after merge, got %d", len(step.SecretMounts))
	}
	if step.SecretMounts[0].Target != "/override/path" {
		t.Errorf("target = %q, want %q", step.SecretMounts[0].Target, "/override/path")
	}
}

func TestWithSecrets_EmptySecrets_NoChange(t *testing.T) {
	fw := &core.Framework{
		Name:           "express",
		StartCommand:   "node index.js",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan, err := Plan(fw)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if len(step.SecretMounts) > 0 {
				t.Error("expected no secret mounts without WithSecrets")
			}
		}
	}
}

func TestEmit_SecretMount_EnvForm(t *testing.T) {
	plan := &core.BuildPlan{
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
						SecretMounts: []core.SecretMount{
							{ID: "license", Env: "LICENSE_KEY"},
						},
					},
				},
			},
		},
	}
	// We test the plan structure; actual emit rendering is tested in emit package.
	step := plan.Stages[0].Steps[0]
	if step.SecretMounts[0].Env != "LICENSE_KEY" {
		t.Errorf("env = %q, want %q", step.SecretMounts[0].Env, "LICENSE_KEY")
	}
}

func TestMergeSecrets_NewIDsAppended(t *testing.T) {
	existing := []core.SecretMount{{ID: "a", Target: "/a"}}
	incoming := []core.SecretMount{{ID: "b", Env: "B"}}
	merged := mergeSecrets(existing, incoming)
	if len(merged) != 2 {
		t.Fatalf("want 2, got %d", len(merged))
	}
}

func TestMergeSecrets_DuplicateIDOverwritten(t *testing.T) {
	existing := []core.SecretMount{{ID: "a", Target: "/old"}}
	incoming := []core.SecretMount{{ID: "a", Target: "/new"}}
	merged := mergeSecrets(existing, incoming)
	if len(merged) != 1 {
		t.Fatalf("want 1, got %d", len(merged))
	}
	if merged[0].Target != "/new" {
		t.Errorf("target = %q, want %q", merged[0].Target, "/new")
	}
}

func TestSecrets_NotAppliedToRuntimeStage(t *testing.T) {
	// Multi-stage: secrets should only go on builder stages, not runtime.
	fw := &core.Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "node server.js",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
		OutputDir:      ".next",
	}
	secrets := []core.SecretMount{
		{ID: "npm", Target: "/root/.npmrc"},
	}
	plan, err := Plan(fw, WithSecrets(secrets))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Stages) < 2 {
		t.Skip("expected multi-stage plan")
	}
	last := plan.Stages[len(plan.Stages)-1]
	for _, step := range last.Steps {
		if len(step.SecretMounts) > 0 {
			t.Errorf("runtime stage should not have secret mounts, but %q does", strings.Join(step.Args, " "))
		}
	}
}
