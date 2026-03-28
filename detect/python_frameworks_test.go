package detect

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFastAPI_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi>=0.100\nuvicorn\n")

	fw := detectFastAPI(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "fastapi" {
		t.Errorf("Name = %q, want %q", fw.Name, "fastapi")
	}
	if fw.Port != 8000 {
		t.Errorf("Port = %d, want 8000", fw.Port)
	}
	if fw.StartCommand != "uvicorn main:app --host 0.0.0.0 --port 8000" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
	if fw.PythonVersion == "" {
		t.Error("PythonVersion is empty")
	}
}

func TestDetectFastAPI_PyprojectToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]
dependencies = ["fastapi>=0.100", "uvicorn"]
`)
	fw := detectFastAPI(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "fastapi" {
		t.Errorf("Name = %q, want %q", fw.Name, "fastapi")
	}
}

func TestDetectFastAPI_NoFastAPIInDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0\n")
	if fw := detectFastAPI(dir); fw != nil {
		t.Errorf("got %q, want nil without fastapi dep", fw.Name)
	}
}

func TestDetectFastAPI_NoDepsFile(t *testing.T) {
	dir := t.TempDir()
	if fw := detectFastAPI(dir); fw != nil {
		t.Errorf("got %q, want nil without deps file", fw.Name)
	}
}

func TestDetectFastAPI_WithUVPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi>=0.100\n")
	writeFile(t, dir, "uv.lock", "")
	fw := detectFastAPI(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.PythonPM != "uv" {
		t.Errorf("PythonPM = %q, want %q", fw.PythonPM, "uv")
	}
	if fw.BuildCommand != "pip install uv && uv sync --frozen" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectFlask_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0\ngunicorn\n")

	fw := detectFlask(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "flask" {
		t.Errorf("Name = %q, want %q", fw.Name, "flask")
	}
	if fw.Port != 5000 {
		t.Errorf("Port = %d, want 5000", fw.Port)
	}
	if fw.StartCommand != "gunicorn --bind 0.0.0.0:5000 app:app" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectFlask_PyprojectToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]
dependencies = ["flask>=3.0"]
`)
	fw := detectFlask(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "flask" {
		t.Errorf("Name = %q, want %q", fw.Name, "flask")
	}
}

func TestDetectFlask_NoFlaskInDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "django>=4.0\n")
	if fw := detectFlask(dir); fw != nil {
		t.Errorf("got %q, want nil without flask dep", fw.Name)
	}
}

func TestDetectFlask_NoDepsFile(t *testing.T) {
	dir := t.TempDir()
	if fw := detectFlask(dir); fw != nil {
		t.Errorf("got %q, want nil without deps file", fw.Name)
	}
}

func TestDetectFlask_WithPythonVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".python-version", "3.12.0")
	writeFile(t, dir, "requirements.txt", "flask==3.0\n")
	fw := detectFlask(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.PythonVersion != "3.12" {
		t.Errorf("PythonVersion = %q, want %q", fw.PythonVersion, "3.12")
	}
}

func TestDetect_FastAPIViaFixture(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "python-fastapi")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "fastapi" {
		t.Errorf("Name = %q, want %q", fw.Name, "fastapi")
	}
}

func TestDetect_FlaskViaFixture(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "python-flask")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "flask" {
		t.Errorf("Name = %q, want %q", fw.Name, "flask")
	}
}

// contains is a substring test helper used in Django Procfile assertions.
func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
