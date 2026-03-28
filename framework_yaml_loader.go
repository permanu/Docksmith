package docksmith

import (
	"fmt"
	"os"
	"strings"

	"github.com/permanu/docksmith/yamldef"
)

// LoadFrameworkDefs loads YAML framework definitions from dir.
// Delegates to yamldef.LoadFrameworkDefs.
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
	name, port, matched := yamldef.EvalDefAgainstDir(def, dir)
	if !matched {
		return nil
	}
	return &Framework{
		Name: name,
		Port: port,
	}
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
		keys := yamldef.SortedKeys(sd.Env)
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
