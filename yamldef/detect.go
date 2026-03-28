package yamldef

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// EvalDetectRules returns true when all combinators are satisfied:
//   - Every rule in All must match (AND; vacuously true when empty).
//   - At least one rule in Any must match (OR; vacuously true when empty).
//   - No rule in None may match (NOT; vacuously true when empty).
func EvalDetectRules(dir string, rules DetectRules) bool {
	for _, r := range rules.All {
		if !EvalRule(dir, r) {
			return false
		}
	}
	if len(rules.Any) > 0 {
		anyMatched := false
		for _, r := range rules.Any {
			if EvalRule(dir, r) {
				anyMatched = true
				break
			}
		}
		if !anyMatched {
			return false
		}
	}
	for _, r := range rules.None {
		if EvalRule(dir, r) {
			return false
		}
	}
	return true
}

// EvalRule evaluates a single DetectRule against dir.
// Rules are tried in this priority order:
//  1. dependency  — checks manifest files for the named package
//  2. contains    — substring match inside a file (requires File)
//  3. regex       — regexp match inside a file (requires File)
//  4. json + path — extract a JSON field and check it is non-empty
//  5. toml + path — extract a TOML field and check it is non-empty
//  6. dir         — check a subdirectory exists
//  7. file        — check a file (or glob pattern) exists
//
// If none of the above fields are set the rule vacuously returns true.
func EvalRule(dir string, rule DetectRule) bool {
	if rule.Dependency != "" {
		return hasDependency(dir, rule.Dependency)
	}
	if rule.File != "" && rule.Contains != "" {
		p, err := ContainedPath(dir, rule.File)
		if err != nil {
			return false
		}
		return fileContains(p, rule.Contains)
	}
	if rule.File != "" && rule.Regex != "" {
		p, err := ContainedPath(dir, rule.File)
		if err != nil {
			return false
		}
		return fileMatchesRegex(p, rule.Regex)
	}
	if rule.JSON != "" && rule.Path != "" {
		p, err := ContainedPath(dir, rule.JSON)
		if err != nil {
			return false
		}
		return jsonPathExists(p, rule.Path)
	}
	if rule.TOML != "" && rule.Path != "" {
		p, err := ContainedPath(dir, rule.TOML)
		if err != nil {
			return false
		}
		return tomlPathExists(p, rule.Path)
	}
	if rule.Dir != "" {
		p, err := ContainedPath(dir, rule.Dir)
		if err != nil {
			return false
		}
		return dirExists(p)
	}
	if rule.File != "" {
		// Glob patterns can't use ContainedPath directly — validate no traversal.
		if strings.Contains(rule.File, "..") || filepath.IsAbs(rule.File) {
			return false
		}
		return fileGlobExists(dir, rule.File)
	}
	// Empty rule — vacuously true.
	return true
}

// fileGlobExists returns true when at least one file in dir matches the glob
// pattern. Falls back to exact name check when the pattern contains no special
// characters.
func fileGlobExists(dir, pattern string) bool {
	// Try glob first.
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		// Malformed pattern — fall back to exact existence check.
		return hasFile(dir, pattern)
	}
	return len(matches) > 0
}

const maxRegexPatternLen = 1024

// fileMatchesRegex returns true when the file at path contains a match for the
// compiled regular expression pattern. Rejects patterns longer than 1024 chars
// to limit ReDoS risk from untrusted YAML definitions.
func fileMatchesRegex(path, pattern string) bool {
	if len(pattern) > maxRegexPatternLen {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	data, err := ReadFileLimited(path)
	if err != nil {
		return false
	}
	return re.Match(data)
}

// jsonPathExists reads the JSON file at path and returns true when the
// dot-separated key path resolves to a non-zero/non-empty value.
func jsonPathExists(path, dotPath string) bool {
	data, err := ReadFileLimited(path)
	if err != nil {
		return false
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return false
	}
	val := ExtractDotPath(root, dotPath)
	return val != nil && val != "" && val != float64(0) && val != false
}

// tomlPathExists reads the TOML file at path and returns true when the
// dot-separated key path resolves to a non-zero/non-empty value.
func tomlPathExists(path, dotPath string) bool {
	data, err := ReadFileLimited(path)
	if err != nil {
		return false
	}
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return false
	}
	val := ExtractDotPath(root, dotPath)
	return val != nil && val != "" && val != int64(0) && val != false
}

// ExtractDotPath traverses a nested map[string]any following the dot-separated
// keys in path, returning the leaf value or nil.
func ExtractDotPath(root any, dotPath string) any {
	parts := strings.Split(dotPath, ".")
	cur := root
	for _, part := range parts {
		if part == "" {
			continue
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[part]
		if !ok {
			return nil
		}
	}
	return cur
}

// hasDependency checks all common manifest files for the named dependency.
// Supported manifest files: package.json, requirements.txt, go.mod, Gemfile,
// composer.json, Cargo.toml, mix.exs.
func hasDependency(dir, dep string) bool {
	return hasPackageJSONDep(dir, dep) ||
		hasRequirementsDep(dir, dep) ||
		hasGoModDep(dir, dep) ||
		hasGemfileDep(dir, dep) ||
		hasComposerDep(dir, dep) ||
		hasCargoDep(dir, dep) ||
		hasMixDep(dir, dep)
}

// hasPackageJSONDep checks package.json dependencies and devDependencies.
func hasPackageJSONDep(dir, dep string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies     map[string]any `json:"dependencies"`
		DevDependencies  map[string]any `json:"devDependencies"`
		PeerDependencies map[string]any `json:"peerDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	_, inDeps := pkg.Dependencies[dep]
	_, inDev := pkg.DevDependencies[dep]
	_, inPeer := pkg.PeerDependencies[dep]
	return inDeps || inDev || inPeer
}

// hasRequirementsDep checks requirements.txt for the named package (case-insensitive).
func hasRequirementsDep(dir, dep string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "requirements.txt"))
	if err != nil {
		return false
	}
	depLower := strings.ToLower(dep)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip version specifier: "django>=3.2" -> "django"
		name := strings.FieldsFunc(line, func(r rune) bool {
			return r == '>' || r == '<' || r == '=' || r == '!' || r == '~' || r == '[' || r == ' '
		})[0]
		if strings.ToLower(name) == depLower {
			return true
		}
	}
	return false
}

// hasGoModDep checks go.mod require directives for the module path prefix.
func hasGoModDep(dir, dep string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, dep+" ") || strings.HasPrefix(line, dep+"/") {
			return true
		}
		// inside require block the line starts with the module path
		if line == dep {
			return true
		}
	}
	return false
}

// hasGemfileDep checks Gemfile for a gem declaration.
func hasGemfileDep(dir, dep string) bool {
	return fileContains(filepath.Join(dir, "Gemfile"), `gem '`+dep) ||
		fileContains(filepath.Join(dir, "Gemfile"), `gem "`+dep)
}

// hasComposerDep checks composer.json require and require-dev for the package.
func hasComposerDep(dir, dep string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		return false
	}
	var composer struct {
		Require    map[string]any `json:"require"`
		RequireDev map[string]any `json:"require-dev"`
	}
	if err := json.Unmarshal(data, &composer); err != nil {
		return false
	}
	_, inReq := composer.Require[dep]
	_, inDev := composer.RequireDev[dep]
	return inReq || inDev
}

// hasCargoDep checks Cargo.toml [dependencies] for the crate name.
func hasCargoDep(dir, dep string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return false
	}
	var cargo struct {
		Dependencies    map[string]any `toml:"dependencies"`
		DevDependencies map[string]any `toml:"dev-dependencies"`
	}
	if err := toml.Unmarshal(data, &cargo); err != nil {
		return false
	}
	_, inDeps := cargo.Dependencies[dep]
	_, inDev := cargo.DevDependencies[dep]
	return inDeps || inDev
}

// hasMixDep checks mix.exs for a {:dep, ...} declaration.
func hasMixDep(dir, dep string) bool {
	return fileContains(filepath.Join(dir, "mix.exs"), `{:`+dep+`,`) ||
		fileContains(filepath.Join(dir, "mix.exs"), `{:`+dep+` ,`)
}
