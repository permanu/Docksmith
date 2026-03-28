package plan

import (
	"github.com/permanu/docksmith/core"
	"strconv"
	"strings"
)

// planElixir builds a two-stage BuildPlan for Elixir applications (Phoenix, plain Mix).
// Stage "builder" fetches deps and compiles a Mix release.
// Stage "runtime" is a minimal alpine image with the release artifacts.
func planElixir(fw *core.Framework) (*core.BuildPlan, error) {
	builderImage := ResolveDockerTag("elixir", "")
	runtimeImage := "alpine:3.21"

	port := fw.Port
	if port == 0 {
		port = 4000
	}

	// mix release produces a self-contained binary at _build/prod/rel/<app>/bin/<app>.
	// We copy rel/ into /app, so the binary is at /app/<app>/bin/<app>.
	// Default to a shell glob that finds it without knowing the app name.
	startArgs := strings.Fields(fw.StartCommand)
	if len(startArgs) == 0 {
		startArgs = []string{"sh", "-c", "/app/*/bin/* start"}
	}

	builder := core.Stage{
		Name: "builder",
		From: builderImage,
		Steps: []core.Step{
			{Type: core.StepRun, Args: []string{"apk add --no-cache git build-base"}},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{Type: core.StepEnv, Args: []string{"MIX_ENV", "prod"}},
			{Type: core.StepRun, Args: []string{"mix local.hex --force && mix local.rebar --force"}},
			{Type: core.StepCopy, Args: []string{"mix.exs", "mix.lock", "./"}},
			{
				Type: core.StepRun,
				Args: []string{"mix deps.get --only prod && mix deps.compile"},
				CacheMount: &core.CacheMount{Target: "/root/.mix"},
			},
			{Type: core.StepCopy, Args: []string{".", "."}},
			{Type: core.StepRun, Args: []string{"mix release"}},
		},
	}

	runtime := core.Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []core.Step{
			{Type: core.StepRun, Args: []string{"apk --no-cache add libstdc++ openssl ncurses-libs"}},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/app/_build/prod/rel", Dst: "."},
				Link:     true,
			},
			{Type: core.StepEnv, Args: []string{"PORT", strconv.Itoa(port)}},
			{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: core.StepCmd, Args: startArgs},
		},
	}

	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "elixir", port)

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "_build", "deps", "*.log"},
	}, nil
}
