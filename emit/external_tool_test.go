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
