package plan

import (
	"github.com/permanu/docksmith/core"
	"fmt"
	"sort"
)

// Plan converts a detected Framework into a BuildPlan.
// Options are applied after the plan is built, overriding defaults.
func Plan(fw *core.Framework, opts ...PlanOption) (*core.BuildPlan, error) {
	if fw == nil {
		return nil, fmt.Errorf("%w: nil framework", core.ErrNotDetected)
	}
	var (
		plan *core.BuildPlan
		err  error
	)
	switch {
	// Bun detectors must run before Node — bun projects also have package.json.
	case core.IsBunFramework(fw.Name):
		plan, err = planBun(fw)
	case core.IsDenoFramework(fw.Name):
		plan, err = planDeno(fw)
	case core.IsNodeFramework(fw.Name):
		plan, err = planNode(fw)
	case core.IsPythonFramework(fw.Name):
		plan, err = planPython(fw)
	case core.IsGoFramework(fw.Name):
		plan, err = planGo(fw)
	case core.IsRubyFramework(fw.Name):
		plan, err = planRuby(fw)
	case core.IsPHPFramework(fw.Name):
		plan, err = planPHP(fw)
	case core.IsJavaFramework(fw.Name):
		plan, err = planJava(fw)
	case core.IsDotnetFramework(fw.Name):
		plan, err = planDotnet(fw)
	case core.IsRustFramework(fw.Name):
		plan, err = planRust(fw)
	case fw.Name == "elixir-phoenix":
		plan, err = planElixir(fw)
	case fw.Name == "static":
		plan, err = planStatic(fw)
	default:
		return nil, fmt.Errorf("%w: %q", core.ErrNotDetected, fw.Name)
	}
	if err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		applyPlanOverrides(plan, ResolvePlanConfig(opts))
	}
	return plan, nil
}

// applyPlanOverrides modifies the last stage of plan based on cfg.
// The last stage is always the runtime stage across all plan builders.
func applyPlanOverrides(plan *core.BuildPlan, cfg *planConfig) {
	if len(plan.Stages) == 0 {
		return
	}

	last := &plan.Stages[len(plan.Stages)-1]

	if cfg.RuntimeImage != nil {
		last.From = *cfg.RuntimeImage
	}

	if cfg.Expose != nil {
		plan.Expose = *cfg.Expose
		replaceOrAddExpose(last, *cfg.Expose)
	}

	if cfg.User != nil {
		removeSteps(last, core.StepUser)
		if *cfg.User != "" {
			last.Steps = append(last.Steps, core.Step{Type: core.StepUser, Args: []string{*cfg.User}})
		}
	}

	if cfg.Healthcheck != nil {
		removeSteps(last, core.StepHealthcheck)
		if *cfg.Healthcheck != "" {
			last.Steps = append(last.Steps, core.Step{Type: core.StepHealthcheck, Args: []string{*cfg.Healthcheck}})
		}
	}

	if cfg.Entrypoint != nil {
		removeSteps(last, core.StepEntrypoint)
		last.Steps = append(last.Steps, core.Step{Type: core.StepEntrypoint, Args: cfg.Entrypoint})
	}

	if len(cfg.ExtraEnv) > 0 {
		keys := make([]string, 0, len(cfg.ExtraEnv))
		for k := range cfg.ExtraEnv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			last.Steps = append(last.Steps, core.Step{Type: core.StepEnv, Args: []string{k, cfg.ExtraEnv[k]}})
		}
	}
}

func removeSteps(stage *core.Stage, t core.StepType) {
	out := stage.Steps[:0]
	for _, s := range stage.Steps {
		if s.Type != t {
			out = append(out, s)
		}
	}
	stage.Steps = out
}

func replaceOrAddExpose(stage *core.Stage, port int) {
	portStr := fmt.Sprintf("%d", port)
	for i, s := range stage.Steps {
		if s.Type == core.StepExpose {
			stage.Steps[i].Args = []string{portStr}
			return
		}
	}
	stage.Steps = append(stage.Steps, core.Step{Type: core.StepExpose, Args: []string{portStr}})
}

