package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith/registry"
)

func TestInstallSuccess(t *testing.T) {
	yamlContent := "name: gleam\nruntime: erlang\n"
	h := sha256.Sum256([]byte(yamlContent))
	checksum := hex.EncodeToString(h[:])

	idx := registry.Index{
		Version: 1,
		Frameworks: map[string]registry.Entry{
			"gleam": {
				Version: "1.2.0",
				Runtime: "erlang",
				SHA256:  checksum,
			},
		},
	}

	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, _ *http.Request) {
		patched := idx
		e := patched.Frameworks["gleam"]
		e.URL = srv.URL + "/gleam.yaml"
		patched.Frameworks["gleam"] = e
		data, _ := json.Marshal(patched)
		w.Write(data)
	})
	mux.HandleFunc("/gleam.yaml", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, yamlContent)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)

	var out, errw bytes.Buffer
	err := execInstall(srv.URL+"/index.json", false, []string{"gleam"}, &out, &errw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "Installed to") {
		t.Errorf("expected 'Installed to' message, got: %s", out.String())
	}

	dest := filepath.Join(home, ".docksmith", "frameworks", "gleam.yaml")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if string(data) != yamlContent {
		t.Errorf("content mismatch: got %q", string(data))
	}
}

func TestInstallNotFound(t *testing.T) {
	srv := serveIndex(t, testIndex)
	t.Setenv("HOME", t.TempDir())

	var out, errw bytes.Buffer
	err := execInstall(srv.URL, false, []string{"nonexistent"}, &out, &errw)
	if err == nil {
		t.Fatal("expected error for unknown framework")
	}

	nf, ok := err.(*errNotFound)
	if !ok {
		t.Fatalf("expected errNotFound, got %T: %v", err, err)
	}
	if nf.name != "nonexistent" {
		t.Errorf("wrong name in error: %s", nf.name)
	}
}

func TestInstallNotFound_suggestions(t *testing.T) {
	srv := serveIndex(t, testIndex)
	t.Setenv("HOME", t.TempDir())

	var out, errw bytes.Buffer
	err := execInstall(srv.URL, false, []string{"go"}, &out, &errw)
	if err == nil {
		t.Fatal("expected error for partial name")
	}

	if !strings.Contains(err.Error(), "did you mean") {
		t.Errorf("expected suggestions in error, got: %s", err.Error())
	}
}

func TestInstallAlreadyInstalled(t *testing.T) {
	srv := serveIndex(t, testIndex)
	home := t.TempDir()
	t.Setenv("HOME", home)

	fwDir := filepath.Join(home, ".docksmith", "frameworks")
	os.MkdirAll(fwDir, 0o755)
	os.WriteFile(filepath.Join(fwDir, "gleam.yaml"), []byte("old"), 0o644)

	var out, errw bytes.Buffer
	err := execInstall(srv.URL, false, []string{"gleam"}, &out, &errw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(errw.String(), "already installed") {
		t.Errorf("expected 'already installed' message, got: %s", errw.String())
	}

	data, _ := os.ReadFile(filepath.Join(fwDir, "gleam.yaml"))
	if string(data) != "old" {
		t.Error("file should not have been overwritten")
	}
}
