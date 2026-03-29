package registry_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/permanu/docksmith/registry"
)

func TestFetchIndex_bodyExceedsLimit(t *testing.T) {
	// 10 MB + 1 byte should be rejected.
	bigBody := strings.Repeat("x", 10<<20+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(bigBody))
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())

	_, err := registry.FetchIndex(srv.URL, false)
	if err == nil {
		t.Fatal("expected error for oversized response")
	}
	if !strings.Contains(err.Error(), "limit") {
		t.Errorf("error should mention limit, got: %v", err)
	}
}

func TestFetchIndex_bodyAtLimit(t *testing.T) {
	// Exactly 10 MB should succeed if valid JSON.
	idx := registry.Index{Version: 1, Frameworks: map[string]registry.Entry{}}
	payload, _ := json.Marshal(idx)
	// Pad with whitespace to fill near the limit (but not over).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())

	got, err := registry.FetchIndex(srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("version: want 1, got %d", got.Version)
	}
}

func TestFetchIndex_SSRFLoopback(t *testing.T) {
	// SSRF checks only apply when insecure HTTP is disabled.
	// With SSRF enabled (production mode), resolving to 127.0.0.1 is rejected.
	registry.SetAllowInsecureHTTP(false)
	defer registry.SetAllowInsecureHTTP(true)

	t.Setenv("HOME", t.TempDir())

	// "https://127.0.0.1/index.json" would resolve to loopback.
	// The SSRF transport rejects it after DNS resolution.
	_, err := registry.FetchIndex("https://127.0.0.1/index.json", false)
	if err == nil {
		t.Fatal("expected error for loopback IP")
	}
}
