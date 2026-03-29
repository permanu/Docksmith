// Package plan converts a detected Framework into an abstract BuildPlan.
// It selects base images, build stages, hardening steps (non-root user,
// tini, health checks, distroless), and BuildKit cache mounts based on
// the runtime. Plans are pure data — no I/O, no side effects.
package plan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/permanu/docksmith/core"
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

// applyPlanOverrides modifies the plan based on cfg.
// The last stage is always the runtime stage across all plan builders.
// The first stage is the deps/builder stage where install happens.
func applyPlanOverrides(plan *core.BuildPlan, cfg *planConfig) {
	if len(plan.Stages) == 0 {
		return
	}

	first := &plan.Stages[0]
	last := &plan.Stages[len(plan.Stages)-1]
	isSingleStage := len(plan.Stages) == 1

	if isSingleStage {
		// For single-stage plans, first and last alias the same stage.
		// Apply all overrides to the single stage in a sensible order:
		// base image first, then system deps, then install, then start command.

		// --- Base image override (BaseImage takes priority, RuntimeImage as fallback) ---
		if cfg.BaseImage != nil {
			first.From = *cfg.BaseImage
		} else if cfg.RuntimeImage != nil {
			first.From = *cfg.RuntimeImage
		}

		// --- System dependencies ---
		if len(cfg.SystemDeps) > 0 {
			depList := strings.Join(cfg.SystemDeps, " ")
			var installCmd string
			if strings.Contains(first.From, "alpine") {
				installCmd = "apk add --no-cache " + depList
			} else {
				installCmd = "apt-get update -qq && apt-get install -y --no-install-recommends " + depList + " && rm -rf /var/lib/apt/lists/*"
			}
			sysStep := core.Step{Type: core.StepRun, Args: []string{installCmd}}
			insertIdx := 0
			for i, s := range first.Steps {
				if s.Type == core.StepWorkdir {
					insertIdx = i + 1
				} else {
					break
				}
			}
			first.Steps = append(first.Steps[:insertIdx], append([]core.Step{sysStep}, first.Steps[insertIdx:]...)...)
		}

		// --- Install command override ---
		if cfg.InstallCmd != nil {
			replaceLastRun(first, *cfg.InstallCmd)
		}
	} else {
		// Multi-stage: first is the builder, last is runtime.

		// --- Base image override: replace the first stage's FROM ---
		if cfg.BaseImage != nil {
			first.From = *cfg.BaseImage
		}

		if cfg.RuntimeImage != nil {
			last.From = *cfg.RuntimeImage
		}

		// --- System dependencies: prepend install step to the first stage ---
		if len(cfg.SystemDeps) > 0 {
			depList := strings.Join(cfg.SystemDeps, " ")
			var installCmd string
			if strings.Contains(first.From, "alpine") {
				installCmd = "apk add --no-cache " + depList
			} else {
				installCmd = "apt-get update -qq && apt-get install -y --no-install-recommends " + depList + " && rm -rf /var/lib/apt/lists/*"
			}
			sysStep := core.Step{Type: core.StepRun, Args: []string{installCmd}}
			insertIdx := 0
			for i, s := range first.Steps {
				if s.Type == core.StepWorkdir {
					insertIdx = i + 1
				} else {
					break
				}
			}
			first.Steps = append(first.Steps[:insertIdx], append([]core.Step{sysStep}, first.Steps[insertIdx:]...)...)
		}

		// --- Install command override: replace the last RUN step in the first stage ---
		if cfg.InstallCmd != nil {
			replaceLastRun(first, *cfg.InstallCmd)
		}
	}

	// --- Build command override: replace the last RUN step in the build stage ---
	if cfg.BuildCmd != nil {
		buildStage := findStageByName(plan, "build")
		if buildStage != nil {
			replaceLastRun(buildStage, *cfg.BuildCmd)
		} else if len(plan.Stages) > 1 {
			// Fallback: use the second-to-last stage if no explicit "build" stage.
			replaceLastRun(&plan.Stages[len(plan.Stages)-2], *cfg.BuildCmd)
		}
	}

	// --- Start command override: replace the CMD step in the runtime stage ---
	if cfg.StartCmd != nil {
		removeSteps(last, core.StepCmd)
		last.Steps = append(last.Steps, core.Step{
			Type: core.StepCmd,
			Args: strings.Fields(*cfg.StartCmd),
		})
	}

	// --- Build cache disabled: strip all cache mounts across all stages ---
	if cfg.NoBuildCache {
		for i := range plan.Stages {
			for j := range plan.Stages[i].Steps {
				plan.Stages[i].Steps[j].CacheMount = nil
			}
		}
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

	if len(cfg.Secrets) > 0 {
		applySecrets(plan, cfg.Secrets)
	}
}

// applySecrets attaches secret mounts to RUN steps in install/build stages.
// Config secrets merge with any pre-existing secret mounts; config wins on ID collision.
func applySecrets(plan *core.BuildPlan, secrets []core.SecretMount) {
	for i := range plan.Stages {
		stage := &plan.Stages[i]
		// Skip the final runtime stage — secrets belong on install/build RUN steps.
		if i == len(plan.Stages)-1 && len(plan.Stages) > 1 {
			continue
		}
		for j := range stage.Steps {
			if stage.Steps[j].Type != core.StepRun {
				continue
			}
			stage.Steps[j].SecretMounts = mergeSecrets(stage.Steps[j].SecretMounts, secrets)
		}
	}
}

func mergeSecrets(existing, incoming []core.SecretMount) []core.SecretMount {
	seen := make(map[string]int, len(existing))
	merged := make([]core.SecretMount, len(existing))
	copy(merged, existing)
	for i, sm := range merged {
		seen[sm.ID] = i
	}
	for _, sm := range incoming {
		if idx, ok := seen[sm.ID]; ok {
			merged[idx] = sm
		} else {
			seen[sm.ID] = len(merged)
			merged = append(merged, sm)
		}
	}
	return merged
}

// findStageByName returns a pointer to the named stage, or nil.
func findStageByName(plan *core.BuildPlan, name string) *core.Stage {
	for i := range plan.Stages {
		if plan.Stages[i].Name == name {
			return &plan.Stages[i]
		}
	}
	return nil
}

// replaceLastRun replaces the last StepRun in a stage with the given command.
func replaceLastRun(stage *core.Stage, cmd string) {
	for i := len(stage.Steps) - 1; i >= 0; i-- {
		if stage.Steps[i].Type == core.StepRun {
			stage.Steps[i].Args = []string{cmd}
			return
		}
	}
	// No existing RUN step — append one.
	stage.Steps = append(stage.Steps, core.Step{Type: core.StepRun, Args: []string{cmd}})
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

