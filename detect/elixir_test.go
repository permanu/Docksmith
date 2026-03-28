package detect

import (
	"path/filepath"
	"testing"
)

func TestDetectElixirPhoenix(t *testing.T) {
	fw := detectElixirPhoenix(filepath.Join("testdata", "fixtures", "elixir-phoenix"))
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "elixir-phoenix" {
		t.Errorf("Name = %q, want elixir-phoenix", fw.Name)
	}
	if fw.Port != 4000 {
		t.Errorf("Port = %d, want 4000", fw.Port)
	}
	if fw.BuildCommand != "mix deps.get && mix compile" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
	if fw.StartCommand != "mix phx.server" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectElixirPhoenix_NoMatch(t *testing.T) {
	if fw := detectElixirPhoenix(filepath.Join("testdata", "fixtures", "elixir-no-phoenix")); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectElixirPhoenix_MissingMixExs(t *testing.T) {
	if fw := detectElixirPhoenix(filepath.Join("testdata", "fixtures", "empty-dir")); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectElixirPhoenix_ViaDetect(t *testing.T) {
	fw, err := Detect(filepath.Join("testdata", "fixtures", "elixir-phoenix"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "elixir-phoenix" {
		t.Errorf("Name = %q, want elixir-phoenix", fw.Name)
	}
}
