package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
)

func TestGenerateDockerignore_basePatterns(t *testing.T) {
	runtimes := []*docksmith.Framework{
		{Name: "nextjs"},
		{Name: "django"},
		{Name: "go"},
		{Name: "rails"},
		{Name: "laravel"},
		{Name: "spring-boot"},
		{Name: "rust"},
		{Name: "static"},
		{Name: "aspnet-core"},
	}

	mustContain := []string{".git", ".env", "Dockerfile", ".dockerignore", "docker-compose*.yml"}

	for _, fw := range runtimes {
		got := docksmith.GenerateDockerignore(fw)
		for _, p := range mustContain {
			if !strings.Contains(got, p) {
				t.Errorf("framework %q: missing base pattern %q", fw.Name, p)
			}
		}
	}
}

func TestGenerateDockerignore_node(t *testing.T) {
	nodeFrameworks := []string{"nextjs", "nuxt", "express", "fastify", "nestjs", "sveltekit"}
	mustContain := []string{"node_modules", ".next", "dist", "build", ".cache", "coverage"}

	for _, name := range nodeFrameworks {
		got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: name})
		for _, p := range mustContain {
			if !strings.Contains(got, p) {
				t.Errorf("framework %q: missing node pattern %q", name, p)
			}
		}
	}
}

func TestGenerateDockerignore_python(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "django"})
	for _, p := range []string{"__pycache__", "*.pyc", ".venv", "venv", ".pytest_cache", ".mypy_cache", "*.egg-info"} {
		if !strings.Contains(got, p) {
			t.Errorf("django: missing python pattern %q", p)
		}
	}
}

func TestGenerateDockerignore_go(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "go"})
	for _, p := range []string{"vendor", "*.test", "*.out"} {
		if !strings.Contains(got, p) {
			t.Errorf("go: missing go pattern %q", p)
		}
	}
}

func TestGenerateDockerignore_ruby(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "rails"})
	for _, p := range []string{".bundle", "vendor/bundle", "log", "tmp"} {
		if !strings.Contains(got, p) {
			t.Errorf("rails: missing ruby pattern %q", p)
		}
	}
}

func TestGenerateDockerignore_php(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "laravel"})
	for _, p := range []string{"vendor", "storage/logs", "bootstrap/cache"} {
		if !strings.Contains(got, p) {
			t.Errorf("laravel: missing php pattern %q", p)
		}
	}
}

func TestGenerateDockerignore_java(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "spring-boot"})
	for _, p := range []string{"target", "build", ".gradle", "*.class", "*.jar"} {
		if !strings.Contains(got, p) {
			t.Errorf("spring-boot: missing java pattern %q", p)
		}
	}
}

func TestGenerateDockerignore_rust(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "rust"})
	if !strings.Contains(got, "target") {
		t.Error("rust: missing 'target' pattern")
	}
}

func TestGenerateDockerignore_static(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "static"})
	if strings.Contains(got, "node_modules") {
		t.Error("static: should not contain node_modules")
	}
	if strings.Contains(got, "__pycache__") {
		t.Error("static: should not contain __pycache__")
	}
}

func TestGenerateDockerignore_eachLineTerminated(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "nextjs"})
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) == 0 {
		t.Fatal("empty dockerignore output")
	}
	for i, line := range lines {
		if line == "" {
			t.Errorf("unexpected blank line at index %d", i)
		}
	}
}

func TestGenerateDockerignore_bun(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "bun"})
	if !strings.Contains(got, "node_modules") {
		t.Error("bun: missing node_modules")
	}
}

func TestGenerateDockerignore_dotnet(t *testing.T) {
	got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: "aspnet-core"})
	for _, p := range []string{"bin", "obj"} {
		if !strings.Contains(got, p) {
			t.Errorf("aspnet-core: missing dotnet pattern %q", p)
		}
	}
}
