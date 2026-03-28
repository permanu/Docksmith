package docksmith_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith"
)

func TestLoadFrameworkDefs_empty(t *testing.T) {
	dir := t.TempDir()
	defs, err := docksmith.LoadFrameworkDefs(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("want 0 defs, got %d", len(defs))
	}
}

func TestLoadFrameworkDefs_nonexistent(t *testing.T) {
	_, err := docksmith.LoadFrameworkDefs("/nonexistent/path/xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent dir, got nil")
	}
}

func TestLoadFrameworkDefs_parsesYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: testfw
runtime: node
detect:
  all:
    - file: test.config.js
plan:
  port: 9000
  stages:
    - name: build
      from: node:20-alpine
      steps:
        - workdir: /app
`
	if err := os.WriteFile(filepath.Join(dir, "testfw.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	defs, err := docksmith.LoadFrameworkDefs(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("want 1 def, got %d", len(defs))
	}
	if defs[0].Name != "testfw" {
		t.Errorf("name: want %q got %q", "testfw", defs[0].Name)
	}
	if defs[0].Plan.Port != 9000 {
		t.Errorf("port: want 9000 got %d", defs[0].Plan.Port)
	}
}

func TestLoadFrameworkDefs_skipsNonYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	validYAML := "name: fw\nruntime: go\nplan:\n  port: 8080\n  stages: []\n"
	if err := os.WriteFile(filepath.Join(dir, "fw.yaml"), []byte(validYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	defs, err := docksmith.LoadFrameworkDefs(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Errorf("want 1 def, got %d", len(defs))
	}
}

func TestLoadFrameworkDefs_rejectsNoName(t *testing.T) {
	dir := t.TempDir()
	yaml := "runtime: node\nplan:\n  port: 3000\n  stages: []\n"
	if err := os.WriteFile(filepath.Join(dir, "noname.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	defs, err := docksmith.LoadFrameworkDefs(dir)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if len(defs) != 0 {
		t.Errorf("want 0 valid defs, got %d", len(defs))
	}
}

func TestLoadAndRegisterFrameworks_missingDirIsOK(t *testing.T) {
	err := docksmith.LoadAndRegisterFrameworks("/nonexistent/frameworks")
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
}

func TestBuildPlanFromDef_noStages(t *testing.T) {
	def := &docksmith.FrameworkDef{Name: "empty"}
	_, err := docksmith.BuildPlanFromDef(def, nil)
	if err == nil {
		t.Fatal("expected error for no stages")
	}
}

func TestBuildPlanFromDef_basic(t *testing.T) {
	def := &docksmith.FrameworkDef{
		Name: "testfw",
		Plan: docksmith.PlanDef{
			Port: 8080,
			Stages: []docksmith.StageDef{
				{
					Name: "build",
					From: "golang:1.22-alpine",
					Steps: []docksmith.StepDef{
						{Workdir: "/app"},
						{Run: "go build -o app ."},
					},
				},
			},
		},
	}

	plan, err := docksmith.BuildPlanFromDef(def, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Framework != "testfw" {
		t.Errorf("framework: want %q got %q", "testfw", plan.Framework)
	}
	if plan.Expose != 8080 {
		t.Errorf("expose: want 8080 got %d", plan.Expose)
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("want 1 stage, got %d", len(plan.Stages))
	}
	if len(plan.Stages[0].Steps) != 2 {
		t.Errorf("want 2 steps, got %d", len(plan.Stages[0].Steps))
	}
}
