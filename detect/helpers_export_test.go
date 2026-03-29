package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith/detect"
)

func TestHasFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "present.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		dir  string
		file string
		want bool
	}{
		{"existing file", dir, "present.txt", true},
		{"missing file", dir, "absent.txt", false},
		{"empty filename", dir, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detect.HasFile(tt.dir, tt.file); got != tt.want {
				t.Errorf("HasFile(%q, %q) = %v, want %v", tt.dir, tt.file, got, tt.want)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", file, true},
		{"missing file", filepath.Join(dir, "nope.txt"), false},
		{"directory returns false", dir, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detect.FileExists(tt.path); got != tt.want {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileContains(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	full := write("full.txt", "hello world\nfoo bar\n")
	empty := write("empty.txt", "")

	tests := []struct {
		name   string
		path   string
		substr string
		want   bool
	}{
		{"exact match", full, "hello world", true},
		{"partial match", full, "foo", true},
		{"no match", full, "missing", false},
		{"empty file", empty, "anything", false},
		{"missing file", filepath.Join(dir, "ghost.txt"), "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detect.FileContains(tt.path, tt.substr); got != tt.want {
				t.Errorf("FileContains(%q, %q) = %v, want %v", tt.path, tt.substr, got, tt.want)
			}
		})
	}
}

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{">=3.9,<4", "3.9"},
		{"^18.0.0", "18"},
		{"~3.11", "3.11"},
		{"22", "22"},
		{"", ""},
		{"lts/*", ""},
		{"stable", ""},
		{"node", ""},
		{"v18.0.0", "18.0.0"},
		{"  v3.11  ", "3.11"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := detect.ParseVersionString(tt.in); got != tt.want {
				t.Errorf("ParseVersionString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractMajorVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"18.0.0", "18"},
		{"3.9.1", "3.9"},
		{"22", "22"},
		{"", ""},
		{"*", ""},
		{">=18.0.0", "18"},
		{"^3.11.4", "3.11"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := detect.ExtractMajorVersion(tt.in); got != tt.want {
				t.Errorf("ExtractMajorVersion(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
