package detect

import (
	"github.com/permanu/docksmith/core"
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

func detectDjango(dir string) *core.Framework {
	if !hasFile(dir, "manage.py") {
		return nil
	}
	if !hasFile(dir, "requirements.txt") && !hasFile(dir, "Pipfile") && !hasFile(dir, "pyproject.toml") {
		return nil
	}
	pm := detectPythonPM(dir)
	return &core.Framework{
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
		return "gunicorn --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2 " + wsgi
	}
	return "gunicorn --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2 config.wsgi:application"
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

func detectFastAPI(dir string) *core.Framework {
	req := filepath.Join(dir, "requirements.txt")
	pyproj := filepath.Join(dir, "pyproject.toml")
	if !(hasFile(dir, "requirements.txt") && fileContains(req, "fastapi")) &&
		!(hasFile(dir, "pyproject.toml") && fileContains(pyproj, "fastapi")) {
		return nil
	}
	pm := detectPythonPM(dir)
	appTarget := detectPythonAppTarget(dir, "FastAPI(")
	if appTarget == "" {
		appTarget = "main:app"
	}
	return &core.Framework{
		Name:          "fastapi",
		BuildCommand:  pythonInstallCmd(pm),
		StartCommand:  "gunicorn " + appTarget + " --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} -k uvicorn.workers.UvicornWorker",
		Port:          8000,
		PythonVersion: detectPythonVersion(dir),
		PythonPM:      pm,
		SystemDeps:    detectPythonSystemDeps(dir),
	}
}

func detectFlask(dir string) *core.Framework {
	req := filepath.Join(dir, "requirements.txt")
	pyproj := filepath.Join(dir, "pyproject.toml")
	if !(hasFile(dir, "requirements.txt") && fileContains(req, "flask")) &&
		!(hasFile(dir, "pyproject.toml") && fileContains(pyproj, "flask")) {
		return nil
	}
	pm := detectPythonPM(dir)
	appTarget := detectPythonAppTarget(dir, "Flask(")
	if appTarget == "" {
		appTarget = "app:app"
	}
	return &core.Framework{
		Name:          "flask",
		BuildCommand:  pythonInstallCmd(pm),
		StartCommand:  "gunicorn " + appTarget + " --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2",
		Port:          8000,
		PythonVersion: detectPythonVersion(dir),
		PythonPM:      pm,
		SystemDeps:    detectPythonSystemDeps(dir),
	}
}

// detectPythonAppTarget scans top-level .py files for a pattern like
// `app = FastAPI(` or `application = Flask(` and returns the gunicorn target
// in "module:variable" format (e.g. "main:app", "server:application").
// Returns "" if nothing found — callers provide the fallback.
func detectPythonAppTarget(dir string, constructorPattern string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.Contains(trimmed, constructorPattern) {
				continue
			}
			// Match patterns: "app = FastAPI(", "application = Flask("
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx == -1 {
				continue
			}
			varName := strings.TrimSpace(trimmed[:eqIdx])
			// Skip invalid variable names (multi-word, dotted, etc.)
			if strings.ContainsAny(varName, " .\t") {
				continue
			}
			module := strings.TrimSuffix(e.Name(), ".py")
			return module + ":" + varName
		}
	}
	return ""
}
