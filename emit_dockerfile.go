package docksmith

import (
	"fmt"
	"strings"
)

// EmitDockerfile serializes a BuildPlan into a Dockerfile string.
// It emits BuildKit-enhanced syntax (cache mounts, secret mounts, --link copies).
// Returns an empty string when the plan has no stages.
func EmitDockerfile(plan *BuildPlan) string {
	if len(plan.Stages) == 0 {
		return ""
	}

	var b strings.Builder

	// BuildKit syntax directive — must appear first.
	b.WriteString("# syntax=docker/dockerfile:1\n")

	for i, stage := range plan.Stages {
		b.WriteString("\n")
		writeStageHeader(&b, stage, i)
		for _, step := range stage.Steps {
			writeStep(&b, step)
		}
	}

	return b.String()
}

func writeStageHeader(b *strings.Builder, stage Stage, idx int) {
	from := sanitizeDockerfileArg(stage.From)
	name := sanitizeDockerfileArg(stage.Name)
	if idx == 0 && name == "" {
		fmt.Fprintf(b, "FROM %s\n", from)
		return
	}
	if name != "" {
		fmt.Fprintf(b, "FROM %s AS %s\n", from, name)
	} else {
		fmt.Fprintf(b, "FROM %s\n", from)
	}
}

func writeStep(b *strings.Builder, step Step) {
	switch step.Type {
	case StepWorkdir:
		fmt.Fprintf(b, "WORKDIR %s\n", sanitizeDockerfileArg(step.Args[0]))

	case StepCopy:
		args := sanitizeArgs(step.Args)
		if step.Link {
			fmt.Fprintf(b, "COPY --link %s\n", strings.Join(args, " "))
		} else {
			fmt.Fprintf(b, "COPY %s\n", strings.Join(args, " "))
		}

	case StepCopyFrom:
		cf := step.CopyFrom
		if cf == nil {
			return
		}
		if step.Link {
			fmt.Fprintf(b, "COPY --from=%s --link %s %s\n",
				sanitizeDockerfileArg(cf.Stage),
				sanitizeDockerfileArg(cf.Src),
				sanitizeDockerfileArg(cf.Dst))
		} else {
			fmt.Fprintf(b, "COPY --from=%s %s %s\n",
				sanitizeDockerfileArg(cf.Stage),
				sanitizeDockerfileArg(cf.Src),
				sanitizeDockerfileArg(cf.Dst))
		}

	case StepRun:
		writeRun(b, step)

	case StepEnv:
		if len(step.Args) == 2 {
			fmt.Fprintf(b, "ENV %s %s\n",
				sanitizeDockerfileArg(step.Args[0]),
				sanitizeDockerfileArg(step.Args[1]))
		}

	case StepArg:
		if len(step.Args) == 1 {
			fmt.Fprintf(b, "ARG %s\n", sanitizeDockerfileArg(step.Args[0]))
		} else if len(step.Args) == 2 {
			fmt.Fprintf(b, "ARG %s=%s\n",
				sanitizeDockerfileArg(step.Args[0]),
				sanitizeDockerfileArg(step.Args[1]))
		}

	case StepExpose:
		fmt.Fprintf(b, "EXPOSE %s\n", sanitizeDockerfileArg(step.Args[0]))

	case StepCmd:
		fmt.Fprintf(b, "CMD [%s]\n", shellSplit(strings.Join(step.Args, " ")))

	case StepEntrypoint:
		fmt.Fprintf(b, "ENTRYPOINT [%s]\n", shellSplit(strings.Join(step.Args, " ")))

	case StepUser:
		fmt.Fprintf(b, "USER %s\n", sanitizeDockerfileArg(step.Args[0]))

	case StepHealthcheck:
		cmd := sanitizeDockerfileArg(strings.Join(step.Args, " "))
		fmt.Fprintf(b, "HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD %s\n", cmd)
	}
}

func writeRun(b *strings.Builder, step Step) {
	var mounts []string
	if step.CacheMount != nil {
		mounts = append(mounts, fmt.Sprintf("--mount=type=cache,target=%s",
			sanitizeDockerfileArg(step.CacheMount.Target)))
	}
	if step.SecretMount != nil {
		mounts = append(mounts, fmt.Sprintf("--mount=type=secret,id=%s,target=%s",
			sanitizeDockerfileArg(step.SecretMount.ID),
			sanitizeDockerfileArg(step.SecretMount.Target)))
	}

	cmd := sanitizeDockerfileArg(strings.Join(step.Args, " "))
	if len(mounts) > 0 {
		fmt.Fprintf(b, "RUN %s %s\n", strings.Join(mounts, " "), cmd)
	} else {
		fmt.Fprintf(b, "RUN %s\n", cmd)
	}
}

func sanitizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = sanitizeDockerfileArg(a)
	}
	return out
}

// planHasExpose returns true if any stage step already emits an EXPOSE instruction.
func planHasExpose(plan *BuildPlan) bool {
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type == StepExpose {
				return true
			}
		}
	}
	return false
}
