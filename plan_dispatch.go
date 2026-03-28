package docksmith

import (
	"fmt"
	"sort"
)

// Plan converts a detected Framework into a BuildPlan.
// Options are applied after the plan is built, overriding defaults.
func Plan(fw *Framework, opts ...PlanOption) (*BuildPlan, error) {
	if fw == nil {
		return nil, fmt.Errorf("%w: nil framework", ErrNotDetected)
	}
	var (
		plan *BuildPlan
		err  error
	)
	switch {
	// Bun detectors must run before Node — bun projects also have package.json.
	case isBunFramework(fw.Name):
		plan, err = planBun(fw)
	case isDenoFramework(fw.Name):
		plan, err = planDeno(fw)
	case isNodeFramework(fw.Name):
		plan, err = planNode(fw)
	case isPythonFramework(fw.Name):
		plan, err = planPython(fw)
	case isGoFramework(fw.Name):
		plan, err = planGo(fw)
	case isRubyFramework(fw.Name):
		plan, err = planRuby(fw)
	case isPHPFramework(fw.Name):
		plan, err = planPHP(fw)
	case isJavaFramework(fw.Name):
		plan, err = planJava(fw)
	case isDotnetFramework(fw.Name):
		plan, err = planDotnet(fw)
	case isRustFramework(fw.Name):
		plan, err = planRust(fw)
	case fw.Name == "elixir-phoenix":
		plan, err = planElixir(fw)
	case fw.Name == "static":
		plan, err = planStatic(fw)
	default:
		return nil, fmt.Errorf("%w: %q", ErrNotDetected, fw.Name)
	}
	if err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		applyPlanOverrides(plan, resolvePlanConfig(opts))
	}
	return plan, nil
}

// applyPlanOverrides modifies the last stage of plan based on cfg.
// The last stage is always the runtime stage across all plan builders.
func applyPlanOverrides(plan *BuildPlan, cfg *planConfig) {
	if len(plan.Stages) == 0 {
		return
	}

	last := &plan.Stages[len(plan.Stages)-1]

	if cfg.runtimeImage != nil {
		last.From = *cfg.runtimeImage
	}

	if cfg.expose != nil {
		plan.Expose = *cfg.expose
		replaceOrAddExpose(last, *cfg.expose)
	}

	if cfg.user != nil {
		removeSteps(last, StepUser)
		if *cfg.user != "" {
			last.Steps = append(last.Steps, Step{Type: StepUser, Args: []string{*cfg.user}})
		}
	}

	if cfg.healthcheck != nil {
		removeSteps(last, StepHealthcheck)
		if *cfg.healthcheck != "" {
			last.Steps = append(last.Steps, Step{Type: StepHealthcheck, Args: []string{*cfg.healthcheck}})
		}
	}

	if cfg.entrypoint != nil {
		removeSteps(last, StepEntrypoint)
		last.Steps = append(last.Steps, Step{Type: StepEntrypoint, Args: cfg.entrypoint})
	}

	if len(cfg.extraEnv) > 0 {
		keys := make([]string, 0, len(cfg.extraEnv))
		for k := range cfg.extraEnv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			last.Steps = append(last.Steps, Step{Type: StepEnv, Args: []string{k, cfg.extraEnv[k]}})
		}
	}
}

func removeSteps(stage *Stage, t StepType) {
	out := stage.Steps[:0]
	for _, s := range stage.Steps {
		if s.Type != t {
			out = append(out, s)
		}
	}
	stage.Steps = out
}

func replaceOrAddExpose(stage *Stage, port int) {
	portStr := fmt.Sprintf("%d", port)
	for i, s := range stage.Steps {
		if s.Type == StepExpose {
			stage.Steps[i].Args = []string{portStr}
			return
		}
	}
	stage.Steps = append(stage.Steps, Step{Type: StepExpose, Args: []string{portStr}})
}

// Bun detectors must run before Node — bun projects also have package.json.
func isBunFramework(name string) bool {
	return name == "bun" || name == "bun-elysia" || name == "bun-hono"
}

func isDenoFramework(name string) bool {
	return name == "deno" || name == "deno-fresh" || name == "deno-oak"
}

func isNodeFramework(name string) bool {
	switch name {
	case "nextjs", "nuxt", "sveltekit", "astro", "remix", "gatsby",
		"vite", "create-react-app", "angular", "vue-cli", "solidstart",
		"nestjs", "express", "fastify":
		return true
	}
	return false
}

func isPythonFramework(name string) bool {
	return name == "django" || name == "fastapi" || name == "flask"
}

func isGoFramework(name string) bool {
	switch name {
	case "go", "go-gin", "go-echo", "go-fiber", "go-std", "go-chi":
		return true
	}
	return false
}

func isRubyFramework(name string) bool {
	return name == "rails" || name == "sinatra"
}

func isPHPFramework(name string) bool {
	return name == "laravel" || name == "wordpress" || name == "symfony" || name == "slim" || name == "php"
}

func isJavaFramework(name string) bool {
	return name == "spring-boot" || name == "quarkus" || name == "micronaut" ||
		name == "maven" || name == "gradle"
}

func isDotnetFramework(name string) bool {
	return name == "aspnet-core" || name == "blazor" || name == "dotnet-worker"
}

func isRustFramework(name string) bool {
	return name == "rust" || name == "rust-generic" || name == "rust-actix" || name == "rust-axum"
}
