// Package plan converts a detected Framework into an abstract BuildPlan.
// It selects base images, build stages, hardening steps (non-root user,
// tini, health checks, distroless), and BuildKit cache mounts based on
// the runtime. Plans are pure data — no I/O, no side effects.
package plan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/permanu/docksmith/config"
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
		cfg := ResolvePlanConfig(opts)
		// Resolve ImageFamily → RuntimeImage when the caller hasn't set an explicit image.
		if cfg.ImageFamily != "" && cfg.RuntimeImage == nil {
			img, ferr := resolveRuntimeImageForFramework(fw, cfg.ImageFamily)
			if ferr != nil {
				return nil, ferr
			}
			cfg.RuntimeImage = &img
		}
		if err = applyPlanOverrides(plan, cfg); err != nil {
			return nil, err
		}
		if cfg.ContextRoot != nil {
			applyContextRoot(plan, *cfg.ContextRoot)
		}
	}
	return plan, nil
}

// resolveRuntimeImageForFramework maps a framework + family to a runtime image tag.
// It derives the canonical runtime name and version from fw, then delegates to
// ResolveRuntimeImage.
func resolveRuntimeImageForFramework(fw *core.Framework, family string) (string, error) {
	switch {
	case core.IsGoFramework(fw.Name):
		return ResolveRuntimeImage("go", fw.GoVersion, family)
	case core.IsNodeFramework(fw.Name):
		return ResolveRuntimeImage("node", fw.NodeVersion, family)
	case core.IsPythonFramework(fw.Name):
		return ResolveRuntimeImage("python", fw.PythonVersion, family)
	case core.IsRubyFramework(fw.Name):
		return ResolveRuntimeImage("ruby", "", family)
	case core.IsRustFramework(fw.Name):
		return ResolveRuntimeImage("rust", "", family)
	default:
		return "", fmt.Errorf("image_family is not supported for runtime %q", fw.Name)
	}
}

// applyPlanOverrides modifies the plan based on cfg.
// The last stage is always the runtime stage across all plan builders.
// The first stage is the deps/builder stage where install happens.
func applyPlanOverrides(plan *core.BuildPlan, cfg *planConfig) error {
	if len(plan.Stages) == 0 {
		return nil
	}

	first := &plan.Stages[0]
	last := &plan.Stages[len(plan.Stages)-1]
	isSingleStage := len(plan.Stages) == 1

	if isSingleStage {
		if cfg.BaseImage != nil {
			first.From = *cfg.BaseImage
		} else if cfg.RuntimeImage != nil {
			first.From = *cfg.RuntimeImage
		}
	} else {
		if cfg.BaseImage != nil {
			first.From = *cfg.BaseImage
		}
		if cfg.RuntimeImage != nil {
			last.From = *cfg.RuntimeImage
		}
	}

	insertSystemDepsStep(first, cfg.SystemDeps)

	if cfg.InstallCmd != nil {
		replaceLastRun(first, *cfg.InstallCmd)
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
			spec, needsCreate := parseUserSpec(*cfg.User)
			if needsCreate {
				if err := createNamedUser(last, spec); err != nil {
					return err
				}
			}
			last.Steps = append(last.Steps, core.Step{Type: core.StepUser, Args: []string{*cfg.User}})
		}
	}

	if cfg.Healthcheck != nil {
		removeSteps(last, core.StepHealthcheck)
		if *cfg.Healthcheck != "" {
			step := core.Step{Type: core.StepHealthcheck, Args: []string{*cfg.Healthcheck}}
			if cfg.HealthcheckOpts != nil {
				o := core.HealthcheckOpts{
					Interval:    cfg.HealthcheckOpts.Interval,
					Timeout:     cfg.HealthcheckOpts.Timeout,
					StartPeriod: cfg.HealthcheckOpts.StartPeriod,
					Retries:     cfg.HealthcheckOpts.Retries,
				}
				step.HealthcheckOpts = &o
			}
			last.Steps = append(last.Steps, step)
		}
	} else if cfg.HealthcheckOpts != nil {
		// Only opts set — update the opts on any existing HEALTHCHECK step in place.
		for i := range last.Steps {
			if last.Steps[i].Type == core.StepHealthcheck {
				o := core.HealthcheckOpts{
					Interval:    cfg.HealthcheckOpts.Interval,
					Timeout:     cfg.HealthcheckOpts.Timeout,
					StartPeriod: cfg.HealthcheckOpts.StartPeriod,
					Retries:     cfg.HealthcheckOpts.Retries,
				}
				last.Steps[i].HealthcheckOpts = &o
				break
			}
		}
	}

	if cfg.EntrypointScript != "" {
		// Custom entrypoint script: copy the script into the runtime stage,
		// make it executable, and wire it as ENTRYPOINT. Tini is bypassed —
		// the script takes PID-1 responsibility (signal forwarding + zombie reaping).
		// Start.Command becomes CMD so the script receives it as positional args.
		removeSteps(last, core.StepEntrypoint)
		last.Steps = append(last.Steps,
			core.Step{
				Type: core.StepCopy,
				Args: []string{"--chmod=755", cfg.EntrypointScript, "/entrypoint.sh"},
			},
			core.Step{Type: core.StepEntrypoint, Args: []string{"/entrypoint.sh"}},
		)
	} else if cfg.Entrypoint != nil {
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

	if len(cfg.Binaries) > 0 {
		// StartCmd takes precedence over the default CMD set by applyBinaries.
		applyBinaries(plan, cfg.Binaries, cfg.StartCmd == nil)
	}

	if len(cfg.LdFlags) > 0 {
		applyLdFlags(plan, cfg.LdFlags)
	}

	if len(cfg.RuntimeAssets) > 0 {
		applyRuntimeAssets(last, cfg.RuntimeAssets)
	}

	if len(cfg.ExternalTools) > 0 {
		applyExternalTools(plan, cfg.ExternalTools)
	}

	if cfg.UseGoWork {
		applyGoWork(plan)
	}

	if len(cfg.RuntimeSystemDeps) > 0 {
		if err := applyRuntimeSystemDeps(last, cfg.RuntimeSystemDeps); err != nil {
			return err
		}
	}

	return nil
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

// applyRuntimeAssets inserts COPY steps for user-declared assets into the
// runtime stage. Steps are inserted before the first USER directive so that
// any --chown reference to a named user is valid (the user already exists).
// If no USER step is present, falls back to inserting before CMD/ENTRYPOINT.
func applyRuntimeAssets(stage *core.Stage, assets []core.AssetCopy) {
	// Find insertion point: before first USER step (chown target must exist).
	// Fallback: before first CMD or ENTRYPOINT.
	insertIdx := len(stage.Steps)
	for i, s := range stage.Steps {
		if s.Type == core.StepUser {
			insertIdx = i
			break
		}
	}
	if insertIdx == len(stage.Steps) {
		for i, s := range stage.Steps {
			if s.Type == core.StepCmd || s.Type == core.StepEntrypoint {
				insertIdx = i
				break
			}
		}
	}
	copies := make([]core.Step, len(assets))
	for i, a := range assets {
		if a.Chown != "" {
			copies[i] = core.Step{Type: core.StepCopy, Args: []string{"--chown=" + a.Chown, a.Src, a.Dst}}
		} else {
			copies[i] = core.Step{Type: core.StepCopy, Args: []string{a.Src, a.Dst}}
		}
	}
	stage.Steps = append(stage.Steps[:insertIdx], append(copies, stage.Steps[insertIdx:]...)...)
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

// insertSystemDepsStep prepends a system package install step to the stage.
// Deps are sanitized to reject shell metacharacters.
func insertSystemDepsStep(stage *core.Stage, deps []string) {
	safe := sanitizeSysDeps(deps)
	if len(safe) == 0 {
		return
	}
	depList := strings.Join(safe, " ")
	var cmd string
	if strings.Contains(stage.From, "alpine") {
		cmd = "apk add --no-cache -- " + depList
	} else {
		cmd = "apt-get update -qq && apt-get install -y --no-install-recommends -- " + depList + " && rm -rf /var/lib/apt/lists/*"
	}
	sysStep := core.Step{Type: core.StepRun, Args: []string{cmd}}
	insertIdx := 0
	for i, s := range stage.Steps {
		if s.Type == core.StepWorkdir {
			insertIdx = i + 1
		} else {
			break
		}
	}
	stage.Steps = append(stage.Steps[:insertIdx], append([]core.Step{sysStep}, stage.Steps[insertIdx:]...)...)
}

// applyRuntimeSystemDeps installs packages in the runtime stage.
// It detects the package manager from the stage's base image:
//   - alpine  → apk add --no-cache
//   - slim/debian → apt-get install
//   - distroless → error (no package manager)
//
// Deps are sanitized to reject shell metacharacters before use.
func applyRuntimeSystemDeps(stage *core.Stage, deps []string) error {
	safe := sanitizeSysDeps(deps)
	if len(safe) == 0 {
		return nil
	}
	if strings.Contains(stage.From, "distroless") {
		return fmt.Errorf("runtime_config.system_deps: distroless images have no package manager; cannot install %v", safe)
	}
	depList := strings.Join(safe, " ")
	var cmd string
	if strings.Contains(stage.From, "alpine") {
		cmd = "apk add --no-cache " + depList
	} else {
		cmd = "apt-get update -qq && apt-get install -y --no-install-recommends " + depList + " && rm -rf /var/lib/apt/lists/*"
	}
	sysStep := core.Step{Type: core.StepRun, Args: []string{cmd}}
	// Insert after FROM / any initial ENV/ARG steps, before the first COPY or RUN.
	insertIdx := 0
	for i, s := range stage.Steps {
		if s.Type == core.StepEnv || s.Type == core.StepArg {
			insertIdx = i + 1
		} else {
			break
		}
	}
	stage.Steps = append(stage.Steps[:insertIdx], append([]core.Step{sysStep}, stage.Steps[insertIdx:]...)...)
	return nil
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

// applyLdFlags rewrites the `-ldflags=` argument in the Go build RUN step
// of the builder stage. It keeps the default "-w -s" flags and appends
// sorted "-X key=value" pairs for determinism.
// It is a no-op for plans that have no "go build" step.
func applyLdFlags(plan *core.BuildPlan, flags map[string]string) {
	// Build sorted -X pairs for determinism.
	keys := make([]string, 0, len(flags))
	for k := range flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	extra := make([]string, 0, len(keys))
	for _, k := range keys {
		extra = append(extra, fmt.Sprintf("-X %s=%s", k, flags[k]))
	}
	suffix := " " + strings.Join(extra, " ")

	for i := range plan.Stages {
		for j := range plan.Stages[i].Steps {
			step := &plan.Stages[i].Steps[j]
			if step.Type != core.StepRun {
				continue
			}
			for k, arg := range step.Args {
				if !strings.Contains(arg, "go build") || !strings.Contains(arg, "-ldflags=") {
					continue
				}
				// Append -X pairs inside the existing -ldflags="..." value.
				// The hardcoded form is: -ldflags="-w -s"
				// We need to produce:    -ldflags="-w -s -X k=v ..."
				step.Args[k] = strings.Replace(arg, `-ldflags="-w -s"`, fmt.Sprintf(`-ldflags="-w -s%s"`, suffix), 1)
			}
		}
	}
}

// applyBinaries rewrites the Go builder stage to emit one RUN go build step per
// binary and adds corresponding COPY --from=builder steps to the runtime stage.
// When setCmd is true it replaces the CMD in the runtime stage to use the first
// binary's output name. Pass setCmd=false when Start.Command will override CMD later.
// It is a no-op when the plan has no builder stage named "builder".
func applyBinaries(plan *core.BuildPlan, bins []config.Binary, setCmd bool) {
	builderIdx := -1
	runtimeIdx := -1
	for i, s := range plan.Stages {
		if s.Name == "builder" {
			builderIdx = i
		}
		if s.Name == "runtime" {
			runtimeIdx = i
		}
	}
	if builderIdx < 0 || runtimeIdx < 0 {
		return
	}

	builder := &plan.Stages[builderIdx]
	runtime := &plan.Stages[runtimeIdx]

	// Find and remove the existing single go build RUN step, preserving everything else.
	// We need the cache mount from the go mod download step to know the pattern.
	newBuilderSteps := make([]core.Step, 0, len(builder.Steps)+len(bins))
	for _, step := range builder.Steps {
		if step.Type == core.StepRun && len(step.Args) > 0 && strings.Contains(step.Args[0], "go build") {
			// Replace this single-binary build step with one step per binary (and per arch).
			for _, b := range bins {
				outName := b.OutputName
				if outName == "" {
					outName = b.Name
				}
				if len(b.Architectures) == 0 {
					// Single-arch build (unchanged behavior from cap 7).
					newBuilderSteps = append(newBuilderSteps, core.Step{
						Type: core.StepRun,
						Args: []string{fmt.Sprintf(`CGO_ENABLED=0 go build -ldflags="-w -s" -o /out/%s %s`, outName, b.Path)},
					})
				} else {
					// Cross-arch: one RUN per GOARCH.
					for _, arch := range b.Architectures {
						newBuilderSteps = append(newBuilderSteps, core.Step{
							Type: core.StepRun,
							Args: []string{fmt.Sprintf(`CGO_ENABLED=0 GOOS=linux GOARCH=%s go build -ldflags="-w -s" -o /out/%s-%s %s`, arch, outName, arch, b.Path)},
						})
					}
				}
			}
		} else {
			newBuilderSteps = append(newBuilderSteps, step)
		}
	}
	builder.Steps = newBuilderSteps

	// Remove the existing single COPY --from=builder step from runtime stage.
	newRuntimeSteps := make([]core.Step, 0, len(runtime.Steps)+len(bins))
	for _, step := range runtime.Steps {
		if step.Type == core.StepCopyFrom && step.CopyFrom != nil && step.CopyFrom.Stage == "builder" {
			// Replace with one COPY per binary (and per arch).
			for _, b := range bins {
				outName := b.OutputName
				if outName == "" {
					outName = b.Name
				}
				if len(b.Architectures) == 0 {
					newRuntimeSteps = append(newRuntimeSteps, core.Step{
						Type:     core.StepCopyFrom,
						CopyFrom: &core.CopyFrom{Stage: "builder", Src: fmt.Sprintf("/out/%s", outName), Dst: fmt.Sprintf("/usr/local/bin/%s", outName)},
					})
				} else {
					for _, arch := range b.Architectures {
						newRuntimeSteps = append(newRuntimeSteps, core.Step{
							Type:     core.StepCopyFrom,
							CopyFrom: &core.CopyFrom{Stage: "builder", Src: fmt.Sprintf("/out/%s-%s", outName, arch), Dst: fmt.Sprintf("/usr/local/bin/%s-%s", outName, arch)},
						})
					}
				}
			}
		} else {
			newRuntimeSteps = append(newRuntimeSteps, step)
		}
	}
	runtime.Steps = newRuntimeSteps

	if setCmd {
		// No StartCmd override: set CMD to first binary's output name.
		// If the first binary has cross-arch targets, use the first listed arch.
		firstOutName := bins[0].OutputName
		if firstOutName == "" {
			firstOutName = bins[0].Name
		}
		if len(bins[0].Architectures) > 0 {
			firstOutName = fmt.Sprintf("%s-%s", firstOutName, bins[0].Architectures[0])
		}
		removeSteps(runtime, core.StepCmd)
		runtime.Steps = append(runtime.Steps, core.Step{
			Type: core.StepCmd,
			Args: []string{fmt.Sprintf("/usr/local/bin/%s", firstOutName)},
		})
	}
	// When setCmd is false the caller already wrote a CMD via StartCmd — leave it.
}

// applyGoWork patches the builder stage's go.mod COPY step to include go.work*
// alongside go.mod and go.sum*. This is required for projects that use Go
// workspaces (go.work) where sub-modules rely on workspace replace directives.
// It is a no-op when no builder stage is found or no go.mod COPY step exists.
func applyGoWork(plan *core.BuildPlan) {
	for i := range plan.Stages {
		if plan.Stages[i].Name != "builder" {
			continue
		}
		for j := range plan.Stages[i].Steps {
			step := &plan.Stages[i].Steps[j]
			if step.Type != core.StepCopy {
				continue
			}
			// Look for the COPY step that copies go.mod (and go.sum*).
			// Its Args slice ends with "./" destination and starts with "go.mod".
			if len(step.Args) >= 2 && step.Args[0] == "go.mod" {
				// Append go.work* before the destination (last arg).
				dst := step.Args[len(step.Args)-1]
				srcs := step.Args[:len(step.Args)-1]
				// Only add go.work* if not already present.
				hasGoWork := false
				for _, s := range srcs {
					if s == "go.work*" {
						hasGoWork = true
						break
					}
				}
				if !hasGoWork {
					step.Args = append(srcs, "go.work*", dst)
				}
				return
			}
		}
	}
}

// applyExternalTools injects StepFetchTool steps into the appropriate stage.
// "builder" tools go into the first stage (appended).
// "runtime" tools (default) are inserted before the first USER directive so
// they execute as root and can write to system paths. If no USER step exists,
// they are appended at the end of the runtime stage.
func applyExternalTools(plan *core.BuildPlan, tools []config.ExternalTool) {
	if len(plan.Stages) == 0 {
		return
	}
	first := &plan.Stages[0]
	last := &plan.Stages[len(plan.Stages)-1]

	// Collect runtime tool steps to insert in bulk before USER (preserves order).
	var runtimeSteps []core.Step
	for _, t := range tools {
		step := core.Step{
			Type:      core.StepFetchTool,
			Args:      []string{t.Name, t.URL, t.SHA256, t.InstallPath, t.Format},
			SHA256Map: t.SHA256Map,
		}
		if t.Stage == "builder" {
			first.Steps = append(first.Steps, step)
		} else {
			runtimeSteps = append(runtimeSteps, step)
		}
	}

	if len(runtimeSteps) == 0 {
		return
	}

	// Insert runtime tool steps before the first USER directive (root-required ops).
	// Fallback: append at end when no USER is present.
	insertIdx := len(last.Steps)
	for i, s := range last.Steps {
		if s.Type == core.StepUser {
			insertIdx = i
			break
		}
	}
	last.Steps = append(last.Steps[:insertIdx], append(runtimeSteps, last.Steps[insertIdx:]...)...)
}
