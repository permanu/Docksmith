package docksmith

import (
	"cmp"
	"fmt"
	"strings"
)

// planPython builds a 2-stage BuildPlan for Python apps.
// Builder stage installs deps into /app/.venv; runtime stage copies only the venv.
func planPython(fw *Framework) (*BuildPlan, error) {
	version := cmp.Or(fw.PythonVersion, "3.12")
	pm := cmp.Or(fw.PythonPM, "pip")
	baseImage := ResolveDockerTag("python", version)

	builderSteps := pythonBuilderSteps(fw, pm)
	runtimeSteps := pythonRuntimeSteps(fw)

	builder := Stage{Name: "builder", From: baseImage, Steps: builderSteps}
	runtime := Stage{Name: "runtime", From: baseImage, Steps: runtimeSteps}

	addTini(&builder, &runtime)
	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "python", fw.Port)

	return &BuildPlan{
		Framework: fw.Name,
		Expose:    fw.Port,
		Stages:    []Stage{builder, runtime},
	}, nil
}

func pythonBuilderSteps(fw *Framework, pm string) []Step {
	steps := []Step{
		{Type: StepWorkdir, Args: []string{"/app"}},
	}

	// build-essential + libpq-dev etc. only needed at compile time
	buildDeps := []string{"build-essential", "libpq-dev", "libffi-dev"}
	buildDeps = append(buildDeps, sanitizeSysDeps(fw.SystemDeps)...)
	steps = append(steps, Step{
		Type: StepRun,
		Args: []string{aptInstall(buildDeps)},
	})

	copyFiles, installCmd, cacheTarget := pythonPMSteps(pm)
	steps = append(steps, Step{Type: StepCopy, Args: copyFiles})
	steps = append(steps, Step{
		Type:       StepRun,
		Args:       []string{installCmd},
		CacheMount: &CacheMount{Target: cacheTarget},
	})
	steps = append(steps, Step{Type: StepCopy, Args: []string{".", "."}})

	return steps
}

func pythonRuntimeSteps(fw *Framework) []Step {
	startCmd := gunicornBind(fw.StartCommand, fw.Port)

	// Runtime only needs the shared-library variant of postgres client, not headers.
	runtimeDeps := []string{"libpq5"}
	runtimeDeps = append(runtimeDeps, sanitizeSysDeps(runtimeSysDeps(fw.SystemDeps))...)

	steps := []Step{
		{Type: StepWorkdir, Args: []string{"/app"}},
		{Type: StepRun, Args: []string{aptInstall(runtimeDeps)}},
		{
			Type:     StepCopyFrom,
			CopyFrom: &CopyFrom{Stage: "builder", Src: "/app/.venv", Dst: "/app/.venv"},
		},
		{
			Type:     StepCopyFrom,
			CopyFrom: &CopyFrom{Stage: "builder", Src: "/app", Dst: "."},
		},
		{Type: StepEnv, Args: []string{"PATH", "/app/.venv/bin:$PATH"}},
		{Type: StepEnv, Args: []string{"PORT", fmt.Sprintf("%d", fw.Port)}},
		{Type: StepExpose, Args: []string{fmt.Sprintf("%d", fw.Port)}},
		{Type: StepCmd, Args: []string{startCmd}, ShellForm: true},
	}
	return steps
}

// pythonPMSteps returns the files to COPY, install RUN command, and cache mount
// target for the given package manager.
func pythonPMSteps(pm string) (copyArgs []string, installCmd, cacheTarget string) {
	switch pm {
	case "uv":
		return []string{"pyproject.toml", "uv.lock*", "./"},
			"pip install --no-cache-dir uv && uv sync --frozen --no-dev --no-editable",
			"/root/.cache/uv"
	case "poetry":
		return []string{"pyproject.toml", "poetry.lock*", "./"},
			"python -m venv /app/.venv && /app/.venv/bin/pip install --no-cache-dir poetry && " +
				"/app/.venv/bin/poetry config virtualenvs.in-project true && " +
				"/app/.venv/bin/poetry install --no-interaction --no-ansi --only main",
			"/root/.cache/pypoetry"
	case "pdm":
		return []string{"pyproject.toml", "pdm.lock*", "./"},
			"python -m venv /app/.venv && /app/.venv/bin/pip install --no-cache-dir pdm && " +
				"/app/.venv/bin/pdm install --no-self --prod",
			"/root/.cache/pip"
	case "pipenv":
		return []string{"Pipfile", "Pipfile.lock*", "./"},
			"python -m venv /app/.venv && VIRTUAL_ENV=/app/.venv /app/.venv/bin/pip install --no-cache-dir pipenv && " +
				"VIRTUAL_ENV=/app/.venv pipenv install --deploy",
			"/root/.cache/pip"
	default: // pip
		return []string{"requirements.txt*", "pyproject.toml*", "./"},
			"python -m venv /app/.venv && " +
				"if [ -f requirements.txt ]; then /app/.venv/bin/pip install --no-cache-dir -r requirements.txt; " +
				"elif [ -f pyproject.toml ]; then /app/.venv/bin/pip install --no-cache-dir .; " +
				"else echo 'WARNING: no requirements.txt or pyproject.toml found'; fi",
			"/root/.cache/pip"
	}
}

// gunicornBind ensures gunicorn commands bind to 0.0.0.0 so the process is
// reachable from outside the container.
func gunicornBind(cmd string, port int) string {
	if !strings.Contains(cmd, "gunicorn") {
		return cmd
	}
	if strings.Contains(cmd, "--bind") || strings.Contains(cmd, "-b ") {
		return cmd
	}
	return cmd + fmt.Sprintf(" --bind 0.0.0.0:%d", port)
}

// sanitizeSysDeps filters to package-name-safe characters only.
func sanitizeSysDeps(deps []string) []string {
	safe := make([]string, 0, len(deps))
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" || !isPackageNameSafe(dep) {
			continue
		}
		safe = append(safe, dep)
	}
	return safe
}

func isPackageNameSafe(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '+' || c == ':') {
			return false
		}
	}
	return true
}

// runtimeSysDeps strips -dev variants from system deps since the runtime
// stage only needs shared libraries, not headers.
func runtimeSysDeps(deps []string) []string {
	runtime := make([]string, 0, len(deps))
	for _, dep := range deps {
		if strings.HasSuffix(dep, "-dev") {
			continue
		}
		runtime = append(runtime, dep)
	}
	return runtime
}

func aptInstall(pkgs []string) string {
	return fmt.Sprintf(
		"apt-get update -qq && apt-get install -y --no-install-recommends %s && rm -rf /var/lib/apt/lists/*",
		strings.Join(pkgs, " "),
	)
}
