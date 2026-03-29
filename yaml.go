package docksmith

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/permanu/docksmith/yamldef"
)

// ---------------------------------------------------------------------------
// YAML framework loader wrappers
// ---------------------------------------------------------------------------

// LoadFrameworkDefs loads YAML framework definitions from dir.
func LoadFrameworkDefs(dir string) ([]*FrameworkDef, error) {
	return yamldef.LoadFrameworkDefs(dir)
}

// LoadAndRegisterFrameworks loads YAML defs from each dir (in order) and
// registers them as detectors. Resolution order after registration:
//  1. .docksmith/frameworks/ in project dir (call with project path first)
//  2. ~/.docksmith/frameworks/ (call with home path second)
//  3. Built-in Go detectors (already registered at init time)
//
// YAML detectors are prepended, so the last dir passed has lowest YAML priority.
// Dirs that don't exist are silently skipped.
func LoadAndRegisterFrameworks(dirs ...string) error {
	var errs []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		defs, err := yamldef.LoadFrameworkDefs(dir)
		if err != nil {
			errs = append(errs, err.Error())
		}
		for _, def := range defs {
			d := def // capture
			RegisterDetector("yaml:"+d.Name, func(dir string) *Framework {
				return evalDefAgainstDir(d, dir)
			})
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// evalDefAgainstDir runs the detect rules for def against dir.
func evalDefAgainstDir(def *FrameworkDef, dir string) *Framework {
	name, port, matched := yamldef.EvalDefAgainstDir(def, dir)
	if !matched {
		return nil
	}
	return &Framework{
		Name: name,
		Port: port,
	}
}

// ---------------------------------------------------------------------------
// YAML framework plan builders
// ---------------------------------------------------------------------------

// BuildPlanFromDef converts a FrameworkDef into a BuildPlan.
// It resolves Base fields via ResolveDockerTag and converts StepDefs to Steps.
func BuildPlanFromDef(def *FrameworkDef, fw *Framework) (*BuildPlan, error) {
	if len(def.Plan.Stages) == 0 {
		return nil, fmt.Errorf("%w: yaml def %q has no stages", ErrInvalidPlan, def.Name)
	}

	port := def.Plan.Port
	if fw != nil && fw.Port != 0 {
		port = fw.Port
	}

	stages := make([]Stage, 0, len(def.Plan.Stages))
	for _, sd := range def.Plan.Stages {
		s, err := buildStageFromDef(sd, fw)
		if err != nil {
			return nil, fmt.Errorf("stage %q: %w", sd.Name, err)
		}
		stages = append(stages, s)
	}

	return &BuildPlan{
		Framework: def.Name,
		Stages:    stages,
		Expose:    port,
	}, nil
}

func buildStageFromDef(sd StageDef, fw *Framework) (Stage, error) {
	from := sd.From
	if from == "" && sd.Base != "" {
		version := versionForBase(sd.Base, fw)
		from = ResolveDockerTag(sd.Base, version)
	}
	if from == "" {
		return Stage{}, fmt.Errorf("stage %q has no from or base", sd.Name)
	}

	steps := make([]Step, 0, len(sd.Steps))
	for _, stepDef := range sd.Steps {
		step, err := buildStepFromDef(stepDef)
		if err != nil {
			return Stage{}, err
		}
		steps = append(steps, step)
	}

	return Stage{Name: sd.Name, From: from, Steps: steps}, nil
}

func buildStepFromDef(sd StepDef) (Step, error) {
	switch {
	case sd.Workdir != "":
		return Step{Type: StepWorkdir, Args: []string{sd.Workdir}}, nil
	case len(sd.Copy) > 0:
		return Step{Type: StepCopy, Args: sd.Copy}, nil
	case sd.CopyFrom != nil:
		return Step{
			Type: StepCopyFrom,
			CopyFrom: &CopyFrom{
				Stage: sd.CopyFrom.Stage,
				Src:   sd.CopyFrom.Src,
				Dst:   sd.CopyFrom.Dst,
			},
		}, nil
	case sd.Run != "":
		s := Step{Type: StepRun, Args: []string{sd.Run}}
		if sd.Cache != "" {
			s.CacheMount = &CacheMount{Target: sd.Cache}
		}
		return s, nil
	case len(sd.Env) > 0:
		keys := yamldef.SortedKeys(sd.Env)
		args := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			args = append(args, k, sd.Env[k])
		}
		return Step{Type: StepEnv, Args: args}, nil
	case sd.Expose != "":
		return Step{Type: StepExpose, Args: []string{sd.Expose}}, nil
	case len(sd.Cmd) > 0:
		return Step{Type: StepCmd, Args: sd.Cmd}, nil
	}
	return Step{}, fmt.Errorf("empty step definition")
}

func versionForBase(base string, fw *Framework) string {
	if fw == nil {
		return ""
	}
	switch base {
	case "node":
		return fw.NodeVersion
	case "python":
		return fw.PythonVersion
	case "go":
		return fw.GoVersion
	}
	return ""
}

// ---------------------------------------------------------------------------
// YAML framework plan with template variable resolution
// ---------------------------------------------------------------------------

// BuildPlanFromDefDir builds a BuildPlan from a FrameworkDef by resolving
// version and package manager from the given project directory.
// Template variables are substituted with strings.ReplaceAll;
// unknown tokens are left in-place (never an error). Supported variables:
// {{runtime}}, {{version}}, {{pm}}, {{lockfile}}, {{install_command}},
// {{build_command}}, {{start_command}}, {{port}}.
func BuildPlanFromDefDir(def *FrameworkDef, dir string) (*BuildPlan, error) {
	return buildPlanFromDefDir(def, dir)
}

func buildPlanFromDefDir(def *FrameworkDef, dir string) (*BuildPlan, error) {
	if def == nil {
		return nil, fmt.Errorf("buildPlanFromDef: def must not be nil")
	}

	version := yamldef.ResolveVersion(def, dir)
	pm := yamldef.ResolvePM(def, dir)

	vars := map[string]string{
		"{{runtime}}":         def.Runtime,
		"{{version}}":         version,
		"{{pm}}":              pm,
		"{{lockfile}}":        yamldef.PMLockfileName(pm),
		"{{install_command}}": yamldef.ResolveInstallCommand(def, pm),
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

func resolveFrom(sd StageDef, def *FrameworkDef, version string) (string, error) {
	if sd.Base != "" {
		return ResolveDockerTag(sd.Base, version), nil
	}
	if sd.From != "" {
		return sd.From, nil
	}
	return "", fmt.Errorf("stage %q: either base or from must be set", sd.Name)
}

func resolveStep(sd StepDef, vars map[string]string) (Step, error) {
	if sd.Workdir != "" {
		return Step{
			Type: StepWorkdir,
			Args: []string{yamldef.Sub(sd.Workdir, vars)},
		}, nil
	}
	if sd.CopyFrom != nil {
		return Step{
			Type: StepCopyFrom,
			CopyFrom: &CopyFrom{
				Stage: yamldef.Sub(sd.CopyFrom.Stage, vars),
				Src:   yamldef.Sub(sd.CopyFrom.Src, vars),
				Dst:   yamldef.Sub(sd.CopyFrom.Dst, vars),
			},
			Link: true,
		}, nil
	}
	if len(sd.Copy) > 0 {
		args := make([]string, len(sd.Copy))
		for i, s := range sd.Copy {
			args[i] = yamldef.Sub(s, vars)
		}
		return Step{Type: StepCopy, Args: args}, nil
	}
	if sd.Run != "" {
		step := Step{
			Type: StepRun,
			Args: []string{yamldef.Sub(sd.Run, vars)},
		}
		if sd.Cache != "" {
			step.CacheMount = &CacheMount{Target: yamldef.Sub(sd.Cache, vars)}
		}
		return step, nil
	}
	if len(sd.Env) > 0 {
		keys := yamldef.SortedKeys(sd.Env)
		args := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			args = append(args, yamldef.Sub(k, vars), yamldef.Sub(sd.Env[k], vars))
		}
		return Step{Type: StepEnv, Args: args}, nil
	}
	if len(sd.Cmd) > 0 {
		args := make([]string, len(sd.Cmd))
		for i, s := range sd.Cmd {
			args[i] = yamldef.Sub(s, vars)
		}
		return Step{Type: StepCmd, Args: args}, nil
	}
	if sd.Expose != "" {
		return Step{
			Type: StepExpose,
			Args: []string{yamldef.Sub(sd.Expose, vars)},
		}, nil
	}
	return Step{}, fmt.Errorf("empty or malformed step definition")
}

// ---------------------------------------------------------------------------
// YAML framework test runner wrappers
// ---------------------------------------------------------------------------

// RunFrameworkTests delegates to yamldef.RunFrameworkTests.
func RunFrameworkTests(yamlPath string) ([]TestResult, error) {
	return yamldef.RunFrameworkTests(yamlPath)
}

// RunFrameworkDefTests delegates to yamldef.RunFrameworkDefTests.
func RunFrameworkDefTests(def *FrameworkDef) error {
	return yamldef.RunFrameworkDefTests(def)
}

// runTestCase runs a single test case using the global detector registry.
func runTestCase(tc TestCase) TestResult {
	dir, err := os.MkdirTemp("", "docksmith-test-*")
	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mktemp: %v", err)}
	}
	defer os.RemoveAll(dir)

	for relPath, content := range tc.Fixture {
		full, pathErr := yamldef.ContainedPath(dir, relPath)
		if pathErr != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("unsafe fixture path %q: %v", relPath, pathErr)}
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mkdir %s: %v", relPath, err)}
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("write %s: %v", relPath, err)}
		}
	}

	fw, detectErr := Detect(dir)
	if detectErr != nil && !errors.Is(detectErr, ErrNotDetected) {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("detect: %v", detectErr)}
	}
	detected := fw != nil && fw.Name != "static"

	if detected != tc.Expect.Detected {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("detected=%v, want %v (framework=%q)", detected, tc.Expect.Detected, fw.Name),
		}
	}

	if tc.Expect.Framework != "" && fw.Name != tc.Expect.Framework {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("framework=%q, want %q", fw.Name, tc.Expect.Framework),
		}
	}

	if tc.Expect.Port != 0 && fw.Port != tc.Expect.Port {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("port=%d, want %d", fw.Port, tc.Expect.Port),
		}
	}

	return TestResult{Name: tc.Name, Passed: true}
}
