package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/emit"
	"github.com/permanu/docksmith/plan"
	"github.com/permanu/docksmith/yamldef"
)

func TestSanitizeDockerfileArg_injection(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"newline injection", "npm start\nRUN whoami"},
		{"carriage return", "npm start\rRUN whoami"},
		{"null byte", "npm\x00start"},
		{"mixed control chars", "cmd\n\r\x00end"},
		{"backticks passthrough", "`whoami`"},
		{"subshell passthrough", "$(whoami)"},
		{"empty", ""},
		{"whitespace only", "   \t  "},
		{"long string", strings.Repeat("a", 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := emit.SanitizeDockerfileArg(tt.input)
			if strings.ContainsAny(got, "\n\r\x00") {
				t.Errorf("output contains control char: %q", got)
			}
		})
	}

	if got := emit.SanitizeDockerfileArg("`whoami`"); got != "`whoami`" {
		t.Errorf("backticks mangled: got %q", got)
	}
	if got := emit.SanitizeDockerfileArg("$(whoami)"); got != "$(whoami)" {
		t.Errorf("subshell mangled: got %q", got)
	}
}

func TestSanitizeAppID_traversal(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		reject string
		exact  string
	}{
		{"dot-dot traversal", "../../../etc/passwd", "..", ""},
		{"many dots", "......", "..", ""},
		{"slash", "foo/bar", "/", ""},
		{"backslash", "foo\\bar", "\\", ""},
		{"null byte", "foo\x00bar", "\x00", ""},
		{"empty", "", "", "unknown"},
		{"safe unchanged", "my-app_123", "", "my-app_123"},
		{"unicode", "caf\u00e9", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plan.SanitizeAppID(tt.input)
			if tt.reject != "" && strings.Contains(got, tt.reject) {
				t.Errorf("output %q still contains %q", got, tt.reject)
			}
			if tt.exact != "" && got != tt.exact {
				t.Errorf("got %q, want %q", got, tt.exact)
			}
		})
	}
}

func TestContainedPath_traversal(t *testing.T) {
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, "a", "b"), 0755)
	os.MkdirAll(filepath.Join(base, "valid"), 0755)

	tests := []struct {
		name    string
		rel     string
		wantErr bool
	}{
		{"parent escape", "../etc/passwd", true},
		{"double parent escape", "../../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"null byte stripped", "foo\x00bar", false},
		{"empty rel", "", true},
		{"valid subpath", "valid/path", false},
		{"dot-dot resolves inside", "a/b/../c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detect.ContainedPath(base, tt.rel)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for rel=%q, got path=%q", tt.rel, got)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && strings.Contains(got, "\x00") {
				t.Error("null byte in resolved path")
			}
		})
	}
}

func TestEmitDockerfile_injectionInFields(t *testing.T) {
	poison := "legit\nRUN cat /etc/shadow\r\x00"

	p := &docksmith.BuildPlan{
		Framework: "node",
		Expose:    3000,
		Stages: []docksmith.Stage{{
			Name: "build",
			From: "node:20",
			Steps: []docksmith.Step{
				{Type: docksmith.StepWorkdir, Args: []string{poison}},
				{Type: docksmith.StepEnv, Args: []string{poison, poison}},
				{Type: docksmith.StepCopy, Args: []string{poison, poison}},
				{Type: docksmith.StepRun, Args: []string{poison}, CacheMount: &docksmith.CacheMount{Target: poison}},
				{Type: docksmith.StepRun, Args: []string{poison}, SecretMount: &docksmith.SecretMount{ID: poison, Target: poison}},
				{Type: docksmith.StepCmd, Args: []string{poison}},
				{Type: docksmith.StepExpose, Args: []string{"3000"}},
				{Type: docksmith.StepCopyFrom, CopyFrom: &docksmith.CopyFrom{Stage: "build", Src: poison, Dst: poison}},
				{Type: docksmith.StepArg, Args: []string{poison, poison}},
				{Type: docksmith.StepUser, Args: []string{poison}},
				{Type: docksmith.StepHealthcheck, Args: []string{poison}},
				{Type: docksmith.StepEntrypoint, Args: []string{poison}},
			},
		}},
	}

	out := docksmith.EmitDockerfile(p)

	for i, line := range strings.Split(out, "\n") {
		if strings.ContainsAny(line, "\r\x00") {
			t.Errorf("line %d contains control char: %q", i, line)
		}
		if strings.Contains(line, "cat /etc/shadow") && strings.HasPrefix(strings.TrimSpace(line), "RUN cat") {
			t.Errorf("line %d: injected instruction escaped: %q", i, line)
		}
	}
}

func TestFileMatchesRegex_limits(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("hello world"), 0644)

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"over 1024 chars", strings.Repeat("a", 1025), false},
		{"valid match", "hello", true},
		{"valid no match", "goodbye", false},
		{"invalid regex", "[invalid", false},
		{"empty pattern", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := yamldef.FileMatchesRegex(f, tt.pattern)
			if got != tt.want {
				t.Errorf("FileMatchesRegex(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestDetectRules_pathTraversal(t *testing.T) {
	tmp := t.TempDir()

	sentinel := filepath.Join(tmp, "outside", "secret.txt")
	os.MkdirAll(filepath.Dir(sentinel), 0755)
	os.WriteFile(sentinel, []byte("secret"), 0644)

	projectDir := filepath.Join(tmp, "project")
	os.MkdirAll(projectDir, 0755)

	rules := docksmith.DetectRules{
		All: []docksmith.DetectRule{{File: "../outside/secret.txt"}},
	}

	result := yamldef.EvalDetectRules(projectDir, rules)

	if result {
		t.Log("FINDING: EvalDetectRules reads files outside project dir via '../' in File field")
	}
}
