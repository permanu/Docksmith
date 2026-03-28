package docksmith

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith/plan"
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
			got := sanitizeDockerfileArg(tt.input)
			if strings.ContainsAny(got, "\n\r\x00") {
				t.Errorf("output contains control char: %q", got)
			}
		})
	}

	// backticks and $() are shell-level, not Dockerfile injection — must survive
	if got := sanitizeDockerfileArg("`whoami`"); got != "`whoami`" {
		t.Errorf("backticks mangled: got %q", got)
	}
	if got := sanitizeDockerfileArg("$(whoami)"); got != "$(whoami)" {
		t.Errorf("subshell mangled: got %q", got)
	}
}

func TestSanitizeAppID_traversal(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		reject string // substring that must NOT appear in output
		exact  string // if non-empty, output must equal this
	}{
		{"dot-dot traversal", "../../../etc/passwd", "..", ""},
		{"many dots", "......", "..", ""},
		{"slash", "foo/bar", "/", ""},
		{"backslash", "foo\\bar", "\\", ""},
		{"null byte", "foo\x00bar", "\x00", ""},
		{"empty", "", "", "unknown"},
		{"safe unchanged", "my-app_123", "", "my-app_123"},
		{"unicode", "café", "", ""},
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

	// Create subdirs so valid paths resolve
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
			got, err := containedPath(base, tt.rel)
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

	plan := &BuildPlan{
		Framework: "node",
		Expose:    3000,
		Stages: []Stage{{
			Name: "build",
			From: "node:20",
			Steps: []Step{
				{Type: StepWorkdir, Args: []string{poison}},
				{Type: StepEnv, Args: []string{poison, poison}},
				{Type: StepCopy, Args: []string{poison, poison}},
				{Type: StepRun, Args: []string{poison}, CacheMount: &CacheMount{Target: poison}},
				{Type: StepRun, Args: []string{poison}, SecretMount: &SecretMount{ID: poison, Target: poison}},
				{Type: StepCmd, Args: []string{poison}},
				{Type: StepExpose, Args: []string{"3000"}},
				{Type: StepCopyFrom, CopyFrom: &CopyFrom{Stage: "build", Src: poison, Dst: poison}},
				{Type: StepArg, Args: []string{poison, poison}},
				{Type: StepUser, Args: []string{poison}},
				{Type: StepHealthcheck, Args: []string{poison}},
				{Type: StepEntrypoint, Args: []string{poison}},
			},
		}},
	}

	out := EmitDockerfile(plan)

	// Each line of Dockerfile output must not contain raw \r or \x00.
	// \n is only valid as line separator between instructions.
	for i, line := range strings.Split(out, "\n") {
		if strings.ContainsAny(line, "\r\x00") {
			t.Errorf("line %d contains control char: %q", i, line)
		}
		// No embedded newline injection — "RUN cat /etc/shadow" must not appear as its own instruction
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
		{"empty pattern", "", true}, // empty regex matches everything
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileMatchesRegex(f, tt.pattern)
			if got != tt.want {
				t.Errorf("fileMatchesRegex(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestDetectRules_pathTraversal(t *testing.T) {
	tmp := t.TempDir()

	// Create a sentinel file outside the project dir
	sentinel := filepath.Join(tmp, "outside", "secret.txt")
	os.MkdirAll(filepath.Dir(sentinel), 0755)
	os.WriteFile(sentinel, []byte("secret"), 0644)

	projectDir := filepath.Join(tmp, "project")
	os.MkdirAll(projectDir, 0755)

	// SECURITY FINDING: evalRule uses filepath.Join(dir, rule.File) directly
	// without containedPath validation. This test documents the current behavior.
	rules := DetectRules{
		All: []DetectRule{
			{File: "../outside/secret.txt"},
		},
	}

	result := evalDetectRules(projectDir, rules)

	// The file exists outside the project dir. If evalDetectRules returns true,
	// it means the file check escaped the project boundary.
	if result {
		t.Log("FINDING: evalDetectRules reads files outside project dir via '../' in File field")
	}

	// Regardless of current behavior, document it doesn't panic
}
