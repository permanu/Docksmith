package detect

import (
	"errors"
	"strings"

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

func TestDetect_RichError_GoModNoMain(t *testing.T) {
	dir := t.TempDir()
	gomod := `module example.com/mylib

go 1.25
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	// A go.mod without main.go or cmd/ — should produce a near-miss.
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, core.ErrNotDetected) {
		t.Fatalf("expected ErrNotDetected, got %v", err)
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	if len(de.NearMisses) == 0 {
		t.Fatal("expected near-misses, got none")
	}
	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "go" && strings.Contains(nm.Found, "go.mod") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Go near-miss, got: %v", de.NearMisses)
	}

	msg := err.Error()
	if !strings.Contains(msg, "docksmith.toml") {
		t.Errorf("error should suggest docksmith.toml, got:\n%s", msg)
	}
	if !strings.Contains(msg, `runtime = "go"`) {
		t.Errorf("error should suggest go runtime in example config, got:\n%s", msg)
	}
}

func TestDetect_RichError_PythonNoFramework(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("requests==2.31.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "python" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected python near-miss, got: %v", de.NearMisses)
	}
}

func TestDetect_RichError_RustNoWebFramework(t *testing.T) {
	dir := t.TempDir()
	cargo := `[package]
name = "mylib"
version = "0.1.0"

[dependencies]
serde = "1"
`
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "rust" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected rust near-miss, got: %v", de.NearMisses)
	}
}

func TestDetect_RichError_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, core.ErrNotDetected) {
		t.Fatalf("expected ErrNotDetected, got %v", err)
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	// Empty dir should have no near-misses but still have suggestions.
	if len(de.NearMisses) != 0 {
		t.Errorf("expected no near-misses for empty dir, got %d", len(de.NearMisses))
	}

	msg := err.Error()
	if !strings.Contains(msg, "docksmith.toml") {
		t.Errorf("should suggest config file, got:\n%s", msg)
	}
	if !strings.Contains(msg, "docksmith registry search") {
		t.Errorf("should suggest registry search, got:\n%s", msg)
	}
}

func TestDetect_RichError_NodeNoFramework(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "name": "my-custom-app",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "^4.17.21"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "node" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected node near-miss, got: %v", de.NearMisses)
	}
}

func TestDetect_RichError_ElixirNoPhoenix(t *testing.T) {
	dir := t.TempDir()
	mixExs := `defmodule MyLib.MixProject do
  use Mix.Project
  def project do
    [app: :mylib, version: "0.1.0"]
  end
end
`
	if err := os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(mixExs), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "elixir" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected elixir near-miss, got: %v", de.NearMisses)
	}
}

func TestDetect_RichError_FilesCheckedPopulated(t *testing.T) {
	dir := t.TempDir()
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	if len(de.FilesChecked) == 0 {
		t.Error("FilesChecked should not be empty")
	}
}

func TestDetect_RichError_RubyNoRails(t *testing.T) {
	dir := t.TempDir()
	gemfile := `source "https://rubygems.org"
gem "puma"
gem "roda"
`
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(gemfile), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "ruby" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ruby near-miss, got: %v", de.NearMisses)
	}
}

func TestDetect_RichError_PHPNoFramework(t *testing.T) {
	dir := t.TempDir()
	composer := `{
  "name": "my/app",
  "require": {
    "php": ">=8.1",
    "monolog/monolog": "^3.0"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composer), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Detect(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var de *core.DetectionError
	if !errors.As(err, &de) {
		t.Fatalf("expected *core.DetectionError, got %T: %v", err, err)
	}

	found := false
	for _, nm := range de.NearMisses {
		if nm.Runtime == "php" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected php near-miss, got: %v", de.NearMisses)
	}
}
