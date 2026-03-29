package docksmith

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/registry"
)

func init() {
	registry.SetAllowInsecureHTTP(true)
}

const testYAMLDef = `name: test-elixir
runtime: elixir
detect:
  all:
    - file: mix.exs
plan:
  port: 4000
`

func serveRegistry(t *testing.T, yamlContent string) *httptest.Server {
	t.Helper()

	h := sha256.Sum256([]byte(yamlContent))
	checksum := hex.EncodeToString(h[:])

	mux := http.NewServeMux()

	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		idx := registry.Index{
			Version: 1,
			Frameworks: map[string]registry.Entry{
				"test-elixir": {
					Version:     "1.0.0",
					Description: "Test elixir framework",
					Runtime:     "elixir",
					Author:      "test",
					SHA256:      checksum,
					URL:         "http://" + r.Host + "/test-elixir.yaml",
				},
			},
		}
		data, _ := json.Marshal(idx)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("/test-elixir.yaml", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yamlContent)
	})

	return httptest.NewServer(mux)
}

func withEmptyDetectors(t *testing.T) {
	t.Helper()
	orig := detect.GetDetectors()
	detect.SetDetectors(nil)
	t.Cleanup(func() { detect.SetDetectors(orig) })
}

func TestAutoFetch_RegistryMatch(t *testing.T) {
	srv := serveRegistry(t, testYAMLDef)
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	withEmptyDetectors(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(`defmodule MyApp do end`), 0o644)

	afo := AutoFetchOptions{RegistryURL: srv.URL + "/index.json"}
	opts := detect.DetectOptions{AutoFetch: NewAutoFetch(afo)}

	fw, err := detect.DetectWithOptions(dir, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("expected framework, got nil")
	}
	if fw.Name != "test-elixir" {
		t.Errorf("Name = %q, want %q", fw.Name, "test-elixir")
	}
	if fw.Port != 4000 {
		t.Errorf("Port = %d, want 4000", fw.Port)
	}
}

func TestAutoFetch_NoRegistryMatch(t *testing.T) {
	idx := registry.Index{Version: 1, Frameworks: map[string]registry.Entry{}}
	data, _ := json.Marshal(idx)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	withEmptyDetectors(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(`defmodule MyApp do end`), 0o644)

	afo := AutoFetchOptions{RegistryURL: srv.URL}
	opts := detect.DetectOptions{AutoFetch: NewAutoFetch(afo)}

	_, err := detect.DetectWithOptions(dir, opts)
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("expected ErrNotDetected, got %v", err)
	}
}

func TestAutoFetch_NetworkFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	withEmptyDetectors(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(`defmodule MyApp do end`), 0o644)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	afo := AutoFetchOptions{RegistryURL: srv.URL + "/index.json"}
	opts := detect.DetectOptions{AutoFetch: NewAutoFetch(afo)}

	_, err := detect.DetectWithOptions(dir, opts)
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("expected ErrNotDetected, got %v", err)
	}
}

func TestAutoFetch_Disabled_ShowsHint(t *testing.T) {
	withEmptyDetectors(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(`defmodule MyApp do end`), 0o644)

	var hintMsg string
	opts := detect.DetectOptions{
		Hint: func(msg string) { hintMsg = msg },
	}

	detect.DetectWithOptions(dir, opts)

	if hintMsg == "" {
		t.Fatal("expected hint message, got empty")
	}
	if !strings.Contains(hintMsg, "docksmith registry search elixir") {
		t.Errorf("hint = %q, want it to contain registry search suggestion", hintMsg)
	}
}

func TestAutoFetch_AlreadyInstalled_NoReDownload(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	withEmptyDetectors(t)

	fwDir := filepath.Join(homeDir, ".docksmith", "frameworks")
	os.MkdirAll(fwDir, 0o755)
	os.WriteFile(filepath.Join(fwDir, "test-elixir.yaml"), []byte(testYAMLDef), 0o644)

	downloadCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-elixir.yaml" {
			downloadCalls++
		}
		// Serve index with test-elixir entry.
		idx := registry.Index{
			Version: 1,
			Frameworks: map[string]registry.Entry{
				"test-elixir": {
					Version: "1.0.0", Runtime: "elixir",
					URL: "http://" + r.Host + "/test-elixir.yaml", SHA256: "abc",
				},
			},
		}
		data, _ := json.Marshal(idx)
		w.Write(data)
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(`defmodule MyApp do end`), 0o644)

	afo := AutoFetchOptions{RegistryURL: srv.URL + "/index.json"}
	opts := detect.DetectOptions{AutoFetch: NewAutoFetch(afo)}

	detect.DetectWithOptions(dir, opts)

	if downloadCalls > 0 {
		t.Errorf("framework was re-downloaded %d time(s), want 0", downloadCalls)
	}
}

func TestAutoFetch_InteractiveDenied(t *testing.T) {
	srv := serveRegistry(t, testYAMLDef)
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	withEmptyDetectors(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(`defmodule MyApp do end`), 0o644)

	afo := AutoFetchOptions{
		RegistryURL: srv.URL + "/index.json",
		Interactive: true,
		ConfirmInstall: func(name, desc string) bool {
			return false
		},
	}
	opts := detect.DetectOptions{AutoFetch: NewAutoFetch(afo)}

	_, err := detect.DetectWithOptions(dir, opts)
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("expected ErrNotDetected when user declines, got %v", err)
	}
}
