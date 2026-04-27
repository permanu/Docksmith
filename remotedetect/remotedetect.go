// Package remotedetect provides framework detection from a flat list of file
// paths (e.g. a GitHub tree API response).  Unlike the local detect package
// which reads files from the filesystem, this package works with path strings
// only, making it suitable for remote repo inspection during deploy wizards.
package remotedetect

import (
	"path/filepath"
	"slices"
	"strings"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// ServiceInfo describes a deployable service found in a repo directory.
type ServiceInfo struct {
	Name          string `json:"name"`
	RootDirectory string `json:"root_directory"`
	Framework     string `json:"framework"`
	BuildCommand  string `json:"build_command"`
	StartCommand  string `json:"start_command"`
	Port          int    `json:"port"`
}

// Result holds the framework auto-detection result for a repo's file tree.
type Result struct {
	Framework       string       `json:"framework"`
	BuildCommand    string       `json:"build_command"`
	StartCommand    string       `json:"start_command"`
	Port            int          `json:"port"`
	HasDockerfile   bool         `json:"has_dockerfile"`
	RootCandidates  []string     `json:"root_candidates"`
	Directories     []string     `json:"directories"`
	Services        []ServiceInfo `json:"services"`
	HasDockerCompose bool        `json:"has_docker_compose"`
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Detect scans a flat list of repo file paths and returns the best framework
// match along with suggested build/start commands and root candidates.
//
// Language-specific frameworks (go, nodejs, nextjs, python, …) are always
// preferred over "dockerfile" — a Dockerfile at the repo root is usually just
// a deployment convenience, while the actual application lives in a
// subdirectory (e.g. backend/, frontend/).
func Detect(paths []string) Result {
	hasDockerfile := false
	hasDockerCompose := false
	manifestDirs := map[string]bool{}

	for _, p := range paths {
		base := filepath.Base(p)
		rawDir := filepath.Dir(p)
		if rawDir == "." {
			rawDir = ""
		}

		switch base {
		case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
			hasDockerCompose = true
		}

		dir := "/" + rawDir
		if dir == "/." {
			dir = "/"
		}

		switch base {
		case "Dockerfile":
			hasDockerfile = true
			manifestDirs[dir] = true
		case "package.json":
			manifestDirs[dir] = true
		case "go.mod":
			manifestDirs[dir] = true
		case "requirements.txt", "pyproject.toml", "Pipfile", "manage.py":
			manifestDirs[dir] = true
		case "Cargo.toml":
			manifestDirs[dir] = true
		}
	}

	rootCandidates := []string{"/"}
	for dir := range manifestDirs {
		if dir != "/" {
			rootCandidates = append(rootCandidates, dir)
		}
	}
	slices.Sort(rootCandidates)

	result := Result{
		HasDockerfile:    hasDockerfile,
		RootCandidates:   rootCandidates,
		Port:             3000,
		HasDockerCompose: hasDockerCompose,
	}

	var services []ServiceInfo
	for _, dir := range rootCandidates {
		if svc := detectService(dir, filterPathsToRoot(paths, dir)); svc != nil {
			services = append(services, *svc)
		}
	}
	slices.SortStableFunc(services, func(a, b ServiceInfo) int {
		if a.RootDirectory == b.RootDirectory {
			return strings.Compare(a.Name, b.Name)
		}
		if a.RootDirectory == "/" {
			return -1
		}
		if b.RootDirectory == "/" {
			return 1
		}
		return strings.Compare(a.RootDirectory, b.RootDirectory)
	})
	result.Services = services

	if best := pickBestService(services); best != nil {
		result.Framework = best.Framework
		result.BuildCommand = best.BuildCommand
		result.StartCommand = best.StartCommand
		result.Port = best.Port
	} else if hasDockerfile {
		result.Framework = "docker"
		result.Port = 3000
	} else {
		result.Framework = "unknown"
		result.Port = 3000
	}

	return result
}

// DetectForRoot runs detection scoped to a specific root directory.
// It returns the detection result with the full repo directory listing.
func DetectForRoot(paths []string, root string) Result {
	detection := Detect(filterPathsToRoot(paths, root))
	detection.Directories = collectDirectories(paths)
	return detection
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// topLevelFileSet returns only the filenames (no directory prefix) from paths.
func topLevelFileSet(paths []string) map[string]bool {
	fileSet := map[string]bool{}
	for _, p := range paths {
		if strings.Contains(p, "/") {
			continue
		}
		fileSet[p] = true
	}
	return fileSet
}

// hasTopLevelPrefix returns true if any top-level path has the given prefix.
func hasTopLevelPrefix(paths []string, prefix string) bool {
	for _, p := range paths {
		if strings.Contains(p, "/") {
			continue
		}
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// normalizeServiceDir extracts a human-readable name and a canonical root
// directory path from a raw directory string.
func normalizeServiceDir(dir string) (name string, rootDir string) {
	name = "app"
	clean := strings.Trim(dir, "/")
	if clean != "" {
		parts := strings.Split(clean, "/")
		name = parts[len(parts)-1]
		rootDir = "/" + clean + "/"
		return name, rootDir
	}
	return name, "/"
}

// detectService inspects a directory subtree and returns a deployable service
// suggestion when the path structure is confident enough.
func detectService(dir string, paths []string) *ServiceInfo {
	fileSet := topLevelFileSet(paths)
	name, rootDir := normalizeServiceDir(dir)
	svc := &ServiceInfo{
		Name:          name,
		RootDirectory: rootDir,
	}

	switch {
	case fileSet["manage.py"]:
		svc.Framework = "django"
		if fileSet["pyproject.toml"] && !fileSet["requirements.txt"] {
			svc.BuildCommand = "pip install ."
		} else {
			svc.BuildCommand = "pip install -r requirements.txt"
		}
		if module := detectPythonModule(paths, "wsgi.py"); module != "" {
			svc.StartCommand = "gunicorn --bind 0.0.0.0:$PORT " + module + ":application"
		}
		svc.Port = 8000
	case fileSet["package.json"] && hasTopLevelPrefix(paths, "next.config"):
		svc.Framework = "nextjs"
		svc.BuildCommand = "npm install && npm run build"
		svc.StartCommand = "npm start"
		svc.Port = 3000
	case fileSet["package.json"]:
		svc.Framework = "nodejs"
		svc.BuildCommand = "npm install && npm run build"
		svc.StartCommand = "npm start"
		svc.Port = 3000
	case fileSet["go.mod"]:
		svc.Framework = "go"
		svc.BuildCommand = "go build -o app ."
		svc.StartCommand = "./app"
		svc.Port = 8080
	case fileSet["Cargo.toml"]:
		svc.Framework = "rust"
		svc.BuildCommand = "cargo build --release"
		svc.StartCommand = "./target/release/app"
		svc.Port = 8080
	case fileSet["requirements.txt"] && fileSet["main.py"]:
		svc.Framework = "fastapi"
		svc.BuildCommand = "pip install -r requirements.txt"
		svc.StartCommand = "uvicorn main:app --host 0.0.0.0 --port $PORT"
		svc.Port = 8000
	case fileSet["requirements.txt"] && fileSet["app.py"]:
		svc.Framework = "flask"
		svc.BuildCommand = "pip install -r requirements.txt"
		svc.StartCommand = "gunicorn --bind 0.0.0.0:$PORT app:app"
		svc.Port = 8000
	case fileSet["requirements.txt"] || fileSet["pyproject.toml"] || fileSet["Pipfile"]:
		svc.Framework = "python"
		if fileSet["pyproject.toml"] && !fileSet["requirements.txt"] {
			svc.BuildCommand = "pip install ."
		} else {
			svc.BuildCommand = "pip install -r requirements.txt"
		}
		svc.StartCommand = ""
		svc.Port = 8000
	case fileSet["Dockerfile"]:
		svc.Framework = "docker"
		svc.Port = 8080
	default:
		return nil
	}

	return svc
}

// pickBestService returns the best service from a sorted list.
// Language-specific frameworks (go, nodejs, python, …) are always preferred
// over "docker" — a Dockerfile at the repo root is usually just a deployment
// convenience, while the actual application lives in a subdirectory.
func pickBestService(candidates []ServiceInfo) *ServiceInfo {
	if len(candidates) == 0 {
		return nil
	}
	for i := range candidates {
		if candidates[i].Framework != "docker" {
			return &candidates[i]
		}
	}
	return &candidates[0] // docker-only fallback
}

// collectDirectories returns all unique directory paths from a flat path list.
func collectDirectories(paths []string) []string {
	dirs := map[string]bool{"/": true}

	for _, p := range paths {
		dir := filepath.Dir(p)
		if dir == "." || dir == "" {
			continue
		}

		parts := strings.Split(dir, "/")
		current := ""
		for _, part := range parts {
			if part == "" {
				continue
			}
			current += "/" + part
			dirs[current+"/"] = true
		}
	}

	out := make([]string, 0, len(dirs))
	for dir := range dirs {
		out = append(out, dir)
	}
	slices.Sort(out)
	return out
}

// filterPathsToRoot filters a flat path list to only include paths under root.
// Paths are rebased so that "backend/cmd/server/main.go" under "/backend/"
// becomes "cmd/server/main.go".
func filterPathsToRoot(paths []string, root string) []string {
	normalized := strings.TrimSpace(root)
	if normalized == "" || normalized == "/" || normalized == "." {
		return paths
	}

	trimmed := strings.Trim(normalized, "/")
	if trimmed == "" {
		return paths
	}

	prefix := trimmed + "/"
	filtered := make([]string, 0)
	for _, p := range paths {
		if p == trimmed {
			filtered = append(filtered, filepath.Base(p))
			continue
		}
		if strings.HasPrefix(p, prefix) {
			filtered = append(filtered, strings.TrimPrefix(p, prefix))
		}
	}
	return filtered
}

// detectPythonModule walks paths looking for filename and returns the
// Python module path (e.g. "myapp.wsgi" from "myapp/wsgi.py").
func detectPythonModule(paths []string, filename string) string {
	for _, p := range paths {
		if !strings.HasSuffix(p, "/"+filename) && p != filename {
			continue
		}
		if isIgnoredPythonPath(p) {
			continue
		}
		trimmed := strings.TrimSuffix(p, ".py")
		trimmed = strings.Trim(trimmed, "/")
		if trimmed == "" {
			return ""
		}
		return strings.ReplaceAll(trimmed, "/", ".")
	}
	return ""
}

func isIgnoredPythonPath(path string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		switch {
		case part == "", part == "__pycache__", part == "node_modules", part == "site-packages":
			return true
		case strings.HasPrefix(part, "."):
			return true
		case part == "venv", part == ".venv", part == "env":
			return true
		}
	}
	return false
}
