package registry_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith/registry"
)

func init() {
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

	t.Setenv("HOME", t.TempDir())

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

func TestFetchIndex_rejectsHTTP(t *testing.T) {
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	t.Setenv("HOME", t.TempDir())

	_, err := registry.FetchIndex("http://example.com/index.json", false)
	if err == nil {
		t.Fatal("expected error for non-HTTPS registry URL")
	}
}

func TestFetchIndex_rejectsFTP(t *testing.T) {
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	t.Setenv("HOME", t.TempDir())

	_, err := registry.FetchIndex("ftp://example.com/index.json", false)
	if err == nil {
		t.Fatal("expected error for ftp:// scheme")
	}
}

func TestFetchIndex_rejectsFileScheme(t *testing.T) {
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	t.Setenv("HOME", t.TempDir())

	_, err := registry.FetchIndex("file:///etc/passwd", false)
	if err == nil {
		t.Fatal("expected error for file:// scheme")
	}
}

func TestFetchIndex_serverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())

	_, err := registry.FetchIndex(srv.URL, false)
	if err == nil {
		t.Fatal("expected error for server 500")
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

func TestSearch_descriptionMatch(t *testing.T) {
	results := registry.Search(&sampleIndex, "HTMX")
	if len(results) != 1 || results[0].Name != "htmx-go" {
		t.Errorf("expected htmx-go from description match, got %v", results)
	}
}

func TestSearch_caseInsensitive(t *testing.T) {
	results := registry.Search(&sampleIndex, "GLEAM")
	if len(results) != 1 || results[0].Name != "gleam" {
		t.Errorf("expected case-insensitive match on gleam, got %v", results)
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
	for i := 1; i < len(results); i++ {
		if results[i-1].Name > results[i].Name {
			t.Errorf("results not sorted: %q > %q", results[i-1].Name, results[i].Name)
		}
	}
}

