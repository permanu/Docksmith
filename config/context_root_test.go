package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateContextRoot_ValidAncestor(t *testing.T) {
	tmp := t.TempDir()
	appDir := filepath.Join(tmp, "apps", "frontend")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rel, err := ValidateContextRoot(tmp, appDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != "apps/frontend" {
		t.Errorf("rel = %q, want %q", rel, "apps/frontend")
	}
}

func TestValidateContextRoot_SameDir(t *testing.T) {
	tmp := t.TempDir()
	rel, err := ValidateContextRoot(tmp, tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != "" {
		t.Errorf("rel = %q, want empty string for same dir", rel)
	}
}

func TestValidateContextRoot_Empty(t *testing.T) {
	rel, err := ValidateContextRoot("", "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != "" {
		t.Errorf("rel = %q, want empty for empty context root", rel)
	}
}

func TestValidateContextRoot_NotAncestor(t *testing.T) {
	tmp := t.TempDir()
	other := t.TempDir()
	_, err := ValidateContextRoot(tmp, other)
	if err == nil {
		t.Fatal("expected error for non-ancestor context root, got nil")
	}
}

func TestValidateContextRoot_PathTraversal(t *testing.T) {
	_, err := ValidateContextRoot("/some/../etc", "/some/app")
	if err == nil {
		t.Fatal("expected error for path traversal in context root, got nil")
	}
}

func TestValidateContextRoot_AppDirTraversal(t *testing.T) {
	_, err := ValidateContextRoot("/some/root", "/some/root/../etc")
	if err == nil {
		t.Fatal("expected error for path traversal in app dir, got nil")
	}
}
