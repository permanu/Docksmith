package docksmith

import (
	"cmp"
	"fmt"
	"strings"
)

// planGo builds a 2-stage BuildPlan for Go apps.
// Builder compiles a static binary; runtime uses a minimal alpine image.
func planGo(fw *Framework) (*BuildPlan, error) {
	version := cmp.Or(fw.GoVersion, "1.26")
	binary, target := goBuildArgs(fw.BuildCommand)

	builderSteps := goBuilderSteps(binary, target)
	runtimeSteps := goRuntimeSteps(binary, fw.Port)

	return &BuildPlan{
		Framework: fw.Name,
		Expose:    fw.Port,
		Stages: []Stage{
			{Name: "builder", From: ResolveDockerTag("go", version), Steps: builderSteps},
			{Name: "runtime", From: "alpine:3.21", Steps: runtimeSteps},
		},
	}, nil
}

func goBuilderSteps(binary, target string) []Step {
	return []Step{
		{Type: StepWorkdir, Args: []string{"/app"}},
		{Type: StepCopy, Args: []string{"go.mod", "go.sum*", "./"}},
		{
			Type:       StepRun,
			Args:       []string{"go mod download"},
			CacheMount: &CacheMount{Target: "/go/pkg/mod"},
		},
		{Type: StepCopy, Args: []string{".", "."}},
		{
			Type: StepRun,
			Args: []string{fmt.Sprintf("CGO_ENABLED=0 go build -o /app/%s %s", binary, target)},
		},
	}
}

func goRuntimeSteps(binary string, port int) []Step {
	return []Step{
		{Type: StepRun, Args: []string{"apk --no-cache add ca-certificates"}},
		{Type: StepWorkdir, Args: []string{"/app"}},
		{
			Type:     StepCopyFrom,
			CopyFrom: &CopyFrom{Stage: "builder", Src: fmt.Sprintf("/app/%s", binary), Dst: fmt.Sprintf("./%s", binary)},
		},
		{Type: StepExpose, Args: []string{fmt.Sprintf("%d", port)}},
		{Type: StepCmd, Args: []string{fmt.Sprintf("./%s", binary)}},
	}
}

// goBuildArgs extracts the output binary name and build target from a build command.
// Falls back to "app" and "." when not specified.
func goBuildArgs(buildCmd string) (binary, target string) {
	binary = "app"
	target = "."

	if buildCmd == "" {
		return binary, target
	}

	parts := strings.Fields(buildCmd)

	// Extract -o value
	for i, p := range parts {
		if p == "-o" && i+1 < len(parts) {
			binary = parts[i+1]
		}
	}

	// Extract build target: last non-flag arg that isn't "go" or "build" or the -o value
	skipNext := false
	for i := len(parts) - 1; i >= 0; i-- {
		if skipNext {
			skipNext = false
			continue
		}
		p := parts[i]
		if p == binary || p == "go" || p == "build" {
			continue
		}
		if strings.HasPrefix(p, "-") {
			// Two-arg flags: skip their value on the next backwards iteration
			if p == "-o" {
				skipNext = true
			}
			continue
		}
		target = p
		break
	}

	return binary, target
}
