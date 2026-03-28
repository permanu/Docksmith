package yamldef

import (
	"os"
	"path/filepath"
	"testing"
)

// makeDetectFixture creates a temporary directory with the given files.
func makeDetectFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// makeDetectDir creates a subdirectory inside the fixture dir.
func makeDetectDir(t *testing.T, base, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(base, rel), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
}

// --- EvalRule: file ---

func TestEvalRuleFileExists(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{"package.json": "{}"})
	rule := DetectRule{File: "package.json"}
	if !EvalRule(dir, rule) {
		t.Error("expected rule to match when file exists")
	}
}

func TestEvalRuleFileNotExists(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	rule := DetectRule{File: "go.mod"}
	if EvalRule(dir, rule) {
		t.Error("expected rule to not match when file is absent")
	}
}

func TestEvalRuleFileGlob(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{"main.py": ""})
	rule := DetectRule{File: "*.py"}
	if !EvalRule(dir, rule) {
		t.Error("expected glob rule to match *.py when main.py exists")
	}
}

func TestEvalRuleFileGlobNoMatch(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{"main.go": ""})
	rule := DetectRule{File: "*.py"}
	if EvalRule(dir, rule) {
		t.Error("expected glob rule *.py to not match when only .go files exist")
	}
}

// --- EvalRule: dir ---

func TestEvalRuleDirExists(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	makeDetectDir(t, dir, "src")
	rule := DetectRule{Dir: "src"}
	if !EvalRule(dir, rule) {
		t.Error("expected dir rule to match when directory exists")
	}
}

func TestEvalRuleDirNotExists(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	rule := DetectRule{Dir: "nonexistent"}
	if EvalRule(dir, rule) {
		t.Error("expected dir rule to not match for absent directory")
	}
}

func TestEvalRuleDirFileIsNotDir(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{"src": "I am a file"})
	rule := DetectRule{Dir: "src"}
	if EvalRule(dir, rule) {
		t.Error("expected dir rule to return false when 'src' is a file, not a directory")
	}
}

// --- EvalRule: contains ---

func TestEvalRuleContains(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"dependencies":{"next":"13.0.0"}}`,
	})
	rule := DetectRule{File: "package.json", Contains: `"next"`}
	if !EvalRule(dir, rule) {
		t.Error("expected contains rule to match")
	}
}

func TestEvalRuleContainsNotFound(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"dependencies":{"express":"4.0.0"}}`,
	})
	rule := DetectRule{File: "package.json", Contains: `"next"`}
	if EvalRule(dir, rule) {
		t.Error("expected contains rule to not match")
	}
}

func TestEvalRuleContainsMissingFile(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	rule := DetectRule{File: "package.json", Contains: "anything"}
	if EvalRule(dir, rule) {
		t.Error("expected contains rule to return false when file is missing")
	}
}

// --- EvalRule: regex ---

func TestEvalRuleRegexMatches(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"go.mod": "module example.com/myapp\n\ngo 1.21\n",
	})
	rule := DetectRule{File: "go.mod", Regex: `^module\s+`}
	if !EvalRule(dir, rule) {
		t.Error("expected regex rule to match")
	}
}

func TestEvalRuleRegexNoMatch(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"go.mod": "module example.com/myapp\n",
	})
	rule := DetectRule{File: "go.mod", Regex: `^package\s+`}
	if EvalRule(dir, rule) {
		t.Error("expected regex rule to not match")
	}
}

func TestEvalRuleRegexInvalidPattern(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"file.txt": "hello",
	})
	rule := DetectRule{File: "file.txt", Regex: `[invalid`}
	if EvalRule(dir, rule) {
		t.Error("expected invalid regex pattern to return false")
	}
}

// --- EvalRule: dependency (package.json) ---

func TestEvalRuleDependencyPackageJSON(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"dependencies":{"next":"13.0.0"},"devDependencies":{"eslint":"8.0.0"}}`,
	})
	tests := []struct {
		dep  string
		want bool
	}{
		{"next", true},
		{"eslint", true},
		{"react", false},
	}
	for _, tt := range tests {
		rule := DetectRule{Dependency: tt.dep}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("dependency %q: got %v, want %v", tt.dep, got, tt.want)
		}
	}
}

func TestEvalRuleDependencyRequirementsTxt(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"requirements.txt": "Django>=3.2\nflask==2.0.0\n# comment\nrequests\n",
	})
	tests := []struct {
		dep  string
		want bool
	}{
		{"Django", true},
		{"django", true}, // case-insensitive
		{"flask", true},
		{"requests", true},
		{"numpy", false},
	}
	for _, tt := range tests {
		rule := DetectRule{Dependency: tt.dep}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("requirements dep %q: got %v, want %v", tt.dep, got, tt.want)
		}
	}
}

func TestEvalRuleDependencyGoMod(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.21\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.9.0\n)\n",
	})
	rule := DetectRule{Dependency: "github.com/gin-gonic/gin"}
	if !EvalRule(dir, rule) {
		t.Error("expected go.mod dependency to match")
	}
	rule2 := DetectRule{Dependency: "github.com/gorilla/mux"}
	if EvalRule(dir, rule2) {
		t.Error("expected absent go.mod dependency to not match")
	}
}

func TestEvalRuleDependencyGemfile(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"Gemfile": "source 'https://rubygems.org'\ngem 'rails', '~> 7.0'\ngem \"devise\"\n",
	})
	tests := []struct {
		dep  string
		want bool
	}{
		{"rails", true},
		{"devise", true},
		{"sinatra", false},
	}
	for _, tt := range tests {
		rule := DetectRule{Dependency: tt.dep}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("Gemfile dep %q: got %v, want %v", tt.dep, got, tt.want)
		}
	}
}

func TestEvalRuleDependencyComposerJSON(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"composer.json": `{"require":{"laravel/framework":"^10.0"},"require-dev":{"phpunit/phpunit":"^10"}}`,
	})
	tests := []struct {
		dep  string
		want bool
	}{
		{"laravel/framework", true},
		{"phpunit/phpunit", true},
		{"symfony/console", false},
	}
	for _, tt := range tests {
		rule := DetectRule{Dependency: tt.dep}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("composer dep %q: got %v, want %v", tt.dep, got, tt.want)
		}
	}
}

func TestEvalRuleDependencyCargoToml(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"Cargo.toml": "[package]\nname = \"myapp\"\n\n[dependencies]\naxum = \"0.7\"\n\n[dev-dependencies]\ntokio = \"1\"\n",
	})
	tests := []struct {
		dep  string
		want bool
	}{
		{"axum", true},
		{"tokio", true},
		{"serde", false},
	}
	for _, tt := range tests {
		rule := DetectRule{Dependency: tt.dep}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("Cargo dep %q: got %v, want %v", tt.dep, got, tt.want)
		}
	}
}

func TestEvalRuleDependencyMixExs(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"mix.exs": "defmodule App.MixProject do\n  defp deps do\n    [{:phoenix, \"~> 1.7\"}, {:ecto, \"~> 3.10\"}]\n  end\nend\n",
	})
	tests := []struct {
		dep  string
		want bool
	}{
		{"phoenix", true},
		{"ecto", true},
		{"plug", false},
	}
	for _, tt := range tests {
		rule := DetectRule{Dependency: tt.dep}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("mix dep %q: got %v, want %v", tt.dep, got, tt.want)
		}
	}
}

// --- EvalRule: json + path ---

func TestEvalRuleJSONPath(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"scripts":{"build":"next build"},"engines":{"node":"18"}}`,
	})
	tests := []struct {
		path string
		want bool
	}{
		{"scripts.build", true},
		{"engines.node", true},
		{"engines.missing", false},
		{"nonexistent.key", false},
	}
	for _, tt := range tests {
		rule := DetectRule{JSON: "package.json", Path: tt.path}
		got := EvalRule(dir, rule)
		if got != tt.want {
			t.Errorf("json path %q: got %v, want %v", tt.path, got, tt.want)
		}
	}
}

// --- EvalRule: toml + path ---

func TestEvalRuleTOMLPath(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"pyproject.toml": "[tool.poetry]\nname = \"myapp\"\n\n[build-system]\nrequires = [\"poetry\"]\n",
	})
	rule := DetectRule{TOML: "pyproject.toml", Path: "tool.poetry.name"}
	if !EvalRule(dir, rule) {
		t.Error("expected toml path rule to match for existing key")
	}
	ruleAbsent := DetectRule{TOML: "pyproject.toml", Path: "tool.poetry.missing"}
	if EvalRule(dir, ruleAbsent) {
		t.Error("expected toml path rule to not match for absent key")
	}
}

// --- EvalRule: empty rule ---

func TestEvalRuleEmptyVacuouslyTrue(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	rule := DetectRule{}
	if !EvalRule(dir, rule) {
		t.Error("empty rule should vacuously return true")
	}
}

// --- EvalDetectRules combinators ---

func TestEvalDetectRulesAllMatch(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"dependencies":{"next":"13"}}`,
	})
	rules := DetectRules{
		All: []DetectRule{
			{File: "package.json"},
			{Dependency: "next"},
		},
	}
	if !EvalDetectRules(dir, rules) {
		t.Error("expected all-match rules to return true")
	}
}

func TestEvalDetectRulesAllPartialFail(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"dependencies":{}}`,
	})
	rules := DetectRules{
		All: []DetectRule{
			{File: "package.json"},
			{Dependency: "next"}, // not present
		},
	}
	if EvalDetectRules(dir, rules) {
		t.Error("expected all-match rules to return false when one fails")
	}
}

func TestEvalDetectRulesAnyMatch(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"requirements.txt": "flask==2.0\n",
	})
	rules := DetectRules{
		Any: []DetectRule{
			{File: "go.mod"},      // absent
			{Dependency: "flask"}, // present
		},
	}
	if !EvalDetectRules(dir, rules) {
		t.Error("expected any-match rules to return true when at least one matches")
	}
}

func TestEvalDetectRulesAnyNoneMatch(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	rules := DetectRules{
		Any: []DetectRule{
			{File: "go.mod"},
			{File: "package.json"},
		},
	}
	if EvalDetectRules(dir, rules) {
		t.Error("expected any-match rules to return false when none match")
	}
}

func TestEvalDetectRulesNoneExcludes(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"Dockerfile": "FROM node:22\n",
	})
	rules := DetectRules{
		None: []DetectRule{
			{File: "Dockerfile"}, // present — should cause false
		},
	}
	if EvalDetectRules(dir, rules) {
		t.Error("expected none-match rules to return false when excluded file exists")
	}
}

func TestEvalDetectRulesNonePassWhenAbsent(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{}`,
	})
	rules := DetectRules{
		All:  []DetectRule{{File: "package.json"}},
		None: []DetectRule{{File: "Dockerfile"}},
	}
	if !EvalDetectRules(dir, rules) {
		t.Error("expected rules to pass when none-excluded file is absent")
	}
}

func TestEvalDetectRulesEmptyVacuouslyTrue(t *testing.T) {
	dir := makeDetectFixture(t, map[string]string{})
	rules := DetectRules{}
	if !EvalDetectRules(dir, rules) {
		t.Error("empty DetectRules should vacuously return true")
	}
}

func TestEvalDetectRulesCombined(t *testing.T) {
	// all + any + none
	dir := makeDetectFixture(t, map[string]string{
		"package.json": `{"dependencies":{"next":"13"},"devDependencies":{"typescript":"5"}}`,
	})
	rules := DetectRules{
		All: []DetectRule{{File: "package.json"}},
		Any: []DetectRule{
			{Dependency: "next"},
			{Dependency: "nuxt"},
		},
		None: []DetectRule{{File: "Dockerfile"}},
	}
	if !EvalDetectRules(dir, rules) {
		t.Error("expected combined rules to match")
	}
}
