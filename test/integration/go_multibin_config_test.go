package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith"
)

func TestLoadConfig_MultiBinary_Fixture(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "go-multibin"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config")
	}
	if len(cfg.Build.Binaries) != 2 {
		t.Fatalf("expected 2 binaries, got %d", len(cfg.Build.Binaries))
	}
	if cfg.Build.Binaries[0].Name != "server" {
		t.Errorf("binaries[0].name = %q, want %q", cfg.Build.Binaries[0].Name, "server")
	}
	if cfg.Build.Binaries[0].Path != "./cmd/server" {
		t.Errorf("binaries[0].path = %q, want %q", cfg.Build.Binaries[0].Path, "./cmd/server")
	}
	if cfg.Build.Binaries[1].Name != "worker" {
		t.Errorf("binaries[1].name = %q, want %q", cfg.Build.Binaries[1].Name, "worker")
	}
}

func TestValidateBinaries_InvalidName(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./server"
[[build.binaries]]
name = "Bad Name"
path = "./cmd/server"
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for invalid binary name, got nil")
	}
}

func TestValidateBinaries_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./server"
[[build.binaries]]
name = "server"
path = ""
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for empty binary path, got nil")
	}
}

func TestValidateBinaries_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./server"
[[build.binaries]]
name = "server"
path = "../evil"
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for path traversal, got nil")
	}
}

func TestValidateBinaries_PathMustStartWithDotSlashOrSlash(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./server"
[[build.binaries]]
name = "server"
path = "cmd/server"
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for path not starting with ./ or /, got nil")
	}
}

func TestValidateBinaries_DuplicateName(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./server"
[[build.binaries]]
name = "server"
path = "./cmd/server"
[[build.binaries]]
name = "server"
path = "./cmd/other"
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for duplicate binary name, got nil")
	}
}

func TestValidateBinaries_InvalidOutputName(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./server"
[[build.binaries]]
name = "server"
path = "./cmd/server"
output_name = "Bad Output"
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for invalid output_name, got nil")
	}
}
