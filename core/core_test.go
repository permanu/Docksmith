package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestFrameworkJSONRoundTrip(t *testing.T) {
	fw := Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		OutputDir:      ".next",
		NodeVersion:    "22",
		PackageManager: "npm",
		SystemDeps:     []string{"curl", "git"},
	}

	data, err := fw.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FrameworkFromJSON(data)
	if err != nil {
		t.Fatalf("FrameworkFromJSON: %v", err)
	}

	if got.Name != fw.Name {
		t.Errorf("Name: got %q, want %q", got.Name, fw.Name)
	}
	if got.BuildCommand != fw.BuildCommand {
		t.Errorf("BuildCommand: got %q, want %q", got.BuildCommand, fw.BuildCommand)
	}
	if got.StartCommand != fw.StartCommand {
		t.Errorf("StartCommand: got %q, want %q", got.StartCommand, fw.StartCommand)
	}
	if got.Port != fw.Port {
		t.Errorf("Port: got %d, want %d", got.Port, fw.Port)
	}
	if got.OutputDir != fw.OutputDir {
		t.Errorf("OutputDir: got %q, want %q", got.OutputDir, fw.OutputDir)
	}
	if got.NodeVersion != fw.NodeVersion {
		t.Errorf("NodeVersion: got %q, want %q", got.NodeVersion, fw.NodeVersion)
	}
	if got.PackageManager != fw.PackageManager {
		t.Errorf("PackageManager: got %q, want %q", got.PackageManager, fw.PackageManager)
	}
	if len(got.SystemDeps) != len(fw.SystemDeps) {
		t.Fatalf("SystemDeps len: got %d, want %d", len(got.SystemDeps), len(fw.SystemDeps))
	}
	for i, dep := range fw.SystemDeps {
		if got.SystemDeps[i] != dep {
			t.Errorf("SystemDeps[%d]: got %q, want %q", i, got.SystemDeps[i], dep)
		}
	}
}

func TestFrameworkOmitemptyFields(t *testing.T) {
	fw := Framework{
		Name:         "go",
		BuildCommand: "go build -o app .",
		StartCommand: "./app",
		Port:         8080,
		GoVersion:    "1.24",
	}

	data, err := fw.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	omitted := []string{
		"output_dir", "node_version", "package_manager",
		"python_version", "python_pm", "system_deps",
		"php_version", "dotnet_version", "java_version",
		"deno_version", "bun_version",
	}
	for _, key := range omitted {
		if _, ok := raw[key]; ok {
			t.Errorf("expected %q to be omitted from JSON, but it was present", key)
		}
	}
}

func TestFrameworkFromJSONEmptyData(t *testing.T) {
	_, err := FrameworkFromJSON(nil)
	if err == nil {
		t.Fatal("expected error for nil data, got nil")
	}

	_, err = FrameworkFromJSON([]byte{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
}

func TestFrameworkFromJSONInvalidJSON(t *testing.T) {
	_, err := FrameworkFromJSON([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestErrorSentinelsDistinct(t *testing.T) {
	if ErrNotDetected == ErrInvalidConfig {
		t.Error("ErrNotDetected and ErrInvalidConfig must be distinct")
	}
	if ErrNotDetected == ErrInvalidPlan {
		t.Error("ErrNotDetected and ErrInvalidPlan must be distinct")
	}
	if ErrInvalidConfig == ErrInvalidPlan {
		t.Error("ErrInvalidConfig and ErrInvalidPlan must be distinct")
	}
}

func TestValidate_EmptyStages(t *testing.T) {
	plan := BuildPlan{Framework: "go", Expose: 8080}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for empty stages, got nil")
	}
	if !errors.Is(err, ErrInvalidPlan) {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_StageNoSteps(t *testing.T) {
	plan := BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []Stage{
			{Name: "build", From: "golang:1.26-alpine"},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for stage with no steps, got nil")
	}
	if !errors.Is(err, ErrInvalidPlan) {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_NonexistentFromStage(t *testing.T) {
	plan := BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []Stage{
			{
				Name:  "runtime",
				From:  "ghost-stage",
				Steps: []Step{{Type: StepCmd, Args: []string{"./app"}}},
			},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for nonexistent from stage, got nil")
	}
	if !errors.Is(err, ErrInvalidPlan) {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_ValidFromBaseImage(t *testing.T) {
	plan := BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []Stage{
			{
				Name:  "build",
				From:  "golang:1.26-alpine",
				Steps: []Step{{Type: StepRun, Args: []string{"go build -o app ."}}},
			},
			{
				Name:  "runtime",
				From:  "build",
				Steps: []Step{{Type: StepCmd, Args: []string{"./app"}}},
			},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected error for valid plan: %v", err)
	}
}

func TestValidate_PortZero_NonStatic(t *testing.T) {
	plan := BuildPlan{
		Framework: "express",
		Expose:    0,
		Stages: []Stage{
			{
				Name:  "runtime",
				From:  "node:22-alpine",
				Steps: []Step{{Type: StepCmd, Args: []string{"node", "index.js"}}},
			},
		},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for port <= 0 on non-static framework, got nil")
	}
	if !errors.Is(err, ErrInvalidPlan) {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}

func TestValidate_PortZero_Static(t *testing.T) {
	plan := BuildPlan{
		Framework: "static",
		Expose:    0,
		Stages: []Stage{
			{
				Name:  "runtime",
				From:  "nginx:alpine",
				Steps: []Step{{Type: StepCopy, Args: []string{"dist", "/usr/share/nginx/html"}}},
			},
		},
	}
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected error for static site with port=0: %v", err)
	}
}

func TestValidate_SecretMount_Valid(t *testing.T) {
	plan := BuildPlan{
		Framework: "python",
		Expose:    8000,
		Stages: []Stage{
			{
				Name: "build",
				From: "python:3.12-slim",
				Steps: []Step{
					{
						Type: StepRun,
						Args: []string{"pip install -r requirements.txt"},
						SecretMounts: []SecretMount{
							{
								ID:     "pip-conf",
								Target: "/root/.pip/pip.conf",
							},
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

func TestIsImageRef(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"golang:1.26-alpine", true},
		{"mcr.microsoft.com/dotnet/sdk:8.0", true},
		{"build", false},
		{"runtime", false},
	}
	for _, tt := range tests {
		if got := IsImageRef(tt.s); got != tt.want {
			t.Errorf("IsImageRef(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}
