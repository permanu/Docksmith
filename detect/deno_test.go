package detect

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith/core"
)

func TestDetectDenoFresh(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    string
	}{
		{"fresh via deno.json import", "deno-fresh", "deno-fresh"},
		{"oak is not fresh", "deno-oak", ""},
		{"plain deno is not fresh", "deno-plain", ""},
		{"no deno.json", "deno-no-json", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join("testdata", "fixtures", tt.fixture)
			fw := detectDenoFresh(dir)
			if tt.want == "" {
				if fw != nil {
					t.Errorf("got %q, want nil", fw.Name)
				}
				return
			}
			if fw == nil {
				t.Fatal("got nil, want framework")
			}
			if fw.Name != tt.want {
				t.Errorf("Name = %q, want %q", fw.Name, tt.want)
			}
			if fw.Port != 8000 {
				t.Errorf("Port = %d, want 8000", fw.Port)
			}
			if fw.StartCommand != "deno run -A main.ts" {
				t.Errorf("StartCommand = %q, want %q", fw.StartCommand, "deno run -A main.ts")
			}
		})
	}
}

func TestDetectDenoFresh_FreshConfigTs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "fresh.config.ts", `export const config = {};`)

	fw := detectDenoFresh(dir)
	if fw == nil {
		t.Fatal("got nil, want deno-fresh")
	}
	if fw.Name != "deno-fresh" {
		t.Errorf("Name = %q, want deno-fresh", fw.Name)
	}
}

func TestDetectDenoOak(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    string
	}{
		{"oak in main.ts", "deno-oak", "deno-oak"},
		{"fresh is not oak", "deno-fresh", ""},
		{"plain deno without oak", "deno-plain", ""},
		{"no deno.json", "deno-no-json", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join("testdata", "fixtures", tt.fixture)
			fw := detectDenoOak(dir)
			if tt.want == "" {
				if fw != nil {
					t.Errorf("got %q, want nil", fw.Name)
				}
				return
			}
			if fw == nil {
				t.Fatal("got nil, want framework")
			}
			if fw.Name != tt.want {
				t.Errorf("Name = %q, want %q", fw.Name, tt.want)
			}
			if fw.Port != 8000 {
				t.Errorf("Port = %d, want 8000", fw.Port)
			}
		})
	}
}

func TestDetectDenoOak_ViaDenoJson(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deno.json", `{"imports": {"oak": "https://deno.land/x/oak@v12/mod.ts"}}`)

	fw := detectDenoOak(dir)
	if fw == nil {
		t.Fatal("got nil, want deno-oak")
	}
	if fw.Name != "deno-oak" {
		t.Errorf("Name = %q, want deno-oak", fw.Name)
	}
	if fw.StartCommand != "deno run --allow-net --allow-read main.ts" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectDenoPlain(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    string
	}{
		{"deno.json present", "deno-plain", "deno"},
		{"deno.jsonc present", "deno-plain-jsonc", "deno"},
		{"no deno.json", "deno-no-json", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join("testdata", "fixtures", tt.fixture)
			fw := detectDenoPlain(dir)
			if tt.want == "" {
				if fw != nil {
					t.Errorf("got %q, want nil", fw.Name)
				}
				return
			}
			if fw == nil {
				t.Fatal("got nil, want framework")
			}
			if fw.Name != tt.want {
				t.Errorf("Name = %q, want %q", fw.Name, tt.want)
			}
			if fw.Port != 8000 {
				t.Errorf("Port = %d, want 8000", fw.Port)
			}
		})
	}
}

func TestDetectDenoVersion(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    string
	}{
		{
			"version from deno.json",
			map[string]string{"deno.json": `{"version": "1.2.3"}`},
			"1.2",
		},
		{
			"version from .dvmrc",
			map[string]string{".dvmrc": "1.40.0\n"},
			"1.40",
		},
		{
			"no version files",
			map[string]string{},
			"2",
		},
		{
			"malformed deno.json falls back to default",
			map[string]string{"deno.json": `not json`},
			"2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			got := detectDenoVersion(dir)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindDenoEntrypoint(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{"main.ts", []string{"main.ts"}, "main.ts"},
		{"mod.ts preferred over default", []string{"mod.ts"}, "mod.ts"},
		{"server.ts", []string{"server.ts"}, "server.ts"},
		{"src/main.ts", []string{"src/main.ts"}, "src/main.ts"},
		{"none — defaults to main.ts", []string{}, "main.ts"},
		{"main.ts wins over mod.ts", []string{"main.ts", "mod.ts"}, "main.ts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				writeFile(t, dir, f, "")
			}
			got := findDenoEntrypoint(dir)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectDeno_FullStack_FreshWins(t *testing.T) {
	// Fresh must win over Oak and plain when $fresh is in deno.json.
	dir := filepath.Join("testdata", "fixtures", "deno-fresh")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "deno-fresh" {
		t.Errorf("Name = %q, want deno-fresh", fw.Name)
	}
}

func TestDetectDeno_FullStack_OakWins(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "deno-oak")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "deno-oak" {
		t.Errorf("Name = %q, want deno-oak", fw.Name)
	}
}

func TestDetectDeno_FullStack_Plain(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "deno-plain")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "deno" {
		t.Errorf("Name = %q, want deno", fw.Name)
	}
}

func TestDetectDeno_NotDetected_WithoutDenoJson(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "deno-no-json")
	_, err := Detect(dir)
	// Without deno.json, Deno should not be detected. Directory has only main.ts
	// which is not a recognized framework indicator — expect ErrNotDetected.
	if err == nil {
		t.Fatal("expected error for dir without deno.json")
	}
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("error = %v, want core.ErrNotDetected", err)
	}
}

func TestDetectDenoFresh_DenoVersion(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "deno-fresh")
	fw := detectDenoFresh(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	// deno-fresh fixture has version "1.2.3" in deno.json
	if fw.DenoVersion != "1.2" {
		t.Errorf("DenoVersion = %q, want 1.2", fw.DenoVersion)
	}
}
