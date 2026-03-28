package yamldef

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVersionFromFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("20\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	def := &FrameworkDef{
		Version: VersionConfig{
			Sources: []VersionSource{{File: ".nvmrc"}},
			Default: "22",
		},
	}
	got := ResolveVersion(def, dir)
	if got != "20" {
		t.Errorf("ResolveVersion: got %q, want %q", got, "20")
	}
}

func TestResolveVersionFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	def := &FrameworkDef{
		Version: VersionConfig{
			Sources: []VersionSource{{File: ".nvmrc"}}, // absent
			Default: "22",
		},
	}
	got := ResolveVersion(def, dir)
	if got != "22" {
		t.Errorf("ResolveVersion: got %q, want %q", got, "22")
	}
}

func TestResolveVersionFromJSONPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"engines":{"node":"18"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	def := &FrameworkDef{
		Version: VersionConfig{
			Sources: []VersionSource{{JSON: "package.json", Path: "engines.node"}},
		},
	}
	got := ResolveVersion(def, dir)
	if got != "18" {
		t.Errorf("ResolveVersion: got %q, want %q", got, "18")
	}
}

func TestResolvePMFromLockfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	def := &FrameworkDef{
		PackageManager: PMConfig{
			Sources: []PMSource{
				{File: "pnpm-lock.yaml", Value: "pnpm"},
				{File: "yarn.lock", Value: "yarn"},
			},
			Default: "npm",
		},
	}
	got := ResolvePM(def, dir)
	if got != "pnpm" {
		t.Errorf("ResolvePM: got %q, want %q", got, "pnpm")
	}
}

func TestResolvePMFromJSONPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"packageManager":"yarn@3.6.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	def := &FrameworkDef{
		PackageManager: PMConfig{
			Sources: []PMSource{
				{JSON: "package.json", Path: "packageManager"},
			},
			Default: "npm",
		},
	}
	got := ResolvePM(def, dir)
	if got != "yarn" {
		t.Errorf("ResolvePM: got %q, want %q", got, "yarn")
	}
}

func TestPMLockfileName(t *testing.T) {
	tests := []struct {
		pm   string
		want string
	}{
		{"npm", "package-lock.json"},
		{"pnpm", "pnpm-lock.yaml"},
		{"yarn", "yarn.lock"},
		{"bun", "bun.lockb"},
		{"pip", "requirements.txt"},
		{"poetry", "poetry.lock"},
		{"cargo", "Cargo.lock"},
		{"bundler", "Gemfile.lock"},
		{"composer", "composer.lock"},
		{"unknown", "package-lock.json"},
	}
	for _, tt := range tests {
		got := PMLockfileName(tt.pm)
		if got != tt.want {
			t.Errorf("PMLockfileName(%q): got %q, want %q", tt.pm, got, tt.want)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]string{"c": "3", "a": "1", "b": "2"}
	keys := SortedKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("SortedKeys: got %v, want [a b c]", keys)
	}
}

func TestSub(t *testing.T) {
	vars := map[string]string{
		"{{name}}": "world",
		"{{lang}}": "Go",
	}
	got := Sub("hello {{name}}, using {{lang}}", vars)
	if got != "hello world, using Go" {
		t.Errorf("Sub: got %q", got)
	}
}

func TestSubUnknownVarLeftInPlace(t *testing.T) {
	vars := map[string]string{"{{known}}": "yes"}
	got := Sub("{{known}} and {{unknown}}", vars)
	if got != "yes and {{unknown}}" {
		t.Errorf("Sub: got %q", got)
	}
}

func TestResolveInstallCommand(t *testing.T) {
	def := &FrameworkDef{
		Defaults: DefaultsDef{
			Install: map[string]string{
				"npm":  "npm ci",
				"pnpm": "pnpm install --frozen-lockfile",
			},
		},
	}
	if got := ResolveInstallCommand(def, "npm"); got != "npm ci" {
		t.Errorf("npm: got %q", got)
	}
	if got := ResolveInstallCommand(def, "pnpm"); got != "pnpm install --frozen-lockfile" {
		t.Errorf("pnpm: got %q", got)
	}
	if got := ResolveInstallCommand(def, "yarn"); got != "" {
		t.Errorf("yarn: got %q, want empty", got)
	}
}
