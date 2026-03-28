package docksmith

import (
	"fmt"

	"github.com/permanu/docksmith/yamldef"
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
		// Alternating [key, value, ...] args in sorted order for determinism.
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
	return Step{Type: StepRun, Args: []string{""}}, nil // no action set
}

// sortedKeys delegates to yamldef.SortedKeys.
func sortedKeys(m map[string]string) []string {
	return yamldef.SortedKeys(m)
}

// pmLockfileName delegates to yamldef.PMLockfileName.
func pmLockfileName(pm string) string {
	return yamldef.PMLockfileName(pm)
}
