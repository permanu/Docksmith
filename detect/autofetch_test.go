package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchQueryFromDir_MarkerFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{"mix.exs -> elixir", []string{"mix.exs"}, "elixir"},
		{"Cargo.toml -> rust", []string{"Cargo.toml"}, "rust"},
		{"go.mod -> go", []string{"go.mod"}, "go"},
		{"package.json -> node", []string{"package.json"}, "node"},
		{"requirements.txt -> python", []string{"requirements.txt"}, "python"},
		{"pyproject.toml -> python", []string{"pyproject.toml"}, "python"},
		{"Gemfile -> ruby", []string{"Gemfile"}, "ruby"},
		{"composer.json -> php", []string{"composer.json"}, "php"},
		{"build.gradle -> java", []string{"build.gradle"}, "java"},
		{"pom.xml -> java", []string{"pom.xml"}, "java"},
		{"deno.json -> deno", []string{"deno.json"}, "deno"},
		{"bun.lockb -> bun", []string{"bun.lockb"}, "bun"},
		{"no markers -> empty", []string{"README.md"}, ""},
		{"empty dir -> empty", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := SearchQueryFromDir(dir)
			if got != tt.want {
				t.Errorf("SearchQueryFromDir = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSearchQueryFromDir_GlobPattern(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte("<Project/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := SearchQueryFromDir(dir)
	if got != "dotnet" {
		t.Errorf("SearchQueryFromDir = %q, want %q", got, "dotnet")
	}
}

func TestSearchQueryFromDir_PriorityOrder(t *testing.T) {
	// mix.exs appears before go.mod in the list, so elixir wins.
	dir := t.TempDir()
	for _, f := range []string{"go.mod", "mix.exs"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := SearchQueryFromDir(dir)
	if got != "elixir" {
		t.Errorf("SearchQueryFromDir = %q, want %q", got, "elixir")
	}
}
