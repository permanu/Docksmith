package emit_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/emit"
	"github.com/permanu/docksmith/plan"
)

func TestExternalToolEmission(t *testing.T) {
	const (
		toolName    = "migrate"
		toolURL     = "https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-${TARGETARCH}.tar.gz"
		toolSHA256  = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
		installPath = "/usr/local/bin"
	)

	fw := &core.Framework{Name: "go", GoVersion: "1.22"}
	p, err := plan.Plan(fw,
		plan.WithStartCommand("./app"),
		plan.WithExternalTools([]config.ExternalTool{
			{
				Name:        toolName,
				URL:         toolURL,
				SHA256:      toolSHA256,
				InstallPath: installPath,
				Stage:       "runtime",
			},
		}),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	df := emit.EmitDockerfile(p)

	if !strings.Contains(df, toolURL) {
		t.Errorf("Dockerfile missing tool URL %q\nDockerfile:\n%s", toolURL, df)
	}
	if !strings.Contains(df, toolSHA256) {
		t.Errorf("Dockerfile missing sha256 %q\nDockerfile:\n%s", toolSHA256, df)
	}
	if !strings.Contains(df, "tar -xzf") {
		t.Errorf("Dockerfile missing tar extract command\nDockerfile:\n%s", df)
	}
	if !strings.Contains(df, "sha256sum -c") {
		t.Errorf("Dockerfile missing sha256sum verification\nDockerfile:\n%s", df)
	}
	if !strings.Contains(df, installPath) {
		t.Errorf("Dockerfile missing install_path %q\nDockerfile:\n%s", installPath, df)
	}
}

func TestExternalToolBuilderStage(t *testing.T) {
	fw := &core.Framework{Name: "go", GoVersion: "1.22"}
	tools := []config.ExternalTool{
		{
			Name:        "mockgen",
			URL:         "https://example.com/mockgen-linux-amd64.tar.gz",
			SHA256:      "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			InstallPath: "/usr/local/bin",
			Stage:       "builder",
		},
	}
	p, err := plan.Plan(fw, plan.WithStartCommand("./app"), plan.WithExternalTools(tools))
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// The tool must appear in the first stage (builder), not the last (runtime).
	if len(p.Stages) < 2 {
		t.Skip("plan has only one stage; builder/runtime distinction N/A")
	}
	firstStage := p.Stages[0]
	found := false
	for _, s := range firstStage.Steps {
		if s.Type == core.StepFetchTool {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected StepFetchTool in first (builder) stage, not found")
	}
	// Must NOT appear in last stage.
	lastStage := p.Stages[len(p.Stages)-1]
	for _, s := range lastStage.Steps {
		if s.Type == core.StepFetchTool {
			t.Errorf("StepFetchTool unexpectedly found in last (runtime) stage")
		}
	}
}

// TestExternalToolFormat_Binary verifies that format="binary" emits curl directly
// to the install path + chmod, with no archive extraction.
func TestExternalToolFormat_Binary(t *testing.T) {
	const (
		toolName    = "atlas"
		toolURL     = "https://release.ariga.io/atlas/atlas-linux-${TARGETARCH}-v1.2.0"
		toolSHA256  = "c12b889e3349f0e5610aec32fe327e5a6911a0e472754a0c381c30c7c0630e88"
		installPath = "/app/bin"
	)

	fw := &core.Framework{Name: "go", GoVersion: "1.22"}
	p, err := plan.Plan(fw,
		plan.WithStartCommand("./app"),
		plan.WithExternalTools([]config.ExternalTool{
			{
				Name:        toolName,
				URL:         toolURL,
				SHA256:      toolSHA256,
				InstallPath: installPath,
				Format:      "binary",
			},
		}),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	df := emit.EmitDockerfile(p)

	if !strings.Contains(df, toolURL) {
		t.Errorf("Dockerfile missing tool URL %q\nDockerfile:\n%s", toolURL, df)
	}
	if !strings.Contains(df, toolSHA256) {
		t.Errorf("Dockerfile missing sha256 %q\nDockerfile:\n%s", toolSHA256, df)
	}
	// Binary format: curl directly to dest, no tar extraction.
	if !strings.Contains(df, "/app/bin/atlas") {
		t.Errorf("Dockerfile missing direct binary dest /app/bin/atlas\nDockerfile:\n%s", df)
	}
	if !strings.Contains(df, "chmod +x") {
		t.Errorf("Dockerfile missing chmod +x for binary format\nDockerfile:\n%s", df)
	}
	if strings.Contains(df, "tar -xzf") {
		t.Errorf("Dockerfile must not contain tar extraction for binary format\nDockerfile:\n%s", df)
	}
	if strings.Contains(df, "unzip") {
		t.Errorf("Dockerfile must not contain unzip for binary format\nDockerfile:\n%s", df)
	}
}

// TestExternalToolFormat_Zip verifies that format="zip" emits unzip extraction.
func TestExternalToolFormat_Zip(t *testing.T) {
	const (
		toolName    = "mytool"
		toolURL     = "https://example.com/mytool-linux-amd64.zip"
		toolSHA256  = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
		installPath = "/usr/local/bin"
	)

	fw := &core.Framework{Name: "go", GoVersion: "1.22"}
	p, err := plan.Plan(fw,
		plan.WithStartCommand("./app"),
		plan.WithExternalTools([]config.ExternalTool{
			{
				Name:        toolName,
				URL:         toolURL,
				SHA256:      toolSHA256,
				InstallPath: installPath,
				Format:      "zip",
			},
		}),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	df := emit.EmitDockerfile(p)

	if !strings.Contains(df, "unzip -o") {
		t.Errorf("Dockerfile missing unzip for zip format\nDockerfile:\n%s", df)
	}
	if strings.Contains(df, "tar -xzf") {
		t.Errorf("Dockerfile must not contain tar extraction for zip format\nDockerfile:\n%s", df)
	}
	if strings.Contains(df, "chmod +x") {
		t.Errorf("Dockerfile must not contain chmod for zip format\nDockerfile:\n%s", df)
	}
	if !strings.Contains(df, toolSHA256) {
		t.Errorf("Dockerfile missing sha256 %q\nDockerfile:\n%s", toolSHA256, df)
	}
}

// TestExternalToolFormat_DefaultIsTargz verifies that omitting format (or
// setting it to "") produces tar.gz behavior identical to the pre-Format behavior.
func TestExternalToolFormat_DefaultIsTargz(t *testing.T) {
	const (
		toolURL    = "https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-${TARGETARCH}.tar.gz"
		toolSHA256 = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	)

	fw := &core.Framework{Name: "go", GoVersion: "1.22"}

	toolNoFormat := config.ExternalTool{
		Name: "migrate", URL: toolURL, SHA256: toolSHA256,
		InstallPath: "/usr/local/bin",
	}
	toolExplicitTargz := config.ExternalTool{
		Name: "migrate", URL: toolURL, SHA256: toolSHA256,
		InstallPath: "/usr/local/bin", Format: "tar.gz",
	}

	planNoFormat, err := plan.Plan(fw, plan.WithStartCommand("./app"), plan.WithExternalTools([]config.ExternalTool{toolNoFormat}))
	if err != nil {
		t.Fatalf("Plan (no format): %v", err)
	}
	planExplicit, err := plan.Plan(fw, plan.WithStartCommand("./app"), plan.WithExternalTools([]config.ExternalTool{toolExplicitTargz}))
	if err != nil {
		t.Fatalf("Plan (explicit tar.gz): %v", err)
	}

	dfNoFormat := emit.EmitDockerfile(planNoFormat)
	dfExplicit := emit.EmitDockerfile(planExplicit)

	if dfNoFormat != dfExplicit {
		t.Errorf("empty format and explicit tar.gz must produce identical Dockerfiles\ngot (no format):\n%s\ngot (explicit):\n%s", dfNoFormat, dfExplicit)
	}
	if !strings.Contains(dfNoFormat, "tar -xzf") {
		t.Errorf("default format must use tar -xzf\nDockerfile:\n%s", dfNoFormat)
	}
}

func TestExternalToolValidation(t *testing.T) {
	cases := []struct {
		name string
		tool config.ExternalTool
		ok   bool
	}{
		{
			name: "valid",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/tool.tar.gz",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
				Stage:       "runtime",
			},
			ok: true,
		},
		{
			name: "valid format binary",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/mytool-linux-amd64",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
				Format:      "binary",
			},
			ok: true,
		},
		{
			name: "valid format zip",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/tool.zip",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
				Format:      "zip",
			},
			ok: true,
		},
		{
			name: "invalid format",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/tool.tar.gz",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
				Format:      "raw",
			},
			ok: false,
		},
		{
			name: "bad name uppercase",
			tool: config.ExternalTool{
				Name:        "MyTool",
				URL:         "https://example.com/tool.tar.gz",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
			},
			ok: false,
		},
		{
			name: "http url",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "http://example.com/tool.tar.gz",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
			},
			ok: false,
		},
		{
			name: "short sha256",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/tool.tar.gz",
				SHA256:      "abc123",
				InstallPath: "/usr/local/bin",
			},
			ok: false,
		},
		{
			name: "relative install path",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/tool.tar.gz",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "usr/local/bin",
			},
			ok: false,
		},
		{
			name: "bad stage",
			tool: config.ExternalTool{
				Name:        "mytool",
				URL:         "https://example.com/tool.tar.gz",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				InstallPath: "/usr/local/bin",
				Stage:       "dev",
			},
			ok: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Runtime:       "go",
				Start:         config.StartConfig{Command: "./app"},
				ExternalTools: []config.ExternalTool{tc.tool},
			}
			err := cfg.Validate()
			if tc.ok && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("expected validation error, got nil")
			}
		})
	}
}
