package detect

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var pythonSystemDepsMap = map[string][]string{
	"psycopg2":        {"libpq-dev"},
	"psycopg2-binary": {},
	"mysqlclient":     {"default-libmysqlclient-dev"},
	"Pillow":          {"libjpeg-dev", "zlib1g-dev"},
	"pillow":          {"libjpeg-dev", "zlib1g-dev"},
	"pycairo":         {"libcairo2-dev"},
	"lxml":            {"libxml2-dev", "libxslt1-dev"},
	"cryptography":    {"libssl-dev", "libffi-dev"},
	"numpy":           {"gfortran", "libopenblas-dev"},
	"scipy":           {"gfortran", "libopenblas-dev"},
	"pdf2image":       {"poppler-utils"},
	"pydub":           {"ffmpeg"},
	"xmlsec":          {"xmlsec1", "libxmlsec1-dev"},
}

func detectPythonVersion(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, ".python-version")); err == nil {
		// Skip comment lines and blank lines, take the first version-like line.
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			return normalizePyVersion(line)
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "runtime.txt")); err == nil {
		line := strings.TrimSpace(string(data))
		line = strings.TrimPrefix(line, "python-")
		if line != "" {
			return normalizePyVersion(line)
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
		re := regexp.MustCompile(`(?:requires-python|python)\s*=\s*"([^"]+)"`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			if v := pyVersionFromConstraint(m[1]); v != "" {
				return v
			}
		}
	}
	return "3.12"
}

func normalizePyVersion(v string) string {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

func pyVersionFromConstraint(constraint string) string {
	re := regexp.MustCompile(`(\d+\.\d+)`)
	if m := re.FindString(constraint); m != "" {
		return m
	}
	return ""
}

func detectPythonPM(dir string) string {
	switch {
	case hasFile(dir, "uv.lock"):
		return "uv"
	case hasFile(dir, "poetry.lock"):
		return "poetry"
	case hasFile(dir, "pdm.lock"):
		return "pdm"
	case hasFile(dir, "Pipfile") || hasFile(dir, "Pipfile.lock"):
		return "pipenv"
	default:
		return "pip"
	}
}

func pythonInstallCmd(pm string) string {
	switch pm {
	case "uv":
		return "pip install uv && uv sync --frozen"
	case "poetry":
		return "pip install poetry && poetry install --no-interaction --no-ansi"
	case "pdm":
		return "pip install pdm && pdm install --no-self"
	case "pipenv":
		return "pip install pipenv && pipenv install --deploy --system"
	default:
		return "pip install --no-cache-dir -r requirements.txt"
	}
}

func detectPythonSystemDeps(dir string) []string {
	pkgNames := make(map[string]bool)
	if data, err := os.ReadFile(filepath.Join(dir, "requirements.txt")); err == nil {
		for _, p := range parseRequirementsTxt(string(data)) {
			pkgNames[p] = true
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
		for _, p := range parsePyprojectDeps(string(data)) {
			pkgNames[p] = true
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "Pipfile")); err == nil {
		for _, p := range parsePipfileDeps(string(data)) {
			pkgNames[p] = true
		}
	}
	seen := make(map[string]bool)
	for pkg := range pkgNames {
		for _, name := range []string{pkg, strings.ToLower(pkg)} {
			if deps, ok := pythonSystemDepsMap[name]; ok {
				for _, d := range deps {
					seen[d] = true
				}
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	result := make([]string, 0, len(seen))
	for d := range seen {
		result = append(result, d)
	}
	sort.Strings(result)
	return result
}

func parseRequirementsTxt(content string) []string {
	var pkgs []string
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		name := line
		if idx := strings.IndexByte(name, '['); idx != -1 {
			name = name[:idx]
		}
		for _, sep := range []string{">=", "<=", "!=", "==", "~=", ">", "<", ";"} {
			if idx := strings.Index(name, sep); idx != -1 {
				name = name[:idx]
			}
		}
		name = strings.TrimSpace(name)
		if name != "" {
			pkgs = append(pkgs, name)
		}
	}
	return pkgs
}

func parsePyprojectDeps(content string) []string {
	var pkgs []string
	re := regexp.MustCompile(`"([a-zA-Z0-9_-]+)(?:\[.*?\])?(?:[><=!~].*?)?"`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			pkgs = append(pkgs, m[1])
		}
	}
	return pkgs
}

func parsePipfileDeps(content string) []string {
	var pkgs []string
	inPackages := false
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "[packages]" {
			inPackages = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPackages = false
			continue
		}
		if inPackages && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if name := strings.TrimSpace(parts[0]); name != "" {
				pkgs = append(pkgs, name)
			}
		}
	}
	return pkgs
}
