package registry_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/permanu/docksmith/registry"
)

func TestFetchIndex_cacheFresh(t *testing.T) {
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

	idx, err := registry.FetchIndex(registryURL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idx.Frameworks) == 0 {
		t.Error("expected frameworks from cache")
	}
}

func TestFetchIndex_cacheStaleRefetch(t *testing.T) {
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

	cp := cachePath(cacheDir, srv.URL)
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}
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

func TestFetchIndex_cacheCorrupt(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	fresh := registry.Index{Version: 3, Frameworks: map[string]registry.Entry{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.Marshal(fresh)
		w.Write(data)
	}))
	defer srv.Close()

	cp := cachePath(cacheDir, srv.URL)
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write garbage to simulate corruption.
	if err := os.WriteFile(cp, []byte("{invalid json!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := registry.FetchIndex(srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Version != 3 {
		t.Errorf("expected fresh index (version 3), got %d", idx.Version)
	}
	// Corrupt file should have been deleted.
	if _, err := os.Stat(cp); err == nil {
		// File should exist now with fresh data from re-fetch.
		data, _ := os.ReadFile(cp)
		var cached registry.Index
		if err := json.Unmarshal(data, &cached); err != nil {
			t.Error("cache file still corrupt after re-fetch")
		}
	}
}

func TestFetchIndex_offlineFallbackStaleCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	// Server that always 500s.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// Seed stale cache for this URL.
	cp := cachePath(cacheDir, srv.URL)
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(cp, old, old); err != nil {
		t.Fatal(err)
	}

	// Fetch should fall back to stale cache.
	idx, err := registry.FetchIndex(srv.URL, false)
	if err != nil {
		t.Fatalf("expected stale cache fallback, got error: %v", err)
	}
	if idx.Version != 1 {
		t.Errorf("expected cached version 1, got %d", idx.Version)
	}
}

func TestFetchIndex_offlineNoCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := registry.FetchIndex("http://irrelevant", true)
	if err == nil {
		t.Fatal("expected error for offline with no cache")
	}
}

func TestFetchIndex_offlineWithStaleCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	registryURL := "http://wont-be-called"
	cp := cachePath(cacheDir, registryURL)
	if err := os.MkdirAll(filepath.Dir(cp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cp, marshalIndex(t, sampleIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	// Backdate so it's stale.
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(cp, old, old); err != nil {
		t.Fatal(err)
	}

	idx, err := registry.FetchIndex(registryURL, true)
	if err != nil {
		t.Fatalf("expected stale cache in offline mode, got: %v", err)
	}
	if idx.Version != 1 {
		t.Errorf("expected version 1, got %d", idx.Version)
	}
}

func TestFetchIndex_atomicCacheWrite(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(marshalIndex(t, sampleIndex))
	}))
	defer srv.Close()

	_, err := registry.FetchIndex(srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no temp files left behind.
	cacheParent := filepath.Join(cacheDir, ".docksmith", "cache")
	entries, err := os.ReadDir(cacheParent)
	if err != nil {
		t.Fatalf("read cache dir: %v", err)
	}
	for _, e := range entries {
		if e.Name()[0] == '.' {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
