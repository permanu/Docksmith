package remotedetect_test

import (
	"testing"

	"github.com/permanu/docksmith/remotedetect"
)

func TestDetect_GoAppWithDockerfile(t *testing.T) {
	// permanu/Deploy pattern: Dockerfile at root, go.mod in backend/
	paths := []string{
		"Dockerfile",
		"backend/go.mod",
		"backend/cmd/server/main.go",
		"backend/internal/api/handlers/apps.go",
		"README.md",
		".gitignore",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "go" {
		t.Errorf("expected framework=go, got %s", r.Framework)
	}
	if !r.HasDockerfile {
		t.Error("expected HasDockerfile=true")
	}
}

func TestDetect_PythonWithDockerfile(t *testing.T) {
	// permanu/AML-Monorepo pattern: Dockerfile at root, Python in subdirectory
	paths := []string{
		"Dockerfile",
		"backend/requirements.txt",
		"backend/main.py",
		"backend/app.py",
		"README.md",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "fastapi" {
		t.Errorf("expected framework=fastapi, got %s", r.Framework)
	}
}

func TestDetect_NodejsOnly(t *testing.T) {
	paths := []string{
		"package.json",
		"src/app/page.tsx",
		"next.config.mjs",
		"tsconfig.json",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "nextjs" {
		t.Errorf("expected framework=nextjs, got %s", r.Framework)
	}
}

func TestDetect_DockerfileOnly(t *testing.T) {
	paths := []string{
		"Dockerfile",
		"README.md",
		".gitignore",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "docker" {
		t.Errorf("expected framework=docker, got %s", r.Framework)
	}
}

func TestDetect_EmptyRepo(t *testing.T) {
	r := remotedetect.Detect(nil)
	if r.Framework != "unknown" {
		t.Errorf("expected framework=unknown, got %s", r.Framework)
	}
}

func TestDetect_GoAtRoot(t *testing.T) {
	paths := []string{
		"go.mod",
		"main.go",
		"cmd/server/main.go",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "go" {
		t.Errorf("expected framework=go, got %s", r.Framework)
	}
}

func TestDetect_Rust(t *testing.T) {
	paths := []string{
		"Cargo.toml",
		"src/main.rs",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "rust" {
		t.Errorf("expected framework=rust, got %s", r.Framework)
	}
}

func TestDetect_Django(t *testing.T) {
	paths := []string{
		"manage.py",
		"requirements.txt",
		"myapp/wsgi.py",
	}
	r := remotedetect.Detect(paths)
	if r.Framework != "django" {
		t.Errorf("expected framework=django, got %s", r.Framework)
	}
}

func TestDetect_MultipleServices(t *testing.T) {
	paths := []string{
		"Dockerfile",
		"backend/go.mod",
		"backend/cmd/server/main.go",
		"frontend/package.json",
		"frontend/src/index.ts",
	}
	r := remotedetect.Detect(paths)
	// Should prefer language-specific framework over dockerfile
	if r.Framework != "go" {
		t.Errorf("expected framework=go (first non-dockerfile), got %s", r.Framework)
	}
	if len(r.Services) < 2 {
		t.Errorf("expected at least 2 services, got %d", len(r.Services))
	}
}

func TestDetect_HasDockerCompose(t *testing.T) {
	paths := []string{
		"package.json",
		"docker-compose.yml",
	}
	r := remotedetect.Detect(paths)
	if !r.HasDockerCompose {
		t.Error("expected HasDockerCompose=true")
	}
}
