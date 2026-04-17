package plan

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateSBOMEmptyDir(t *testing.T) {
	_, err := GenerateSBOM(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty contextDir, got nil")
	}
}

func TestGenerateSBOMSyftMissing(t *testing.T) {
	// When syft is NOT on PATH, GenerateSBOM must return (nil, nil).
	// We simulate by overriding PATH to an empty temp dir.
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })

	empty := t.TempDir()
	if err := os.Setenv("PATH", empty); err != nil {
		t.Fatalf("os.Setenv: %v", err)
	}

	raw, err := GenerateSBOM(context.Background(), t.TempDir())
	if err != nil {
		t.Errorf("expected nil error when syft missing, got %v", err)
	}
	if raw != nil {
		t.Errorf("expected nil SBOM when syft missing, got %s", raw)
	}
}

func TestGenerateSBOMLive(t *testing.T) {
	if _, err := exec.LookPath("syft"); err != nil {
		t.Skip("syft not installed on PATH; skipping live SBOM test")
	}

	// Seed a minimal project so syft has something to scan.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	raw, err := GenerateSBOM(context.Background(), dir)
	if err != nil {
		t.Fatalf("GenerateSBOM: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil SBOM from syft")
	}
	if !json.Valid(raw) {
		t.Fatalf("SBOM is not valid JSON: %s", raw)
	}

	// CycloneDX JSON carries a "bomFormat":"CycloneDX" field at the root.
	var doc struct {
		BomFormat   string `json:"bomFormat"`
		SpecVersion string `json:"specVersion"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if doc.BomFormat != "CycloneDX" {
		t.Errorf("bomFormat = %q, want CycloneDX", doc.BomFormat)
	}
	if doc.SpecVersion == "" {
		t.Errorf("specVersion empty")
	}
}
