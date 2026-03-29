package registry_test

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

	"github.com/permanu/docksmith/registry"
)

func init() {
	// Tests use httptest servers which serve over HTTP.
	registry.SetAllowInsecureHTTP(true)
}

var sampleIndex = registry.Index{
	Version: 1,
	Frameworks: map[string]registry.Entry{
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

func marshalIndex(t *testing.T, idx registry.Index) []byte {
	t.Helper()
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}
	return data
}

// cachePath computes the expected cache file for a given registry URL.
func cachePath(home, registryURL string) string {
	h := sha256.Sum256([]byte(registryURL))
	name := "index-" + hex.EncodeToString(h[:]) + ".json"
	return filepath.Join(home, ".docksmith", "cache", name)
}

func TestFetchIndex_fromServer(t *testing.T) {
	payload := marshalIndex(t, sampleIndex)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
	}))
	defer srv.Close()

	// Use a fresh cache dir so we always hit the server.
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	idx, err := registry.FetchIndex(srv.URL, false)
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

func TestFetchIndex_cacheTTL(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	registryURL := "http://should-not-be-called"
	cp := cachePath(cacheDir, registryURL)
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}

	// Fresh cache — should use it without hitting network.
	idx, err := registry.FetchIndex(registryURL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Frameworks) == 0 {
		t.Error("expected frameworks from cache")
	}
}

func TestFetchIndex_expiredCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	calls := 0
	fresh := registry.Index{Version: 2, Frameworks: map[string]registry.Entry{
		"newfw": {Version: "0.0.1"},
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write(marshalIndex(t, fresh))
	}))
	defer srv.Close()

	// Seed cache for this server URL.
	cp := cachePath(cacheDir, srv.URL)
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	// Backdate the file by 25 hours to simulate expiry.
	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(cp, old, old); err != nil {
		t.Fatal(err)
	}

	idx, err := registry.FetchIndex(srv.URL, false)
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

func TestFetchIndex_offline_noCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	_, err := registry.FetchIndex("http://irrelevant", true)
	if err == nil {
		t.Fatal("expected error for offline with no cache")
	}
}

func TestFetchIndex_serverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	_, err := registry.FetchIndex(srv.URL, false)
	if err == nil {
		t.Fatal("expected error for server 500")
	}
}

func TestFetchIndex_rejectsHTTP(t *testing.T) {
	// Temporarily disable the test hook.
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	_, err := registry.FetchIndex("http://example.com/index.json", false)
	if err == nil {
		t.Fatal("expected error for non-HTTPS registry URL")
	}
}

func TestSearch_exactMatch(t *testing.T) {
	results := registry.Search(&sampleIndex, "gleam")
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Name != "gleam" {
		t.Errorf("want gleam, got %s", results[0].Name)
	}
}

func TestSearch_partialMatch(t *testing.T) {
	results := registry.Search(&sampleIndex, "go")
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

func TestSearch_runtimeMatch(t *testing.T) {
	results := registry.Search(&sampleIndex, "node")
	if len(results) != 1 || results[0].Name != "solid" {
		t.Errorf("expected solid (runtime=node), got %v", results)
	}
}

func TestSearch_noMatch(t *testing.T) {
	results := registry.Search(&sampleIndex, "zig")
	if len(results) != 0 {
		t.Errorf("want 0 results for 'zig', got %d", len(results))
	}
}

func TestSearch_emptyQuery(t *testing.T) {
	results := registry.Search(&sampleIndex, "")
	if len(results) != len(sampleIndex.Frameworks) {
		t.Errorf("empty query should return all %d entries, got %d", len(sampleIndex.Frameworks), len(results))
	}
}

func TestSearch_sortedByName(t *testing.T) {
	results := registry.Search(&sampleIndex, "")
	if len(results) < 2 {
		t.Skip("need at least 2 results to test sorting")
	}
	for i := 1; i < len(results); i++ {
		if results[i-1].Name > results[i].Name {
			t.Errorf("results not sorted: %q > %q", results[i-1].Name, results[i].Name)
		}
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

func TestInstallFramework_noURL(t *testing.T) {
	entry := registry.Entry{Name: "broken"}
	_, err := registry.InstallFramework(entry)
	if err == nil {
		t.Fatal("expected error for entry with no URL")
	}
}

func TestInstallFramework_pathTraversal(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"../etc/passwd"},
		{"../../evil"},
		{"foo/bar"},
		{`foo\bar`},
		{".."},
		{"."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := registry.Entry{
				Name:   tc.name,
				URL:    "http://example.com/fw.yaml",
				SHA256: "abc",
			}
			_, err := registry.InstallFramework(entry)
			if err == nil {
				t.Errorf("expected error for malicious name %q", tc.name)
			}
		})
	}
}

func TestInstallFramework_rejectsHTTP(t *testing.T) {
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

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
