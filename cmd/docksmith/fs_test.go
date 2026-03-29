package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteProjectFileRejectsSymlinkDestination(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "docksmith.toml")); err != nil {
		t.Fatal(err)
	}

	if _, err := writeProjectFile(dir, "docksmith.toml", []byte("runtime = \"node\"\n"), 0o644, true); err == nil {
		t.Fatal("expected symlink destination to be rejected")
	}

	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "outside" {
		t.Fatalf("outside file was modified: %q", got)
	}
}

func TestProjectFilePathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if _, err := projectFilePath(dir, "../escape.txt"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}
