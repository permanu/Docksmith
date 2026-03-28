package docksmith_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/permanu/docksmith"
)

func TestBuild_nextjs(t *testing.T) {
	dockerfile, fw, err := docksmith.Build("testdata/fixtures/node-nextjs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil || fw.Name != "nextjs" {
		t.Fatalf("want framework nextjs, got %v", fw)
	}
	if dockerfile == "" {
		t.Fatal("expected non-empty Dockerfile")
	}
	if !strings.Contains(dockerfile, "FROM") {
		t.Error("Dockerfile missing FROM instruction")
	}
}

func TestBuild_pythonDjango(t *testing.T) {
	dockerfile, fw, err := docksmith.Build("testdata/fixtures/python-django")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil || fw.Name != "django" {
		t.Fatalf("want framework django, got %v", fw)
	}
	if !strings.Contains(dockerfile, "python") {
		t.Error("Dockerfile missing python base image")
	}
}

func TestBuild_goStdRoot(t *testing.T) {
	dockerfile, fw, err := docksmith.Build("testdata/fixtures/go-std-root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil || !strings.HasPrefix(fw.Name, "go") {
		t.Fatalf("want go framework, got %v", fw)
	}
	if !strings.Contains(dockerfile, "golang") {
		t.Error("Dockerfile missing golang base image")
	}
}

func TestBuild_emptyDir(t *testing.T) {
	_, _, err := docksmith.Build("testdata/fixtures/empty-dir")
	if err == nil {
		t.Fatal("expected error for empty dir, got nil")
	}
	if !errors.Is(err, docksmith.ErrNotDetected) {
		t.Errorf("error = %v, want ErrNotDetected", err)
	}
}

func TestBuild_withDockerfile(t *testing.T) {
	dockerfile, fw, err := docksmith.Build("testdata/fixtures/with-dockerfile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil || fw.Name != "dockerfile" {
		t.Fatalf("want dockerfile framework, got %v", fw)
	}
	if dockerfile != "" {
		t.Errorf("expected empty string for dockerfile framework, got %q", dockerfile)
	}
}

func TestGenerateDockerfile_backwardCompat(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "nextjs",
		Port:         3000,
		NodeVersion:  "22",
		BuildCommand: "npm run build",
		StartCommand: "node server.js",
	}
	got, err := docksmith.GenerateDockerfile(fw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty Dockerfile")
	}
	if !strings.Contains(got, "FROM") {
		t.Error("Dockerfile missing FROM")
	}
}

func TestGenerateDockerfile_dockerframeReturnsEmpty(t *testing.T) {
	fw := &docksmith.Framework{Name: "dockerfile", Port: 8080}
	got, err := docksmith.GenerateDockerfile(fw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for dockerfile framework, got %q", got)
	}
}

func TestGenerateDockerfile_nilReturnsEmpty(t *testing.T) {
	got, err := docksmith.GenerateDockerfile(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for nil framework, got %q", got)
	}
}

func TestBuildWithOptions_customConfig(t *testing.T) {
	opts := docksmith.DetectOptions{}
	dockerfile, fw, err := docksmith.BuildWithOptions("testdata/fixtures/node-nextjs", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw.Name != "nextjs" {
		t.Errorf("want nextjs, got %q", fw.Name)
	}
	if dockerfile == "" {
		t.Fatal("expected non-empty Dockerfile")
	}
}
