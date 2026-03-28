package plan

import (
	"github.com/permanu/docksmith/core"
	"errors"
	"testing"
)

func TestValidate_EmptyStages(t *testing.T) {
	plan := core.BuildPlan{Framework: "go", Expose: 8080}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for empty stages, got nil")
	}
	if !errors.Is(err, core.ErrInvalidPlan) {
		t.Errorf("expected core.ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_StageNoSteps(t *testing.T) {
	plan := core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{Name: "build", From: "golang:1.26-alpine"},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for stage with no steps, got nil")
	}
	if !errors.Is(err, core.ErrInvalidPlan) {
		t.Errorf("expected core.ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_NonexistentFromStage(t *testing.T) {
	plan := core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name:  "runtime",
				From:  "ghost-stage",
				Steps: []core.Step{{Type: core.StepCmd, Args: []string{"./app"}}},
			},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for nonexistent from stage, got nil")
	}
	if !errors.Is(err, core.ErrInvalidPlan) {
		t.Errorf("expected core.ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_ValidFromBaseImage(t *testing.T) {
	plan := core.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []core.Stage{
			{
				Name:  "build",
				From:  "golang:1.26-alpine",
				Steps: []core.Step{{Type: core.StepRun, Args: []string{"go build -o app ."}}},
			},
			{
				Name:  "runtime",
				From:  "build",
				Steps: []core.Step{{Type: core.StepCmd, Args: []string{"./app"}}},
			},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected error for valid plan: %v", err)
	}
}

func TestValidate_PortZero_NonStatic(t *testing.T) {
	plan := core.BuildPlan{
		Framework: "express",
		Expose:    0,
		Stages: []core.Stage{
			{
				Name:  "runtime",
				From:  "node:22-alpine",
				Steps: []core.Step{{Type: core.StepCmd, Args: []string{"node", "index.js"}}},
			},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for port <= 0 on non-static framework, got nil")
	}
	if !errors.Is(err, core.ErrInvalidPlan) {
		t.Errorf("expected core.ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_PortZero_Static(t *testing.T) {
	// Static sites served by a proxy don't need an expose port.
	plan := core.BuildPlan{
		Framework: "static",
		Expose:    0,
		Stages: []core.Stage{
			{
				Name:  "runtime",
				From:  "nginx:alpine",
				Steps: []core.Step{{Type: core.StepCopy, Args: []string{"dist", "/usr/share/nginx/html"}}},
			},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected error for static site with port=0: %v", err)
	}
}

func TestValidate_SecretMount_Valid(t *testing.T) {
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
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected error for plan with secret mount: %v", err)
	}
}
