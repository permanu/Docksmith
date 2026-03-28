package docksmith

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// LoadConfig — YAML parsing
// ---------------------------------------------------------------------------

func TestLoadConfig_YAML_FullFields(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config, want non-nil")
	}
	if cfg.Runtime != "node" {
		t.Errorf("Runtime = %q, want %q", cfg.Runtime, "node")
	}
	if cfg.Version != "20" {
		t.Errorf("Version = %q, want %q", cfg.Version, "20")
	}
	if cfg.Build != "npm run build" {
		t.Errorf("Build = %q, want %q", cfg.Build, "npm run build")
	}
	if cfg.Start != "node server.js" {
		t.Errorf("Start = %q, want %q", cfg.Start, "node server.js")
	}
	if cfg.Port != 4000 {
		t.Errorf("Port = %d, want 4000", cfg.Port)
	}
	if cfg.Env["NODE_ENV"] != "production" {
		t.Errorf("Env[NODE_ENV] = %q, want %q", cfg.Env["NODE_ENV"], "production")
	}
	if cfg.Env["LOG_LEVEL"] != "info" {
		t.Errorf("Env[LOG_LEVEL] = %q, want %q", cfg.Env["LOG_LEVEL"], "info")
	}
	if len(cfg.SystemDeps) != 2 || cfg.SystemDeps[0] != "libssl-dev" {
		t.Errorf("SystemDeps = %v, want [libssl-dev curl]", cfg.SystemDeps)
	}
}

// ---------------------------------------------------------------------------
// LoadConfig — JSON parsing
// ---------------------------------------------------------------------------

func TestLoadConfig_JSON_FullFields(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil config, want non-nil")
	}
	if cfg.Runtime != "python" {
		t.Errorf("Runtime = %q, want %q", cfg.Runtime, "python")
	}
	if cfg.Version != "3.11" {
		t.Errorf("Version = %q, want %q", cfg.Version, "3.11")
	}
	if cfg.Start != "gunicorn app:app" {
		t.Errorf("Start = %q, want %q", cfg.Start, "gunicorn app:app")
	}
	if cfg.Port != 5000 {
		t.Errorf("Port = %d, want 5000", cfg.Port)
	}
	if cfg.Env["PYTHONDONTWRITEBYTECODE"] != "1" {
		t.Errorf("Env[PYTHONDONTWRITEBYTECODE] = %q, want %q", cfg.Env["PYTHONDONTWRITEBYTECODE"], "1")
	}
	if len(cfg.SystemDeps) != 1 || cfg.SystemDeps[0] != "libpq-dev" {
		t.Errorf("SystemDeps = %v, want [libpq-dev]", cfg.SystemDeps)
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func TestLoadConfig_MissingRuntime_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	content := "start: node index.js\n"
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), content)

	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for missing runtime, got nil")
	}
}

func TestLoadConfig_InvalidRuntime_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	content := "runtime: fortran\nstart: ./run\n"
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), content)

	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for invalid runtime, got nil")
	}
}

func TestLoadConfig_MissingStart_NonStatic_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	content := "runtime: go\nversion: \"1.22\"\n"
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), content)

	_, err := LoadConfig(dir)
	if err == nil {
		t.Fatal("want error for missing start command on non-static runtime, got nil")
	}
}

func TestLoadConfig_Static_NoStartRequired(t *testing.T) {
	dir := t.TempDir()
	content := "runtime: static\n"
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), content)

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error for static runtime without start: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil, want config")
	}
}

// ---------------------------------------------------------------------------
// Dockerfile mode
// ---------------------------------------------------------------------------

func TestLoadConfig_DockerfileMode_NoRuntimeRequired(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-dockerfile-mode"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil, want config")
	}
	if cfg.Dockerfile != "./Dockerfile.prod" {
		t.Errorf("Dockerfile = %q, want %q", cfg.Dockerfile, "./Dockerfile.prod")
	}
}

// ---------------------------------------------------------------------------
// Default ports
// ---------------------------------------------------------------------------

func TestLoadConfig_DefaultPorts(t *testing.T) {
	cases := []struct {
		runtime string
		want    int
	}{
		{"node", 3000},
		{"python", 8000},
		{"go", 8080},
		{"php", 80},
		{"java", 8080},
		{"dotnet", 8080},
		{"rust", 8080},
		{"ruby", 3000},
		{"elixir", 4000},
		{"deno", 8000},
		{"bun", 3000},
		{"static", 80},
	}

	startCmds := map[string]string{
		"node":   "node index.js",
		"python": "gunicorn app:app",
		"go":     "go run .",
		"php":    "php -S 0.0.0.0:80",
		"java":   "java -jar app.jar",
		"dotnet": "dotnet run",
		"rust":   "./target/release/app",
		"ruby":   "bundle exec puma",
		"elixir": "mix phx.server",
		"deno":   "deno run --allow-net main.ts",
		"bun":    "bun run index.ts",
		"static": "", // no start required
	}

	for _, tc := range cases {
		t.Run(tc.runtime, func(t *testing.T) {
			dir := t.TempDir()
			var content string
			if startCmds[tc.runtime] != "" {
				content = "runtime: " + tc.runtime + "\nstart: " + startCmds[tc.runtime] + "\n"
			} else {
				content = "runtime: " + tc.runtime + "\n"
			}
			mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), content)

			cfg, err := LoadConfig(dir)
			if err != nil {
				t.Fatalf("LoadConfig error: %v", err)
			}
			if cfg.Port != tc.want {
				t.Errorf("runtime=%q Port = %d, want %d", tc.runtime, cfg.Port, tc.want)
			}
		})
	}
}

// port override takes precedence over default
func TestLoadConfig_PortOverride_Respected(t *testing.T) {
	dir := t.TempDir()
	content := "runtime: node\nstart: node server.js\nport: 9090\n"
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), content)

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
}

// ---------------------------------------------------------------------------
// ToFramework — all 12 runtimes
// ---------------------------------------------------------------------------

func TestConfig_ToFramework_AllRuntimes(t *testing.T) {
	cases := []struct {
		runtime    string
		wantName   string
		start      string
		versionKey string // which Framework field to verify version on
		version    string
	}{
		{"node", "express", "node index.js", "node", "18"},
		{"python", "flask", "gunicorn app:app", "python", "3.11"},
		{"go", "go-std", "go run .", "go", "1.22"},
		{"php", "php", "php -S 0.0.0.0:80", "php", "8.2"},
		{"java", "maven", "java -jar app.jar", "java", "17"},
		{"dotnet", "aspnet-core", "dotnet run", "dotnet", "8.0"},
		{"rust", "rust-generic", "./target/release/app", "", ""},
		{"ruby", "rails", "bundle exec puma", "", ""},
		{"elixir", "elixir-phoenix", "mix phx.server", "", ""},
		{"deno", "deno", "deno run main.ts", "deno", "1.40"},
		{"bun", "bun", "bun run index.ts", "bun", "1.0"},
		{"static", "static", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.runtime, func(t *testing.T) {
			cfg := &Config{
				Runtime: tc.runtime,
				Start:   tc.start,
				Version: tc.version,
				Port:    8080,
			}
			if tc.runtime == "bun" {
				cfg.PackageManager = "bun"
			}

			fw := cfg.ToFramework()
			if fw == nil {
				t.Fatal("got nil Framework")
			}
			if fw.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", fw.Name, tc.wantName)
			}

			// verify version field is populated when expected
			switch tc.versionKey {
			case "node":
				if fw.NodeVersion != tc.version {
					t.Errorf("NodeVersion = %q, want %q", fw.NodeVersion, tc.version)
				}
			case "python":
				if fw.PythonVersion != tc.version {
					t.Errorf("PythonVersion = %q, want %q", fw.PythonVersion, tc.version)
				}
			case "go":
				if fw.GoVersion != tc.version {
					t.Errorf("GoVersion = %q, want %q", fw.GoVersion, tc.version)
				}
			case "php":
				if fw.PHPVersion != tc.version {
					t.Errorf("PHPVersion = %q, want %q", fw.PHPVersion, tc.version)
				}
			case "java":
				if fw.JavaVersion != tc.version {
					t.Errorf("JavaVersion = %q, want %q", fw.JavaVersion, tc.version)
				}
			case "dotnet":
				if fw.DotnetVersion != tc.version {
					t.Errorf("DotnetVersion = %q, want %q", fw.DotnetVersion, tc.version)
				}
			case "deno":
				if fw.DenoVersion != tc.version {
					t.Errorf("DenoVersion = %q, want %q", fw.DenoVersion, tc.version)
				}
			case "bun":
				if fw.BunVersion != tc.version {
					t.Errorf("BunVersion = %q, want %q", fw.BunVersion, tc.version)
				}
			}
		})
	}
}

func TestConfig_ToFramework_GoDefaultBuildCommand(t *testing.T) {
	cfg := &Config{Runtime: "go", Start: "go run .", Port: 8080}
	fw := cfg.ToFramework()
	if fw.BuildCommand != "go build -o app ." {
		t.Errorf("BuildCommand = %q, want %q", fw.BuildCommand, "go build -o app .")
	}
}

func TestConfig_ToFramework_GoCustomBuildCommand_NotOverridden(t *testing.T) {
	cfg := &Config{Runtime: "go", Build: "make build", Start: "go run .", Port: 8080}
	fw := cfg.ToFramework()
	if fw.BuildCommand != "make build" {
		t.Errorf("BuildCommand = %q, want %q", fw.BuildCommand, "make build")
	}
}

func TestConfig_ToFramework_DockerfileMode(t *testing.T) {
	cfg := &Config{Dockerfile: "./Dockerfile.custom"}
	fw := cfg.ToFramework()
	if fw.Name != "dockerfile" {
		t.Errorf("Name = %q, want %q", fw.Name, "dockerfile")
	}
	if fw.OutputDir != "./Dockerfile.custom" {
		t.Errorf("OutputDir = %q, want %q", fw.OutputDir, "./Dockerfile.custom")
	}
}

// ---------------------------------------------------------------------------
// Config file priority: docksmith.toml > docksmith.yaml
// ---------------------------------------------------------------------------

func TestLoadConfig_Priority_TomlBeforeYAML(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-priority-toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil, want config")
	}
	// toml has runtime=go, yaml has runtime=node — toml wins
	if cfg.Runtime != "go" {
		t.Errorf("Runtime = %q, want %q (toml should win over yaml)", cfg.Runtime, "go")
	}
}

func TestLoadConfig_Priority_YAMLBeforeYML(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), "runtime: python\nstart: gunicorn app:app\n")
	mustWriteFile(t, filepath.Join(dir, "docksmith.yml"), "runtime: node\nstart: node index.js\n")

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Runtime != "python" {
		t.Errorf("Runtime = %q, want %q (yaml before yml)", cfg.Runtime, "python")
	}
}

func TestLoadConfig_Priority_YMLBeforeJSON(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "docksmith.yml"), "runtime: python\nstart: gunicorn app:app\n")
	mustWriteFile(t, filepath.Join(dir, "docksmith.json"), `{"runtime":"node","start":"node index.js"}`)

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Runtime != "python" {
		t.Errorf("Runtime = %q, want %q (yml before json)", cfg.Runtime, "python")
	}
}

func TestLoadConfig_Priority_JSONBeforeDotYAML(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "docksmith.json"), `{"runtime":"node","start":"node index.js"}`)
	mustWriteFile(t, filepath.Join(dir, ".docksmith.yaml"), "runtime: python\nstart: gunicorn app:app\n")

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Runtime != "node" {
		t.Errorf("Runtime = %q, want %q (json before .docksmith.yaml)", cfg.Runtime, "node")
	}
}

// ---------------------------------------------------------------------------
// Missing config file: returns nil, nil
// ---------------------------------------------------------------------------

func TestLoadConfig_MissingFile_ReturnsNilNil(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("want nil error, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("want nil config for empty dir, got %+v", cfg)
	}
}

// ---------------------------------------------------------------------------
// DetectOptions.ConfigFileNames custom names
// ---------------------------------------------------------------------------

func TestDetectWithOptions_ConfigFileNames_CustomName(t *testing.T) {
	cfg, err := loadConfigWithNames(
		filepath.Join("testdata", "fixtures", "config-custom-name"),
		[]string{"deploy.yaml"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil, want config")
	}
	if cfg.Runtime != "ruby" {
		t.Errorf("Runtime = %q, want %q", cfg.Runtime, "ruby")
	}
}

func TestDetectWithOptions_ConfigFileNames_SkipsDefaultNames(t *testing.T) {
	dir := t.TempDir()
	// write a docksmith.yaml but custom names list doesn't include it
	mustWriteFile(t, filepath.Join(dir, "docksmith.yaml"), "runtime: node\nstart: node index.js\n")

	cfg, err := loadConfigWithNames(dir, []string{"myapp.toml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// docksmith.yaml exists but wasn't in custom list → should not find it
	if cfg != nil {
		t.Errorf("want nil config when custom names don't match existing files, got %+v", cfg)
	}
}

func TestDetectWithOptions_CustomConfigNames_WiredIntoDetect(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "config-custom-name")
	fw, err := DetectWithOptions(dir, DetectOptions{ConfigFileNames: []string{"deploy.yaml"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil framework")
	}
	// ruby maps to "rails"
	if fw.Name != "rails" {
		t.Errorf("Name = %q, want %q", fw.Name, "rails")
	}
}

func TestDetect_ConfigTakesPriorityOverAutoDetection(t *testing.T) {
	// config-yaml has node runtime + a docksmith.yaml, but no package.json
	// so auto-detection would fall through to static.
	// With config loading, it should return the node runtime.
	dir := filepath.Join("testdata", "fixtures", "config-yaml")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil framework")
	}
	// node maps to "express"
	if fw.Name != "express" {
		t.Errorf("Name = %q, want %q (config should take priority)", fw.Name, "express")
	}
}

// ---------------------------------------------------------------------------
// TOML parsing
// ---------------------------------------------------------------------------

func TestLoadConfig_TOML_BasicFields(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "fixtures", "config-priority-toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("got nil, want config")
	}
	if cfg.Runtime != "go" {
		t.Errorf("Runtime = %q, want go", cfg.Runtime)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
