package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith"
)

func TestValidateArchitectures_InvalidArch(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./agent"
[[build.binaries]]
name = "agent"
path = "./cmd/agent"
architectures = ["amd64", "mips"]
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for unsupported arch, got nil")
	}
}

func TestValidateArchitectures_DuplicateArch(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./agent"
[[build.binaries]]
name = "agent"
path = "./cmd/agent"
architectures = ["amd64", "amd64"]
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	_, err := docksmith.LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for duplicate arch entry, got nil")
	}
}

func TestValidateArchitectures_EmptyListAllowed(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./agent"
[[build.binaries]]
name = "agent"
path = "./cmd/agent"
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	cfg, err := docksmith.LoadConfig(dir)
	if err != nil {
		t.Fatalf("empty Architectures should be allowed: %v", err)
	}
	if len(cfg.Build.Binaries[0].Architectures) != 0 {
		t.Errorf("expected empty Architectures, got %v", cfg.Build.Binaries[0].Architectures)
	}
}

func TestValidateArchitectures_AllValidArchsAccepted(t *testing.T) {
	dir := t.TempDir()
	content := `runtime = "go"
[start]
command = "./agent"
[[build.binaries]]
name = "agent"
path = "./cmd/agent"
architectures = ["amd64", "arm64", "arm", "386", "riscv64"]
`
	mustWriteFile(t, filepath.Join(dir, "docksmith.toml"), content)
	cfg, err := docksmith.LoadConfig(dir)
	if err != nil {
		t.Fatalf("all valid archs should be accepted: %v", err)
	}
	if len(cfg.Build.Binaries[0].Architectures) != 5 {
		t.Errorf("expected 5 architectures, got %d", len(cfg.Build.Binaries[0].Architectures))
	}
}

func TestLoadConfig_CrossArch_Fixture(t *testing.T) {
	cfg, err := docksmith.LoadConfig(filepath.Join("../../testdata", "fixtures", "go-multibin-crossarch"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config")
	}
	if len(cfg.Build.Binaries) != 2 {
		t.Fatalf("expected 2 binaries, got %d", len(cfg.Build.Binaries))
	}
	agent := cfg.Build.Binaries[0]
	if agent.Name != "permanu-agent" {
		t.Errorf("binaries[0].name = %q, want %q", agent.Name, "permanu-agent")
	}
	if len(agent.Architectures) != 2 {
		t.Fatalf("expected 2 architectures for agent, got %d", len(agent.Architectures))
	}
	if agent.Architectures[0] != "amd64" || agent.Architectures[1] != "arm64" {
		t.Errorf("unexpected architectures: %v", agent.Architectures)
	}
	// Second binary has no Architectures.
	if len(cfg.Build.Binaries[1].Architectures) != 0 {
		t.Errorf("worker should have no architectures, got %v", cfg.Build.Binaries[1].Architectures)
	}
}
