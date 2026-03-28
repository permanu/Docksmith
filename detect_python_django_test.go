package docksmith

import (
	"path/filepath"
	"testing"
)

func TestDetectDjango_Basic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "")
	writeFile(t, dir, "requirements.txt", "Django>=4.2\n")

	fw := detectDjango(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "django" {
		t.Errorf("Name = %q, want %q", fw.Name, "django")
	}
	if fw.Port != 8000 {
		t.Errorf("Port = %d, want 8000", fw.Port)
	}
	if fw.PythonVersion == "" {
		t.Error("PythonVersion is empty")
	}
	if fw.PythonPM != "pip" {
		t.Errorf("PythonPM = %q, want %q", fw.PythonPM, "pip")
	}
	if fw.BuildCommand == "" {
		t.Error("BuildCommand is empty")
	}
	if fw.StartCommand == "" {
		t.Error("StartCommand is empty")
	}
}

func TestDetectDjango_NoManagePy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "Django>=4.2\n")
	if fw := detectDjango(dir); fw != nil {
		t.Errorf("got %q, want nil without manage.py", fw.Name)
	}
}

func TestDetectDjango_NoDepsFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "")
	if fw := detectDjango(dir); fw != nil {
		t.Errorf("got %q, want nil without deps file", fw.Name)
	}
}

func TestDetectDjango_WithPyproject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "")
	writeFile(t, dir, "pyproject.toml", `[project]
dependencies = ["django>=4.2"]
requires-python = ">=3.11"
`)
	fw := detectDjango(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.PythonVersion != "3.11" {
		t.Errorf("PythonVersion = %q, want %q", fw.PythonVersion, "3.11")
	}
}

func TestDetectDjango_WithPoetryLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "")
	writeFile(t, dir, "requirements.txt", "django\n")
	writeFile(t, dir, "poetry.lock", "")
	fw := detectDjango(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.PythonPM != "poetry" {
		t.Errorf("PythonPM = %q, want %q", fw.PythonPM, "poetry")
	}
}

func TestDetectDjango_WithUVLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "")
	writeFile(t, dir, "requirements.txt", "django\n")
	writeFile(t, dir, "uv.lock", "")
	fw := detectDjango(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.PythonPM != "uv" {
		t.Errorf("PythonPM = %q, want %q", fw.PythonPM, "uv")
	}
}

func TestDetectDjango_WithSystemDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "")
	writeFile(t, dir, "requirements.txt", "django\npsycopg2>=2.9\n")
	fw := detectDjango(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	found := false
	for _, d := range fw.SystemDeps {
		if d == "libpq-dev" {
			found = true
		}
	}
	if !found {
		t.Errorf("SystemDeps = %v, want to contain libpq-dev", fw.SystemDeps)
	}
}

func TestDetectDjangoStartCommand_Procfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Procfile", "web: gunicorn myapp.wsgi --bind 0.0.0.0:$PORT\n")
	cmd := detectDjangoStartCommand(dir)
	if cmd == "" {
		t.Fatal("got empty start command")
	}
	if cmd != "gunicorn myapp.wsgi --bind 0.0.0.0:$PORT" {
		t.Errorf("StartCommand = %q", cmd)
	}
}

func TestDetectDjangoStartCommand_ProcfileIPv6Rewrite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Procfile", "web: gunicorn myapp.wsgi --bind [::]:${PORT:-8000}\n")
	cmd := detectDjangoStartCommand(dir)
	if cmd == "" {
		t.Fatal("got empty start command")
	}
	if cmd != "gunicorn myapp.wsgi --bind 0.0.0.0:${PORT:-8000}" {
		t.Errorf("StartCommand = %q, expected IPv6 bind rewritten to IPv4", cmd)
	}
}

func TestDetectDjangoStartCommand_ProcfileGunicornConfRemoved(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Procfile", "web: gunicorn --config gunicorn.conf.py myapp.wsgi\n")
	cmd := detectDjangoStartCommand(dir)
	if cmd == "" {
		t.Fatal("got empty start command")
	}
	if contains(cmd, "--config gunicorn.conf.py") {
		t.Errorf("StartCommand should not contain --config gunicorn.conf.py, got %q", cmd)
	}
	if !contains(cmd, "--bind 0.0.0.0:${PORT:-8000}") {
		t.Errorf("StartCommand should contain --bind, got %q", cmd)
	}
}

func TestDetectDjangoStartCommand_ProcfileNonGunicorn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Procfile", "web: python manage.py runserver 0.0.0.0:8000\n")
	cmd := detectDjangoStartCommand(dir)
	if cmd == "" {
		t.Fatal("got empty start command")
	}
}

func TestDetectDjangoStartCommand_WSGIAutoDetect(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "myproject/wsgi.py", "application = ...\n")
	cmd := detectDjangoStartCommand(dir)
	want := "gunicorn --bind 0.0.0.0:${PORT:-8000} myproject.wsgi:application"
	if cmd != want {
		t.Errorf("StartCommand = %q, want %q", cmd, want)
	}
}

func TestDetectDjangoStartCommand_Fallback(t *testing.T) {
	dir := t.TempDir()
	cmd := detectDjangoStartCommand(dir)
	want := "gunicorn --bind 0.0.0.0:${PORT:-8000} config.wsgi:application"
	if cmd != want {
		t.Errorf("StartCommand = %q, want %q", cmd, want)
	}
}

func TestDetectDjangoWSGIModule_Finds(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "gettingstarted/wsgi.py", "")
	if got := detectDjangoWSGIModule(dir); got != "gettingstarted.wsgi:application" {
		t.Errorf("detectDjangoWSGIModule = %q, want %q", got, "gettingstarted.wsgi:application")
	}
}

func TestDetectDjangoWSGIModule_SkipsVenv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "venv/wsgi.py", "")
	writeFile(t, dir, ".venv/wsgi.py", "")
	if got := detectDjangoWSGIModule(dir); got != "" {
		t.Errorf("detectDjangoWSGIModule = %q, want empty (venv should be skipped)", got)
	}
}

func TestDetectDjangoWSGIModule_SkipsDotDirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".hidden/wsgi.py", "")
	if got := detectDjangoWSGIModule(dir); got != "" {
		t.Errorf("detectDjangoWSGIModule = %q, want empty (dot-dirs should be skipped)", got)
	}
}

func TestDetect_DjangoViaFixture(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "python-django")
	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "django" {
		t.Errorf("Name = %q, want %q", fw.Name, "django")
	}
	if fw.Port != 8000 {
		t.Errorf("Port = %d, want 8000", fw.Port)
	}
}
