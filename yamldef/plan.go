package yamldef

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// ResolveVersion tries each VersionSource in order and returns the first
// non-empty result. Falls back to def.Version.Default or "".
func ResolveVersion(def *FrameworkDef, dir string) string {
	for _, src := range def.Version.Sources {
		if v := extractVersionFromSource(src, dir); v != "" {
			return v
		}
	}
	return def.Version.Default
}

// extractVersionFromSource reads one VersionSource and returns the version string.
func extractVersionFromSource(src VersionSource, dir string) string {
	if src.File != "" {
		p, err := ContainedPath(dir, src.File)
		if err != nil {
			return ""
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
	if src.JSON != "" && src.Path != "" {
		p, err := ContainedPath(dir, src.JSON)
		if err != nil {
			return ""
		}
		return ExtractJSONStringPath(p, src.Path)
	}
	if src.TOML != "" && src.Path != "" {
		p, err := ContainedPath(dir, src.TOML)
		if err != nil {
			return ""
		}
		return ExtractTOMLStringPath(p, src.Path)
	}
	return ""
}

// ResolvePM detects the package manager using PMConfig sources.
func ResolvePM(def *FrameworkDef, dir string) string {
	for _, src := range def.PackageManager.Sources {
		if v := extractPMFromSource(src, dir); v != "" {
			return v
		}
	}
	return def.PackageManager.Default
}

// extractPMFromSource reads one PMSource and returns the package manager name.
func extractPMFromSource(src PMSource, dir string) string {
	if src.JSON != "" && src.Path != "" {
		p, err := ContainedPath(dir, src.JSON)
		if err != nil {
			return ""
		}
		raw := ExtractJSONStringPath(p, src.Path)
		if raw == "" {
			return ""
		}
		// Strip version suffix: "pnpm@8.6.0" -> "pnpm"
		return strings.SplitN(raw, "@", 2)[0]
	}
	if src.File != "" && src.Value != "" {
		p, err := ContainedPath(dir, src.File)
		if err != nil {
			return ""
		}
		if FileExists(p) {
			return src.Value
		}
	}
	return ""
}

// ExtractJSONStringPath returns the string at dotPath in a JSON file, or "".
func ExtractJSONStringPath(path, dotPath string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}
	val := ExtractDotPath(root, dotPath)
	if val == nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// ExtractTOMLStringPath returns the string at dotPath in a TOML file, or "".
func ExtractTOMLStringPath(path, dotPath string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return ""
	}
	val := ExtractDotPath(root, dotPath)
	if val == nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// PMLockfileName returns the canonical lockfile for the given package manager.
func PMLockfileName(pm string) string {
	switch pm {
	case "pnpm":
		return "pnpm-lock.yaml"
	case "yarn":
		return "yarn.lock"
	case "bun":
		return "bun.lockb"
	case "pip", "python":
		return "requirements.txt"
	case "poetry":
		return "poetry.lock"
	case "cargo":
		return "Cargo.lock"
	case "bundler", "gem":
		return "Gemfile.lock"
	case "composer":
		return "composer.lock"
	default:
		return "package-lock.json"
	}
}

// ResolveInstallCommand returns Defaults.Install[pm] or "".
func ResolveInstallCommand(def *FrameworkDef, pm string) string {
	if def.Defaults.Install == nil {
		return ""
	}
	return def.Defaults.Install[pm]
}

// Sub substitutes all known template variables in s; unknown tokens stay as-is.
func Sub(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

// SortedKeys returns the keys of m in lexicographic order (insertion sort).
func SortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
