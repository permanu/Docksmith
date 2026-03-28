package docksmith

import (
	"strings"
	"testing"
)

func laravelFramework() *Framework {
	return &Framework{
		Name:         "laravel",
		BuildCommand: "composer install --no-dev --optimize-autoloader",
		StartCommand: "php artisan serve --host=0.0.0.0 --port=8000",
		Port:         8000,
		PHPVersion:   "8.3",
	}
}

func wordpressFramework() *Framework {
	return &Framework{
		Name:       "wordpress",
		StartCommand: "apache2-foreground",
		Port:       80,
		PHPVersion: "8.3",
	}
}

func symfonyFramework() *Framework {
	return &Framework{
		Name:         "symfony",
		BuildCommand: "composer install --no-dev --optimize-autoloader",
		StartCommand: "apache2-foreground",
		Port:         80,
		PHPVersion:   "8.3",
	}
}

func mustPlanPHP(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planPHP(fw)
	if err != nil {
		t.Fatalf("planPHP: %v", err)
	}
	return plan
}

// --- WordPress ---

func TestPlanPHP_WordPress_SingleStage(t *testing.T) {
	plan := mustPlanPHP(t, wordpressFramework())
	if len(plan.Stages) != 1 {
		t.Fatalf("wordpress: expected 1 stage, got %d", len(plan.Stages))
	}
}

func TestPlanPHP_WordPress_BaseImage(t *testing.T) {
	plan := mustPlanPHP(t, wordpressFramework())
	from := plan.Stages[0].From
	if !strings.HasPrefix(from, "wordpress:php") {
		t.Errorf("wordpress image: got %q, want wordpress:php... prefix", from)
	}
}

func TestPlanPHP_WordPress_ValidatesOK(t *testing.T) {
	plan := mustPlanPHP(t, wordpressFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

// --- Laravel ---

func TestPlanPHP_Laravel_TwoStages(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("laravel: expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanPHP_Laravel_BuilderBaseImage(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	want := ResolveDockerTag("php", "8.3")
	if plan.Stages[0].From != want {
		t.Errorf("laravel builder from: got %q, want %q", plan.Stages[0].From, want)
	}
}

func TestPlanPHP_Laravel_BuilderRunsComposer(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "composer install") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("laravel builder should run composer install")
	}
}

func TestPlanPHP_Laravel_BuilderHasCacheMount(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil {
			found = true
		}
	}
	if !found {
		t.Error("laravel builder should have a composer cache mount")
	}
}

func TestPlanPHP_Laravel_RuntimeCopiesFromBuilder(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCopyFrom && step.CopyFrom != nil && step.CopyFrom.Stage == "builder" {
			found = true
		}
	}
	if !found {
		t.Error("laravel runtime should copy from builder")
	}
}

func TestPlanPHP_Laravel_ValidatesOK(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanPHP_Laravel_ExposedPort(t *testing.T) {
	plan := mustPlanPHP(t, laravelFramework())
	if plan.Expose != 8000 {
		t.Errorf("expose: got %d, want 8000", plan.Expose)
	}
}

// --- Symfony ---

func TestPlanPHP_Symfony_SingleStage(t *testing.T) {
	plan := mustPlanPHP(t, symfonyFramework())
	if len(plan.Stages) != 1 {
		t.Fatalf("symfony: expected 1 stage, got %d", len(plan.Stages))
	}
}

func TestPlanPHP_Symfony_BaseImage(t *testing.T) {
	plan := mustPlanPHP(t, symfonyFramework())
	want := ResolveDockerTag("php-apache", "8.3")
	if plan.Stages[0].From != want {
		t.Errorf("symfony from: got %q, want %q", plan.Stages[0].From, want)
	}
}

func TestPlanPHP_Symfony_RunsComposer(t *testing.T) {
	plan := mustPlanPHP(t, symfonyFramework())
	stage := plan.Stages[0]
	found := false
	for _, step := range stage.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "composer install") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("symfony stage should run composer install")
	}
}

func TestPlanPHP_Symfony_ValidatesOK(t *testing.T) {
	plan := mustPlanPHP(t, symfonyFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanPHP_DefaultPHPVersion(t *testing.T) {
	fw := &Framework{Name: "laravel", Port: 8000, StartCommand: "php artisan serve"}
	plan := mustPlanPHP(t, fw)
	// Default PHP version should be used — image should contain php
	if !strings.Contains(plan.Stages[0].From, "php:") {
		t.Errorf("expected php image, got %q", plan.Stages[0].From)
	}
}
