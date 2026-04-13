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
	if fw.StartCommand != "gunicorn main:app --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} -k uvicorn.workers.UvicornWorker" {
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
	if fw.Port != 8000 {
		t.Errorf("Port = %d, want 8000", fw.Port)
	}
	if fw.StartCommand != "gunicorn app:app --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2" {
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

func TestDetectPythonAppTarget_FastAPI(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "server.py", "from fastapi import FastAPI\n\napi = FastAPI(title=\"MyApp\")\n")
	target := detectPythonAppTarget(dir, "FastAPI(")
	if target != "server:api" {
		t.Errorf("got %q, want %q", target, "server:api")
	}
}

func TestDetectPythonAppTarget_Flask(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "application.py", "from flask import Flask\n\napp = Flask(__name__)\n")
	target := detectPythonAppTarget(dir, "Flask(")
	if target != "application:app" {
		t.Errorf("got %q, want %q", target, "application:app")
	}
}

func TestDetectPythonAppTarget_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "utils.py", "import os\n")
	target := detectPythonAppTarget(dir, "FastAPI(")
	if target != "" {
		t.Errorf("got %q, want empty string", target)
	}
}

func TestDetectPythonAppTarget_SkipsSubdirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nested/main.py", "app = FastAPI()\n")
	target := detectPythonAppTarget(dir, "FastAPI(")
	if target != "" {
		t.Errorf("got %q, want empty (should not scan subdirs)", target)
	}
}

func TestDetectFastAPI_InfersAppTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi>=0.100\nuvicorn\n")
	writeFile(t, dir, "server.py", "from fastapi import FastAPI\napi = FastAPI()\n")
	fw := detectFastAPI(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if !contains(fw.StartCommand, "server:api") {
		t.Errorf("StartCommand should use detected target, got %q", fw.StartCommand)
	}
}

func TestDetectFlask_InfersAppTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0\n")
	writeFile(t, dir, "wsgi.py", "from myapp import create_app\napp = Flask(__name__)\n")
	fw := detectFlask(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if !contains(fw.StartCommand, "wsgi:app") {
		t.Errorf("StartCommand should use detected target, got %q", fw.StartCommand)
	}
}

// contains is a substring test helper used in Django Procfile assertions.
func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
