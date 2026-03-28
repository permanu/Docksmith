package docksmith

import (
	"strconv"
	"strings"
)

// planElixir builds a two-stage BuildPlan for Elixir applications (Phoenix, plain Mix).
// Stage "builder" fetches deps and compiles a Mix release.
// Stage "runtime" is a minimal alpine image with the release artifacts.
func planElixir(fw *Framework) (*BuildPlan, error) {
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

	builder := Stage{
		Name: "builder",
		From: builderImage,
		Steps: []Step{
			{Type: StepRun, Args: []string{"apk add --no-cache git build-base"}},
			{Type: StepWorkdir, Args: []string{"/app"}},
			{Type: StepEnv, Args: []string{"MIX_ENV", "prod"}},
			{Type: StepRun, Args: []string{"mix local.hex --force && mix local.rebar --force"}},
			{Type: StepCopy, Args: []string{"mix.exs", "mix.lock", "./"}},
			{
				Type: StepRun,
				Args: []string{"mix deps.get --only prod && mix deps.compile"},
				CacheMount: &CacheMount{Target: "/root/.mix"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
			{Type: StepRun, Args: []string{"mix release"}},
		},
	}

	runtime := Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []Step{
			{Type: StepRun, Args: []string{"apk --no-cache add libstdc++ openssl ncurses-libs"}},
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app/_build/prod/rel", Dst: "."},
				Link:     true,
			},
			{Type: StepEnv, Args: []string{"PORT", strconv.Itoa(port)}},
			{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: StepCmd, Args: startArgs},
		},
	}

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "_build", "deps", "*.log"},
	}, nil
}
