package plan

import (
	"github.com/permanu/docksmith/core"
	"strconv"
	"strings"
)

// planRust builds a two-stage BuildPlan for Rust applications.
// Stage "builder" compiles with cargo build --release on a rust-alpine image.
// Stage "runtime" is a minimal alpine image with just the compiled binary.
func planRust(fw *core.Framework) (*core.BuildPlan, error) {
	builderImage := ResolveDockerTag("rust", "")
	runtimeImage := "gcr.io/distroless/cc-debian12:nonroot"

	port := fw.Port
	if port == 0 {
		port = 8080
	}

	startCmd := fw.StartCommand
	if startCmd == "" {
		startCmd = "./app"
	}
	startArgs := strings.Fields(startCmd)

	builder := core.Stage{
		Name: "builder",
		From: builderImage,
		Steps: []core.Step{
			{Type: core.StepRun, Args: []string{"apk add --no-cache musl-dev"}},
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			// Cache dependency compilation by building a stub binary first.
			{Type: core.StepCopy, Args: []string{"Cargo.toml", "Cargo.lock*", "./"}},
			{
				Type: core.StepRun,
				Args: []string{
					"mkdir src && echo 'fn main() {}' > src/main.rs && " +
						"cargo build --release && rm -rf src",
				},
				CacheMount: &core.CacheMount{Target: "/usr/local/cargo/registry"},
			},
			{Type: core.StepCopy, Args: []string{".", "."}},
			{Type: core.StepRun, Args: []string{"cargo build --release"}},
		},
	}

	runtime := core.Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []core.Step{
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/app/target/release", Dst: "."},
				Link:     true,
			},
			{Type: core.StepUser, Args: []string{"nonroot"}},
			{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: core.StepCmd, Args: startArgs},
		},
	}

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "target", "*.log"},
	}, nil
}
