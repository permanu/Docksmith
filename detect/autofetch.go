package detect

import (
	"path/filepath"
	"strings"
)

// markerFiles maps project marker filenames to a search query for the
// community registry. Only the first match is used.
var markerFiles = []struct {
	file  string
	query string
}{
	{"mix.exs", "elixir"},
	{"Cargo.toml", "rust"},
	{"go.mod", "go"},
	{"package.json", "node"},
	{"requirements.txt", "python"},
	{"pyproject.toml", "python"},
	{"Gemfile", "ruby"},
	{"composer.json", "php"},
	{"build.gradle", "java"},
	{"pom.xml", "java"},
	{"*.csproj", "dotnet"},
	{"deno.json", "deno"},
	{"bun.lockb", "bun"},
}

// SearchQueryFromDir inspects dir for well-known marker files and returns a
// registry search query that describes the project's runtime. Returns "" when
// no marker is found.
func SearchQueryFromDir(dir string) string {
	for _, m := range markerFiles {
		if strings.Contains(m.file, "*") {
			matches, _ := filepath.Glob(filepath.Join(dir, m.file))
			if len(matches) > 0 {
				return m.query
			}
			continue
		}
		if hasFile(dir, m.file) {
			return m.query
		}
	}
	return ""
}
