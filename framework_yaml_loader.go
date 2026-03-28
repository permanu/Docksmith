package docksmith

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFrameworkDefs loads YAML framework definitions from dir.
// Files that fail to parse are skipped with an error returned after all files
// are attempted. The caller decides whether partial results are acceptable.
func LoadFrameworkDefs(dir string) ([]*FrameworkDef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read framework dir %s: %w", dir, err)
	}

	var defs []*FrameworkDef
	var errs []string
	for _, e := range entries {
		if e.IsDir() || !isYAMLFile(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		var def FrameworkDef
		if err := yaml.Unmarshal(data, &def); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		if def.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: missing name field", e.Name()))
			continue
		}
		defs = append(defs, &def)
	}

	if len(errs) > 0 {
		return defs, fmt.Errorf("framework load errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return defs, nil
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
		defs, err := LoadFrameworkDefs(dir)
		if err != nil {
			// partial load — still register what we got, log the error
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
// Returns a populated Framework on match, nil otherwise.
func evalDefAgainstDir(def *FrameworkDef, dir string) *Framework {
	if !evalDetectRules(dir, def.Detect) {
		return nil
	}
	port := def.Plan.Port
	if port == 0 {
		port = def.Plan.Port
	}
	return &Framework{
		Name: def.Name,
		Port: port,
	}
}

func isYAMLFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

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
		s, err := buildStage(sd, fw)
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

func buildStage(sd StageDef, fw *Framework) (Stage, error) {
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
		step, err := buildStep(stepDef)
		if err != nil {
			return Stage{}, err
		}
		steps = append(steps, step)
	}

	return Stage{Name: sd.Name, From: from, Steps: steps}, nil
}

func buildStep(sd StepDef) (Step, error) {
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
		keys := sortedKeys(sd.Env)
		for _, k := range keys {
			return Step{Type: StepEnv, Args: []string{k, sd.Env[k]}}, nil
		}
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
