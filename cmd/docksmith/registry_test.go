package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/permanu/docksmith/registry"
)

func init() {
	registry.SetAllowInsecureHTTP(true)
}

var testIndex = registry.Index{
	Version: 1,
	Frameworks: map[string]registry.Entry{
		"gleam": {
			Version:     "1.2.0",
			Description: "Gleam to Erlang shipment",
			Runtime:     "erlang",
			Author:      "gleam-community",
			SHA256:      "abc",
			URL:         "http://example.com/gleam.yaml",
		},
		"htmx-go": {
			Version:     "0.1.0",
			Description: "Go + HTMX templates",
			Runtime:     "go",
			Author:      "community",
			SHA256:      "def",
			URL:         "http://example.com/htmx-go.yaml",
		},
		"solid": {
			Version:     "1.0.0",
			Description: "SolidJS frontend",
			Runtime:     "node",
			Author:      "solid-team",
			SHA256:      "ghi",
			URL:         "http://example.com/solid.yaml",
		},
	},
}

func serveIndex(t *testing.T, idx registry.Index) *httptest.Server {
	t.Helper()
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRegistryURL_flagOverride(t *testing.T) {
	t.Setenv("DOCKSMITH_REGISTRY", "http://env.example.com/index.json")
	got := registryURL("http://flag.example.com/index.json")
	if got != "http://flag.example.com/index.json" {
		t.Errorf("flag should win: got %q", got)
	}
}

func TestRegistryURL_envFallback(t *testing.T) {
	t.Setenv("DOCKSMITH_REGISTRY", "http://env.example.com/index.json")
	got := registryURL("")
	if got != "http://env.example.com/index.json" {
		t.Errorf("env should be used: got %q", got)
	}
}

func TestRegistryURL_default(t *testing.T) {
	t.Setenv("DOCKSMITH_REGISTRY", "")
	got := registryURL("")
	if got != registry.DefaultRegistryURL {
		t.Errorf("should fall back to default: got %q", got)
	}
}

func TestFindEntry_caseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gleam", "gleam"},
		{"GLEAM", "gleam"},
		{"Gleam", "gleam"},
		{"htmx-go", "htmx-go"},
		{"HTMX-GO", "htmx-go"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			e, ok := findEntry(&testIndex, tt.input)
			if !ok {
				t.Fatalf("expected match for %q", tt.input)
			}
			if e.Name != tt.want {
				t.Errorf("got name %q, want %q", e.Name, tt.want)
			}
		})
	}
}

func TestFindEntry_notFound(t *testing.T) {
	_, ok := findEntry(&testIndex, "nonexistent")
	if ok {
		t.Error("expected no match")
	}
}

func TestSuggestNames_substringMatch(t *testing.T) {
	suggestions := suggestNames(&testIndex, "go")
	found := false
	for _, s := range suggestions {
		if s == "htmx-go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected htmx-go in suggestions, got %v", suggestions)
	}
}

func TestSuggestNames_noMatch(t *testing.T) {
	suggestions := suggestNames(&testIndex, "zzz")
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions, got %v", suggestions)
	}
}

func TestSuggestNames_maxThree(t *testing.T) {
	big := &registry.Index{Frameworks: map[string]registry.Entry{
		"a-x": {}, "b-x": {}, "c-x": {}, "d-x": {}, "e-x": {},
	}}
	suggestions := suggestNames(big, "x")
	if len(suggestions) > 3 {
		t.Errorf("max 3 suggestions, got %d", len(suggestions))
	}
}

func TestSearchOutput_text(t *testing.T) {
	srv := serveIndex(t, testIndex)
	t.Setenv("HOME", t.TempDir())

	var out, errw bytes.Buffer
	err := execSearch(config{format: "text"}, srv.URL, false, []string{"gleam"}, &out, &errw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "NAME") {
		t.Error("text output missing table header")
	}
	if !strings.Contains(out.String(), "gleam") {
		t.Error("text output missing gleam entry")
	}
	if !strings.Contains(out.String(), "erlang") {
		t.Error("text output missing runtime column")
	}
}

func TestSearchOutput_json(t *testing.T) {
	srv := serveIndex(t, testIndex)
	t.Setenv("HOME", t.TempDir())

	var out, errw bytes.Buffer
	err := execSearch(config{format: "json"}, srv.URL, false, []string{"gleam"}, &out, &errw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var results []searchResult
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		t.Fatalf("json parse: %v\nraw: %s", err, out.String())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "gleam" {
		t.Errorf("expected gleam, got %s", results[0].Name)
	}
}

func TestSearchOutput_noResults(t *testing.T) {
	srv := serveIndex(t, testIndex)
	t.Setenv("HOME", t.TempDir())

	var out, errw bytes.Buffer
	err := execSearch(config{format: "text"}, srv.URL, false, []string{"zzzzz"}, &out, &errw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(errw.String(), `no frameworks found matching "zzzzz"`) {
		t.Errorf("expected 'no frameworks found matching' message, got: %s", errw.String())
	}
}

func TestRegistryFlagOverride(t *testing.T) {
	srv := serveIndex(t, testIndex)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DOCKSMITH_REGISTRY", "http://should-not-be-used")

	var out, errw bytes.Buffer
	err := execSearch(config{format: "text"}, srv.URL, false, []string{""}, &out, &errw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "gleam") {
		t.Error("flag URL should override env var")
	}
}
