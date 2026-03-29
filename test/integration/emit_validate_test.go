package integration_test

import (
	"testing"

	"github.com/permanu/docksmith"
)

// Go and Rust use distroless — no shell for healthcheck, no adduser.
var distrolessSkips = map[string]bool{
	"no HEALTHCHECK in final stage":                        true,
	"no USER instruction in final stage (running as root)": true,
}

// TestEmitValidDockerfile runs detect -> plan -> emit for every runtime family
// and validates the emitted Dockerfile for syntax, hardening, and security.
func TestEmitValidDockerfile(t *testing.T) {
	fixtureTests := []struct {
		name    string
		fixture string
	}{
		{"node", "node-nextjs"},
		{"python", "python-django"},
		{"go", "go-std-root"},
		{"ruby", "rails"},
		{"php", "laravel"},
		{"rust", "rust-actix"},
		{"elixir", "elixir-phoenix"},
		{"deno", "deno-plain"},
	}

	skips := map[string]map[string]bool{
		"go":   distrolessSkips,
		"rust": distrolessSkips,
	}

	for _, tt := range fixtureTests {
		t.Run(tt.name, func(t *testing.T) {
			dockerfile := mustBuildFixture(t, tt.fixture)
			assertDockerfileValid(t, dockerfile, tt.name, skips[tt.name])
		})
	}

	runSyntheticRuntimes(t)
}

func mustBuildFixture(t *testing.T, fixture string) string {
	t.Helper()
	dockerfile, _, err := docksmith.Build("../../testdata/fixtures/" + fixture)
	if err != nil {
		t.Fatalf("Build(%s): %v", fixture, err)
	}
	if dockerfile == "" {
		t.Fatalf("Build(%s) produced empty Dockerfile", fixture)
	}
	return dockerfile
}

// runSyntheticRuntimes tests runtimes that lack testdata fixtures.
func runSyntheticRuntimes(t *testing.T) {
	t.Run("java", func(t *testing.T) {
		fw := &docksmith.Framework{
			Name: "spring-boot", JavaVersion: "21", Port: 8080,
		}
		assertDockerfileValid(t, mustEmit(t, fw), "java", nil)
	})

	t.Run("dotnet", func(t *testing.T) {
		fw := &docksmith.Framework{
			Name: "aspnet-core", DotnetVersion: "8.0", Port: 5000,
		}
		assertDockerfileValid(t, mustEmit(t, fw), "dotnet", nil)
	})

	t.Run("bun", func(t *testing.T) {
		fw := &docksmith.Framework{
			Name: "bun", BunVersion: "1", PackageManager: "bun",
			Port: 3000, StartCommand: "bun run index.ts",
		}
		assertDockerfileValid(t, mustEmit(t, fw), "bun", nil)
	})

	t.Run("static", func(t *testing.T) {
		fw := &docksmith.Framework{
			Name: "static", OutputDir: "public", Port: 0,
		}
		// nginx serves from /usr/share/nginx/html — no WORKDIR needed.
		staticSkips := map[string]bool{
			"no WORKDIR set in final stage": true,
		}
		assertDockerfileValid(t, mustEmit(t, fw), "static", staticSkips)
	})
}

func mustEmit(t *testing.T, fw *docksmith.Framework) string {
	t.Helper()
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatalf("Plan(%s): %v", fw.Name, err)
	}
	out := docksmith.EmitDockerfile(p)
	if out == "" {
		t.Fatalf("EmitDockerfile(%s) produced empty output", fw.Name)
	}
	return out
}
