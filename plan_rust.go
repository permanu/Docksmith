package docksmith

import (
	"strconv"
	"strings"
)

// planRust builds a two-stage BuildPlan for Rust applications.
// Stage "builder" compiles with cargo build --release on a rust-alpine image.
// Stage "runtime" is a minimal alpine image with just the compiled binary.
func planRust(fw *Framework) (*BuildPlan, error) {
	builderImage := ResolveDockerTag("rust", "")
	runtimeImage := "gcr.io/distroless/cc-debian12:nonroot"

	port := fw.Port
	if port == 0 {
		port = 8080
	}

	startCmd := fw.StartCommand
	if startCmd == "" {
		startCmd = "./target/release/app"
	}
	startArgs := strings.Fields(startCmd)

	builder := Stage{
		Name: "builder",
		From: builderImage,
		Steps: []Step{
			{Type: StepRun, Args: []string{"apk add --no-cache musl-dev"}},
			{Type: StepWorkdir, Args: []string{"/app"}},
			// Cache dependency compilation by building a stub binary first.
			{Type: StepCopy, Args: []string{"Cargo.toml", "Cargo.lock", "./"}},
			{
				Type: StepRun,
				Args: []string{
					"mkdir src && echo 'fn main() {}' > src/main.rs && " +
						"cargo build --release && rm -rf src",
				},
				CacheMount: &CacheMount{Target: "/usr/local/cargo/registry"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
			{Type: StepRun, Args: []string{"cargo build --release"}},
		},
	}

	runtime := Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app/target/release", Dst: "."},
				Link:     true,
			},
			{Type: StepUser, Args: []string{"nonroot"}},
			{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: StepCmd, Args: startArgs},
		},
	}

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "target", "*.log"},
	}, nil
}
