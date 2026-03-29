package detect

import (
	"errors"
	"github.com/permanu/docksmith/core"
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Dockerfile(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "with-dockerfile")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "dockerfile" {
		t.Errorf("Name = %q, want %q", fw.Name, "dockerfile")
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
}

func TestDetect_EmptyDir_ReturnsError(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "empty-dir")
	fw, err := Detect(dir)
	if err == nil {
		t.Fatalf("expected error for empty dir, got framework %q", fw.Name)
	}
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("error = %v, want core.ErrNotDetected", err)
	}
}

func TestDetect_StaticSite_WithHTML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>hello</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw.Name != "static" {
		t.Errorf("Name = %q, want %q", fw.Name, "static")
	}
}

func TestDetect_BinaryOnly_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.exe"), []byte{0x4d, 0x5a}, 0o644); err != nil {
		t.Fatal(err)
	}
	fw, err := Detect(dir)
	if err == nil {
		t.Fatalf("expected error for binary-only dir, got framework %q", fw.Name)
	}
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("error = %v, want core.ErrNotDetected", err)
	}
}

func TestDetect_PriorityOrder(t *testing.T) {
	dir := t.TempDir()

	// reset registry after test
	orig := detectors
	t.Cleanup(func() { detectors = orig })

	var order []string
	detectors = []NamedDetector{
		{"first", func(d string) *core.Framework {
			order = append(order, "first")
			return &core.Framework{Name: "first", Port: 1}
		}},
		{"second", func(d string) *core.Framework {
			order = append(order, "second")
			return &core.Framework{Name: "second", Port: 2}
		}},
	}

	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw.Name != "first" {
		t.Errorf("Name = %q, want %q", fw.Name, "first")
	}
	if len(order) != 1 || order[0] != "first" {
		t.Errorf("ran detectors %v, want only [first]", order)
	}
}

func TestRegisterDetector_Prepends(t *testing.T) {
	orig := detectors
	t.Cleanup(func() { detectors = orig })

	detectors = []NamedDetector{
		{"existing", func(d string) *core.Framework { return nil }},
	}

	RegisterDetector("new", func(d string) *core.Framework { return nil })

	if len(detectors) != 2 {
		t.Fatalf("len = %d, want 2", len(detectors))
	}
	if detectors[0].Name != "new" {
		t.Errorf("detectors[0].Name = %q, want %q", detectors[0].Name, "new")
	}
	if detectors[1].Name != "existing" {
		t.Errorf("detectors[1].Name = %q, want %q", detectors[1].Name, "existing")
	}
}

func TestRegisterDetectorBefore(t *testing.T) {
	orig := detectors
	t.Cleanup(func() { detectors = orig })

	detectors = []NamedDetector{
		{"a", func(d string) *core.Framework { return nil }},
		{"b", func(d string) *core.Framework { return nil }},
	}

	RegisterDetectorBefore("b", "inserted", func(d string) *core.Framework { return nil })

	if len(detectors) != 3 {
		t.Fatalf("len = %d, want 3", len(detectors))
	}
	names := make([]string, len(detectors))
	for i, nd := range detectors {
		names[i] = nd.Name
	}
	want := []string{"a", "inserted", "b"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("detectors[%d].Name = %q, want %q", i, names[i], w)
		}
	}
}

func TestRegisterDetectorBefore_NotFound_Prepends(t *testing.T) {
	orig := detectors
	t.Cleanup(func() { detectors = orig })

	detectors = []NamedDetector{
		{"x", func(d string) *core.Framework { return nil }},
	}

	RegisterDetectorBefore("nonexistent", "fallback", func(d string) *core.Framework { return nil })

	if detectors[0].Name != "fallback" {
		t.Errorf("detectors[0].Name = %q, want %q", detectors[0].Name, "fallback")
	}
}

func TestDetectWithOptions_AcceptsConfigFileNames(t *testing.T) {
	dir := t.TempDir()
	// Write a valid config so DetectWithOptions returns a framework, not a parse error.
	cfg := "runtime = \"node\"\n\n[start]\ncommand = \"node index.js\"\n"
	if err := os.WriteFile(filepath.Join(dir, "myapp.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	fw, err := DetectWithOptions(dir, DetectOptions{ConfigFileNames: []string{"myapp.toml"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name == "" {
		t.Error("Name is empty, want non-empty")
	}
}

func TestDetectWithOptions_BrokenConfigReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docksmith.toml"), []byte("[broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Error("expected error for broken config, got nil")
	}
}
