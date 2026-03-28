package docksmith

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Django must run before FastAPI/Flask — manage.py is the strongest signal.
	RegisterDetector("django", detectDjango)
	RegisterDetector("fastapi", detectFastAPI)
	RegisterDetector("flask", detectFlask)
}

func detectDjango(dir string) *Framework {
	if !hasFile(dir, "manage.py") {
		return nil
	}
	if !hasFile(dir, "requirements.txt") && !hasFile(dir, "Pipfile") && !hasFile(dir, "pyproject.toml") {
		return nil
	}
	pm := detectPythonPM(dir)
	return &Framework{
		Name:          "django",
		BuildCommand:  pythonInstallCmd(pm),
		StartCommand:  detectDjangoStartCommand(dir),
		Port:          8000,
		PythonVersion: detectPythonVersion(dir),
		PythonPM:      pm,
		SystemDeps:    detectPythonSystemDeps(dir),
	}
}

// detectDjangoStartCommand resolves the gunicorn invocation in priority order:
//  1. Procfile web: line — most accurate, strip IPv6-only bind
//  2. WSGI auto-discovery via wsgi.py scan
//  3. Fallback to config.wsgi:application
func detectDjangoStartCommand(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, "Procfile")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "web:") {
				continue
			}
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "web:"))
			if !strings.Contains(cmd, "gunicorn") {
				continue
			}
			// IPv6-only bind breaks Docker-internal health checks when IPV6_V6ONLY is set.
			cmd = strings.ReplaceAll(cmd, "[::]:${PORT", "0.0.0.0:${PORT")
			cmd = strings.ReplaceAll(cmd, "[::]:", "0.0.0.0:")
			cmd = strings.ReplaceAll(cmd, "--config gunicorn.conf.py", "")
			cmd = strings.TrimSpace(cmd)
			if !strings.Contains(cmd, "--bind") && !strings.Contains(cmd, " -b ") {
				cmd = cmd + " --bind 0.0.0.0:${PORT:-8000}"
			}
			return cmd
		}
	}
	if wsgi := detectDjangoWSGIModule(dir); wsgi != "" {
		return "gunicorn --bind 0.0.0.0:${PORT:-8000} " + wsgi
	}
	return "gunicorn --bind 0.0.0.0:${PORT:-8000} config.wsgi:application"
}

// detectDjangoWSGIModule scans one level deep for wsgi.py and returns the dotted
// module path (e.g. "myapp.wsgi:application").
func detectDjangoWSGIModule(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	skip := map[string]bool{
		"__pycache__": true, ".git": true, "venv": true, ".venv": true,
		"node_modules": true, "static": true, "media": true,
		"tests": true, "test": true,
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if skip[name] || strings.HasPrefix(name, ".") {
			continue
		}
		if fileExists(filepath.Join(dir, name, "wsgi.py")) {
			return name + ".wsgi:application"
		}
	}
	return ""
}

func detectFastAPI(dir string) *Framework {
	req := filepath.Join(dir, "requirements.txt")
	pyproj := filepath.Join(dir, "pyproject.toml")
	if !(hasFile(dir, "requirements.txt") && fileContains(req, "fastapi")) &&
		!(hasFile(dir, "pyproject.toml") && fileContains(pyproj, "fastapi")) {
		return nil
	}
	pm := detectPythonPM(dir)
	return &Framework{
		Name:          "fastapi",
		BuildCommand:  pythonInstallCmd(pm),
		StartCommand:  "uvicorn main:app --host 0.0.0.0 --port 8000",
		Port:          8000,
		PythonVersion: detectPythonVersion(dir),
		PythonPM:      pm,
		SystemDeps:    detectPythonSystemDeps(dir),
	}
}

func detectFlask(dir string) *Framework {
	req := filepath.Join(dir, "requirements.txt")
	pyproj := filepath.Join(dir, "pyproject.toml")
	if !(hasFile(dir, "requirements.txt") && fileContains(req, "flask")) &&
		!(hasFile(dir, "pyproject.toml") && fileContains(pyproj, "flask")) {
		return nil
	}
	pm := detectPythonPM(dir)
	return &Framework{
		Name:          "flask",
		BuildCommand:  pythonInstallCmd(pm),
		StartCommand:  "gunicorn --bind 0.0.0.0:5000 app:app",
		Port:          5000,
		PythonVersion: detectPythonVersion(dir),
		PythonPM:      pm,
		SystemDeps:    detectPythonSystemDeps(dir),
	}
}
