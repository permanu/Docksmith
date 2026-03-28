package yamldef_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith/yamldef"
)

func TestLoadFrameworkDefs_empty(t *testing.T) {
	dir := t.TempDir()
	defs, err := yamldef.LoadFrameworkDefs(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("want 0 defs, got %d", len(defs))
	}
}

func TestLoadFrameworkDefs_nonexistent(t *testing.T) {
	_, err := yamldef.LoadFrameworkDefs("/nonexistent/path/xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent dir, got nil")
	}
}

func TestLoadFrameworkDefs_parsesYAML(t *testing.T) {
	dir := t.TempDir()
	yml := `
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
	if err := os.WriteFile(filepath.Join(dir, "testfw.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}

	defs, err := yamldef.LoadFrameworkDefs(dir)
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

	defs, err := yamldef.LoadFrameworkDefs(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Errorf("want 1 def, got %d", len(defs))
	}
}

func TestLoadFrameworkDefs_rejectsNoName(t *testing.T) {
	dir := t.TempDir()
	yml := "runtime: node\nplan:\n  port: 3000\n  stages: []\n"
	if err := os.WriteFile(filepath.Join(dir, "noname.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	defs, err := yamldef.LoadFrameworkDefs(dir)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
	if len(defs) != 0 {
		t.Errorf("want 0 valid defs, got %d", len(defs))
	}
}
