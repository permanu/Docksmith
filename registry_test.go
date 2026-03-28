package docksmith_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/permanu/docksmith"
)

var sampleIndex = docksmith.RegistryIndex{
	Version: 1,
	Frameworks: map[string]docksmith.RegistryEntry{
		"gleam": {
			Version:     "1.2.0",
			Description: "Gleam to Erlang shipment",
			Runtime:     "erlang",
			Author:      "gleam-community",
		},
		"htmx-go": {
			Version:     "0.1.0",
			Description: "Go + HTMX templates",
			Runtime:     "go",
			Author:      "community",
		},
		"solid": {
			Version:     "1.0.0",
			Description: "SolidJS frontend",
			Runtime:     "node",
			Author:      "solid-team",
		},
	},
}

func marshalIndex(t *testing.T, idx docksmith.RegistryIndex) []byte {
	t.Helper()
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}
	return data
}

func TestFetchRegistryIndex_fromServer(t *testing.T) {
	payload := marshalIndex(t, sampleIndex)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}))
	defer srv.Close()

	// Use a fresh cache dir so we always hit the server.
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	idx, err := docksmith.FetchRegistryIndex(srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Version != 1 {
		t.Errorf("version: want 1 got %d", idx.Version)
	}
	if _, ok := idx.Frameworks["gleam"]; !ok {
		t.Error("gleam entry missing from index")
	}
}

func TestFetchRegistryIndex_cacheTTL(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	cachePath := filepath.Join(cacheDir, ".docksmith", "cache", "index.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}

	// Fresh cache — should use it without hitting network.
	idx, err := docksmith.FetchRegistryIndex("http://should-not-be-called", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Frameworks) == 0 {
		t.Error("expected frameworks from cache")
	}
}

func TestFetchRegistryIndex_expiredCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	cachePath := filepath.Join(cacheDir, ".docksmith", "cache", "index.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	// Backdate the file by 25 hours to simulate expiry.
	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(cachePath, old, old); err != nil {
		t.Fatal(err)
	}

	calls := 0
	fresh := docksmith.RegistryIndex{Version: 2, Frameworks: map[string]docksmith.RegistryEntry{
		"newfw": {Version: "0.0.1"},
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write(marshalIndex(t, fresh))
	}))
	defer srv.Close()

	idx, err := docksmith.FetchRegistryIndex(srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Version != 2 {
		t.Errorf("expected fresh index (version 2), got %d", idx.Version)
	}
	if calls != 1 {
		t.Errorf("expected 1 server call, got %d", calls)
	}
}

func TestFetchRegistryIndex_offline_noCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	_, err := docksmith.FetchRegistryIndex("http://irrelevant", true)
	if err == nil {
		t.Fatal("expected error for offline with no cache")
	}
}

func TestFetchRegistryIndex_serverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	_, err := docksmith.FetchRegistryIndex(srv.URL, false)
	if err == nil {
		t.Fatal("expected error for server 500")
	}
}

func TestSearchRegistry_exactMatch(t *testing.T) {
	results := docksmith.SearchRegistry(&sampleIndex, "gleam")
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Name != "gleam" {
		t.Errorf("want gleam, got %s", results[0].Name)
	}
}

func TestSearchRegistry_partialMatch(t *testing.T) {
	results := docksmith.SearchRegistry(&sampleIndex, "go")
	// matches htmx-go (name) and htmx-go (runtime=go) and possibly solid's description
	found := false
	for _, r := range results {
		if r.Name == "htmx-go" {
			found = true
		}
	}
	if !found {
		t.Error("htmx-go should match query 'go'")
	}
}

func TestSearchRegistry_runtimeMatch(t *testing.T) {
	results := docksmith.SearchRegistry(&sampleIndex, "node")
	if len(results) != 1 || results[0].Name != "solid" {
		t.Errorf("expected solid (runtime=node), got %v", results)
	}
}

func TestSearchRegistry_noMatch(t *testing.T) {
	results := docksmith.SearchRegistry(&sampleIndex, "zig")
	if len(results) != 0 {
		t.Errorf("want 0 results for 'zig', got %d", len(results))
	}
}

func TestSearchRegistry_emptyQuery(t *testing.T) {
	results := docksmith.SearchRegistry(&sampleIndex, "")
	if len(results) != len(sampleIndex.Frameworks) {
		t.Errorf("empty query should return all %d entries, got %d", len(sampleIndex.Frameworks), len(results))
	}
}

func TestInstallFramework_writesFile(t *testing.T) {
	yamlContent := "name: gleam\nruntime: erlang\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yamlContent)
	}))
	defer srv.Close()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// SHA256 of "name: gleam\nruntime: erlang\n"
	h := sha256.Sum256([]byte(yamlContent))
	checksum := hex.EncodeToString(h[:])

	entry := docksmith.RegistryEntry{
		Name:    "gleam",
		Version: "1.2.0",
		URL:     srv.URL + "/gleam.yaml",
		SHA256:  checksum,
	}

	dest, err := docksmith.InstallFramework(entry)
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

func TestInstallFramework_noURL(t *testing.T) {
	entry := docksmith.RegistryEntry{Name: "broken"}
	_, err := docksmith.InstallFramework(entry)
	if err == nil {
		t.Fatal("expected error for entry with no URL")
	}
}
