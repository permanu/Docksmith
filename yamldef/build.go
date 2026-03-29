package yamldef

import (
	"fmt"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/plan"
)

// BuildPlanFromDef converts a FrameworkDef into a BuildPlan.
// It resolves Base fields via ResolveDockerTag and converts StepDefs to Steps.
func BuildPlanFromDef(def *FrameworkDef, fw *core.Framework) (*core.BuildPlan, error) {
	if len(def.Plan.Stages) == 0 {
		return nil, fmt.Errorf("%w: yaml def %q has no stages", core.ErrInvalidPlan, def.Name)
	}

	port := def.Plan.Port
	if fw != nil && fw.Port != 0 {
		port = fw.Port
	}

	stages := make([]core.Stage, 0, len(def.Plan.Stages))
	for _, sd := range def.Plan.Stages {
		s, err := buildStageFromDef(sd, fw)
		if err != nil {
			return nil, fmt.Errorf("stage %q: %w", sd.Name, err)
		}
		stages = append(stages, s)
	}

	return &core.BuildPlan{
		Framework: def.Name,
		Stages:    stages,
		Expose:    port,
	}, nil
}

func buildStageFromDef(sd StageDef, fw *core.Framework) (core.Stage, error) {
	from := sd.From
	if from == "" && sd.Base != "" {
		version := versionForBase(sd.Base, fw)
		from = plan.ResolveDockerTag(sd.Base, version)
	}
	if from == "" {
		return core.Stage{}, fmt.Errorf("stage %q has no from or base", sd.Name)
	}

	steps := make([]core.Step, 0, len(sd.Steps))
	for _, stepDef := range sd.Steps {
		step, err := buildStepFromDef(stepDef)
		if err != nil {
			return core.Stage{}, err
		}
		steps = append(steps, step)
	}

	return core.Stage{Name: sd.Name, From: from, Steps: steps}, nil
}

func buildStepFromDef(sd StepDef) (core.Step, error) {
	switch {
	case sd.Workdir != "":
		return core.Step{Type: core.StepWorkdir, Args: []string{sd.Workdir}}, nil
	case len(sd.Copy) > 0:
		return core.Step{Type: core.StepCopy, Args: sd.Copy}, nil
	case sd.CopyFrom != nil:
		return core.Step{
			Type: core.StepCopyFrom,
			CopyFrom: &core.CopyFrom{
				Stage: sd.CopyFrom.Stage,
				Src:   sd.CopyFrom.Src,
				Dst:   sd.CopyFrom.Dst,
			},
		}, nil
	case sd.Run != "":
		s := core.Step{Type: core.StepRun, Args: []string{sd.Run}}
		if sd.Cache != "" {
			s.CacheMount = &core.CacheMount{Target: sd.Cache}
		}
		return s, nil
	case len(sd.Env) > 0:
		keys := SortedKeys(sd.Env)
		args := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			args = append(args, k, sd.Env[k])
		}
		return core.Step{Type: core.StepEnv, Args: args}, nil
	case sd.Expose != "":
		return core.Step{Type: core.StepExpose, Args: []string{sd.Expose}}, nil
	case len(sd.Cmd) > 0:
		return core.Step{Type: core.StepCmd, Args: sd.Cmd}, nil
	}
	return core.Step{}, fmt.Errorf("empty step definition")
}

func versionForBase(base string, fw *core.Framework) string {
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

// BuildPlanFromDefDir builds a BuildPlan from a FrameworkDef by resolving
// version and package manager from the given project directory.
// Template variables are substituted with strings.ReplaceAll;
// unknown tokens are left in-place (never an error). Supported variables:
// {{runtime}}, {{version}}, {{pm}}, {{lockfile}}, {{install_command}},
// {{build_command}}, {{start_command}}, {{port}}.
func BuildPlanFromDefDir(def *FrameworkDef, dir string) (*core.BuildPlan, error) {
	if def == nil {
		return nil, fmt.Errorf("buildPlanFromDef: def must not be nil")
	}

	version := ResolveVersion(def, dir)
	pm := ResolvePM(def, dir)

	vars := map[string]string{
		"{{runtime}}":         def.Runtime,
		"{{version}}":         version,
		"{{pm}}":              pm,
		"{{lockfile}}":        PMLockfileName(pm),
		"{{install_command}}": ResolveInstallCommand(def, pm),
		"{{build_command}}":   def.Defaults.Build,
		"{{start_command}}":   def.Defaults.Start,
		"{{port}}":            fmt.Sprintf("%d", def.Plan.Port),
	}

	stages := make([]core.Stage, 0, len(def.Plan.Stages))
	for _, sd := range def.Plan.Stages {
		s, err := resolveStage(sd, def, vars, version)
		if err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}

	return &core.BuildPlan{
		Framework: def.Name,
		Expose:    def.Plan.Port,
		Stages:    stages,
	}, nil
}

func resolveStage(sd StageDef, def *FrameworkDef, vars map[string]string, version string) (core.Stage, error) {
	from, err := resolveFrom(sd, def, version)
	if err != nil {
		return core.Stage{}, err
	}

	steps := make([]core.Step, 0, len(sd.Steps))
	for _, stepDef := range sd.Steps {
		step, err := resolveStep(stepDef, vars)
		if err != nil {
			return core.Stage{}, err
		}
		steps = append(steps, step)
	}

	return core.Stage{
		Name:  sd.Name,
		From:  from,
		Steps: steps,
	}, nil
}

func resolveFrom(sd StageDef, def *FrameworkDef, version string) (string, error) {
	if sd.Base != "" {
		return plan.ResolveDockerTag(sd.Base, version), nil
	}
	if sd.From != "" {
		return sd.From, nil
	}
	return "", fmt.Errorf("stage %q: either base or from must be set", sd.Name)
}

func resolveStep(sd StepDef, vars map[string]string) (core.Step, error) {
	if sd.Workdir != "" {
		return core.Step{
			Type: core.StepWorkdir,
			Args: []string{Sub(sd.Workdir, vars)},
		}, nil
	}
	if sd.CopyFrom != nil {
		return core.Step{
			Type: core.StepCopyFrom,
			CopyFrom: &core.CopyFrom{
				Stage: Sub(sd.CopyFrom.Stage, vars),
				Src:   Sub(sd.CopyFrom.Src, vars),
				Dst:   Sub(sd.CopyFrom.Dst, vars),
			},
			Link: true,
		}, nil
	}
	if len(sd.Copy) > 0 {
		args := make([]string, len(sd.Copy))
		for i, s := range sd.Copy {
			args[i] = Sub(s, vars)
		}
		return core.Step{Type: core.StepCopy, Args: args}, nil
	}
	if sd.Run != "" {
		step := core.Step{
			Type: core.StepRun,
			Args: []string{Sub(sd.Run, vars)},
		}
		if sd.Cache != "" {
			step.CacheMount = &core.CacheMount{Target: Sub(sd.Cache, vars)}
		}
		return step, nil
	}
	if len(sd.Env) > 0 {
		keys := SortedKeys(sd.Env)
		args := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			args = append(args, Sub(k, vars), Sub(sd.Env[k], vars))
		}
		return core.Step{Type: core.StepEnv, Args: args}, nil
	}
	if len(sd.Cmd) > 0 {
		args := make([]string, len(sd.Cmd))
		for i, s := range sd.Cmd {
			args[i] = Sub(s, vars)
		}
		return core.Step{Type: core.StepCmd, Args: args}, nil
	}
	if sd.Expose != "" {
		return core.Step{
			Type: core.StepExpose,
			Args: []string{Sub(sd.Expose, vars)},
		}, nil
	}
	return core.Step{}, fmt.Errorf("empty or malformed step definition")
}
