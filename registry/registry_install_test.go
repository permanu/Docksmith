package registry_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith/registry"
)

func TestInstallFramework_writesFile(t *testing.T) {
	yamlContent := "name: gleam\nruntime: erlang\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yamlContent)
	}))
	defer srv.Close()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	h := sha256.Sum256([]byte(yamlContent))
	checksum := hex.EncodeToString(h[:])

	entry := registry.Entry{
		Name:    "gleam",
		Version: "1.2.0",
		URL:     srv.URL + "/gleam.yaml",
		SHA256:  checksum,
	}

	dest, err := registry.InstallFramework(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(homeDir, ".docksmith", "frameworks", "gleam.yaml")
	if dest != want {
		t.Errorf("dest: want %q got %q", want, dest)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if string(data) != yamlContent {
		t.Errorf("content: want %q got %q", yamlContent, string(data))
	}
}

func TestInstallFramework_sha256Mismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "actual content")
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())

	entry := registry.Entry{
		Name:   "badfw",
		URL:    srv.URL + "/bad.yaml",
		SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	_, err := registry.InstallFramework(entry)
	if err == nil {
		t.Fatal("expected error for sha256 mismatch")
	}
}

func TestInstallFramework_noURL(t *testing.T) {
	_, err := registry.InstallFramework(registry.Entry{Name: "broken"})
	if err == nil {
		t.Fatal("expected error for entry with no URL")
	}
}

func TestInstallFramework_missingSHA256(t *testing.T) {
	_, err := registry.InstallFramework(registry.Entry{Name: "nosha", URL: "https://example.com/fw.yaml"})
	if err == nil {
		t.Fatal("expected error for missing sha256")
	}
}

func TestInstallFramework_pathTraversal(t *testing.T) {
	tests := []string{
		"../etc/passwd",
		"../../evil",
		"foo/bar",
		`foo\bar`,
		"..",
		".",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			entry := registry.Entry{
				Name:   name,
				URL:    "http://example.com/fw.yaml",
				SHA256: "abc",
			}
			_, err := registry.InstallFramework(entry)
			if err == nil {
				t.Errorf("expected error for malicious name %q", name)
			}
		})
	}
}

func TestInstallFramework_rejectsHTTP(t *testing.T) {
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	t.Setenv("HOME", t.TempDir())

	entry := registry.Entry{
		Name:   "evil",
		URL:    "http://example.com/evil.yaml",
		SHA256: "abc",
	}
	_, err := registry.InstallFramework(entry)
	if err == nil {
		t.Fatal("expected error for non-HTTPS download URL")
	}
}
