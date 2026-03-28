package docksmith

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// buildPlanFromDef converts a FrameworkDef into a BuildPlan using context
// detected from dir. Template variables are substituted with strings.ReplaceAll;
// unknown tokens are left in-place (never an error). Supported variables:
// {{runtime}}, {{version}}, {{pm}}, {{lockfile}}, {{install_command}},
// {{build_command}}, {{start_command}}, {{port}}.
func buildPlanFromDef(def *FrameworkDef, dir string) (*BuildPlan, error) {
	if def == nil {
		return nil, fmt.Errorf("buildPlanFromDef: def must not be nil")
	}

	version := resolveVersion(def, dir)
	pm := resolvePM(def, dir)

	vars := map[string]string{
		"{{runtime}}":         def.Runtime,
		"{{version}}":         version,
		"{{pm}}":              pm,
		"{{lockfile}}":         pmLockfileName(pm),
		"{{install_command}}": resolveInstallCommand(def, pm),
		"{{build_command}}":   def.Defaults.Build,
		"{{start_command}}":   def.Defaults.Start,
		"{{port}}":            fmt.Sprintf("%d", def.Plan.Port),
	}

	stages := make([]Stage, 0, len(def.Plan.Stages))
	for _, sd := range def.Plan.Stages {
		s, err := resolveStage(sd, def, vars, version)
		if err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}

	return &BuildPlan{
		Framework: def.Name,
		Expose:    def.Plan.Port,
		Stages:    stages,
	}, nil
}

// resolveStage converts a StageDef to a Stage with template variables expanded.
func resolveStage(sd StageDef, def *FrameworkDef, vars map[string]string, version string) (Stage, error) {
	from, err := resolveFrom(sd, def, version)
	if err != nil {
		return Stage{}, err
	}

	steps := make([]Step, 0, len(sd.Steps))
	for _, stepDef := range sd.Steps {
		step, err := resolveStep(stepDef, vars)
		if err != nil {
			return Stage{}, err
		}
		steps = append(steps, step)
	}

	return Stage{
		Name:  sd.Name,
		From:  from,
		Steps: steps,
	}, nil
}

// resolveFrom returns the Docker image for a stage. Base is resolved via
// ResolveDockerTag; From is used as-is. Base takes precedence over From.
func resolveFrom(sd StageDef, def *FrameworkDef, version string) (string, error) {
	if sd.Base != "" {
		return ResolveDockerTag(sd.Base, version), nil
	}
	if sd.From != "" {
		return sd.From, nil
	}
	return "", fmt.Errorf("stage %q: either base or from must be set", sd.Name)
}

// resolveStep converts a StepDef to a Step, expanding all template variables.
func resolveStep(sd StepDef, vars map[string]string) (Step, error) {
	if sd.Workdir != "" {
		return Step{
			Type: StepWorkdir,
			Args: []string{sub(sd.Workdir, vars)},
		}, nil
	}
	if sd.CopyFrom != nil {
		return Step{
			Type: StepCopyFrom,
			CopyFrom: &CopyFrom{
				Stage: sub(sd.CopyFrom.Stage, vars),
				Src:   sub(sd.CopyFrom.Src, vars),
				Dst:   sub(sd.CopyFrom.Dst, vars),
			},
			Link: true,
		}, nil
	}
	if len(sd.Copy) > 0 {
		args := make([]string, len(sd.Copy))
		for i, s := range sd.Copy {
			args[i] = sub(s, vars)
		}
		return Step{Type: StepCopy, Args: args}, nil
	}
	if sd.Run != "" {
		step := Step{
			Type: StepRun,
			Args: []string{sub(sd.Run, vars)},
		}
		if sd.Cache != "" {
			step.CacheMount = &CacheMount{Target: sub(sd.Cache, vars)}
		}
		return step, nil
	}
	if len(sd.Env) > 0 {
		// Alternating [key, value, ...] args in sorted order for determinism.
		keys := sortedKeys(sd.Env)
		args := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			args = append(args, sub(k, vars), sub(sd.Env[k], vars))
		}
		return Step{Type: StepEnv, Args: args}, nil
	}
	if len(sd.Cmd) > 0 {
		args := make([]string, len(sd.Cmd))
		for i, s := range sd.Cmd {
			args[i] = sub(s, vars)
		}
		return Step{Type: StepCmd, Args: args}, nil
	}
	if sd.Expose != "" {
		return Step{
			Type: StepExpose,
			Args: []string{sub(sd.Expose, vars)},
		}, nil
	}
	return Step{Type: StepRun, Args: []string{""}}, nil // no action set
}

// sub substitutes all known template variables in s; unknown tokens stay as-is.
func sub(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

// sortedKeys returns the keys of m in lexicographic order (insertion sort).
func sortedKeys(m map[string]string) []string {
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

// resolveVersion tries each VersionSource in order and returns the first
// non-empty result. Falls back to def.Version.Default or "".
func resolveVersion(def *FrameworkDef, dir string) string {
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
		p, err := containedPath(dir, src.File)
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
		p, err := containedPath(dir, src.JSON)
		if err != nil {
			return ""
		}
		return extractJSONStringPath(p, src.Path)
	}
	if src.TOML != "" && src.Path != "" {
		p, err := containedPath(dir, src.TOML)
		if err != nil {
			return ""
		}
		return extractTOMLStringPath(p, src.Path)
	}
	return ""
}

// resolvePM detects the package manager using PMConfig sources.
func resolvePM(def *FrameworkDef, dir string) string {
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
		p, err := containedPath(dir, src.JSON)
		if err != nil {
			return ""
		}
		raw := extractJSONStringPath(p, src.Path)
		if raw == "" {
			return ""
		}
		// Strip version suffix: "pnpm@8.6.0" → "pnpm"
		return strings.SplitN(raw, "@", 2)[0]
	}
	if src.File != "" && src.Value != "" {
		p, err := containedPath(dir, src.File)
		if err != nil {
			return ""
		}
		if fileExists(p) {
			return src.Value
		}
	}
	return ""
}

// extractJSONStringPath returns the string at dotPath in a JSON file, or "".
func extractJSONStringPath(path, dotPath string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}
	val := extractDotPath(root, dotPath)
	if val == nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// extractTOMLStringPath returns the string at dotPath in a TOML file, or "".
func extractTOMLStringPath(path, dotPath string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return ""
	}
	val := extractDotPath(root, dotPath)
	if val == nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

// pmLockfileName returns the canonical lockfile for the given package manager.
func pmLockfileName(pm string) string {
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

// resolveInstallCommand returns Defaults.Install[pm] or "".
func resolveInstallCommand(def *FrameworkDef, pm string) string {
	if def.Defaults.Install == nil {
		return ""
	}
	return def.Defaults.Install[pm]
}
