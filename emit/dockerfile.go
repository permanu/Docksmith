// Package emit serializes a BuildPlan into a Dockerfile string.
// The output is standard Dockerfile syntax with multi-stage builds,
// BuildKit cache mounts, and no proprietary extensions.
package emit

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/permanu/docksmith/core"
)

// EmitDockerfile serializes a BuildPlan into a Dockerfile string.
// It emits BuildKit-enhanced syntax (cache mounts, secret mounts, --link copies).
// Returns an empty string when the plan has no stages.
func EmitDockerfile(plan *core.BuildPlan) string {
	return EmitDockerfileWithManifest(plan, nil)
}

// EmitDockerfileWithManifest is EmitDockerfile plus io.permanu.* OCI label
// emission. When m is non-nil, BuildLabels(*m) is appended to the final stage
// (and only the final stage — intermediate stages carry no manifest metadata
// so buildkit layer caching stays stable across unrelated manifest changes).
// Passing nil is equivalent to EmitDockerfile.
func EmitDockerfileWithManifest(plan *core.BuildPlan, m *core.BuildManifest) string {
	if len(plan.Stages) == 0 {
		slog.Warn("EmitDockerfile called with empty build plan")
		return ""
	}

	var b strings.Builder

	// BuildKit syntax directive — must appear first.
	b.WriteString("# syntax=docker/dockerfile:1\n")

	finalIdx := len(plan.Stages) - 1
	for i, stage := range plan.Stages {
		b.WriteString("\n")
		writeStageHeader(&b, stage, i)
		for _, step := range stage.Steps {
			writeStep(&b, step)
		}
		if i == finalIdx && m != nil {
			appendFinalStageLabels(&b, *m)
		}
	}

	return b.String()
}

func writeStageHeader(b *strings.Builder, stage core.Stage, idx int) {
	from := SanitizeDockerfileArg(stage.From)
	name := SanitizeDockerfileArg(stage.Name)
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

func writeStep(b *strings.Builder, step core.Step) {
	switch step.Type {
	case core.StepWorkdir:
		fmt.Fprintf(b, "WORKDIR %s\n", SanitizeDockerfileArg(step.Args[0]))

	case core.StepCopy:
		args := SanitizeArgs(step.Args)
		if step.Link {
			fmt.Fprintf(b, "COPY --link %s\n", strings.Join(args, " "))
		} else {
			fmt.Fprintf(b, "COPY %s\n", strings.Join(args, " "))
		}

	case core.StepCopyFrom:
		cf := step.CopyFrom
		if cf == nil {
			return
		}
		if step.Link {
			fmt.Fprintf(b, "COPY --from=%s --link %s %s\n",
				SanitizeDockerfileArg(cf.Stage),
				SanitizeDockerfileArg(cf.Src),
				SanitizeDockerfileArg(cf.Dst))
		} else {
			fmt.Fprintf(b, "COPY --from=%s %s %s\n",
				SanitizeDockerfileArg(cf.Stage),
				SanitizeDockerfileArg(cf.Src),
				SanitizeDockerfileArg(cf.Dst))
		}

	case core.StepRun:
		writeRun(b, step)

	case core.StepEnv:
		if len(step.Args) == 2 {
			fmt.Fprintf(b, "ENV %s %s\n",
				SanitizeDockerfileArg(step.Args[0]),
				SanitizeDockerfileArg(step.Args[1]))
		}

	case core.StepArg:
		if len(step.Args) == 1 {
			fmt.Fprintf(b, "ARG %s\n", SanitizeDockerfileArg(step.Args[0]))
		} else if len(step.Args) == 2 {
			fmt.Fprintf(b, "ARG %s=%s\n",
				SanitizeDockerfileArg(step.Args[0]),
				SanitizeDockerfileArg(step.Args[1]))
		}

	case core.StepExpose:
		fmt.Fprintf(b, "EXPOSE %s\n", SanitizeDockerfileArg(step.Args[0]))

	case core.StepCmd:
		if step.ShellForm {
			fmt.Fprintf(b, "CMD %s\n", SanitizeDockerfileArg(strings.Join(step.Args, " ")))
		} else {
			fmt.Fprintf(b, "CMD [%s]\n", ShellSplit(strings.Join(step.Args, " ")))
		}

	case core.StepEntrypoint:
		fmt.Fprintf(b, "ENTRYPOINT [%s]\n", ShellSplit(strings.Join(step.Args, " ")))

	case core.StepUser:
		fmt.Fprintf(b, "USER %s\n", SanitizeDockerfileArg(step.Args[0]))

	case core.StepHealthcheck:
		cmd := SanitizeDockerfileArg(strings.Join(step.Args, " "))
		fmt.Fprintf(b, "HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD %s\n", cmd)
	}
}

func writeRun(b *strings.Builder, step core.Step) {
	var mounts []string
	if step.CacheMount != nil {
		mounts = append(mounts, fmt.Sprintf("--mount=type=cache,target=%s",
			SanitizeDockerfileArg(step.CacheMount.Target)))
	}
	for _, sm := range step.SecretMounts {
		mounts = append(mounts, formatSecretMount(sm))
	}

	cmd := SanitizeDockerfileArg(strings.Join(step.Args, " "))
	if len(mounts) > 0 {
		fmt.Fprintf(b, "RUN %s %s\n", strings.Join(mounts, " "), cmd)
	} else {
		fmt.Fprintf(b, "RUN %s\n", cmd)
	}
}

func formatSecretMount(sm core.SecretMount) string {
	id := SanitizeDockerfileArg(sm.ID)
	if sm.Target != "" {
		return fmt.Sprintf("--mount=type=secret,id=%s,target=%s", id, SanitizeDockerfileArg(sm.Target))
	}
	return fmt.Sprintf("--mount=type=secret,id=%s,env=%s", id, SanitizeDockerfileArg(sm.Env))
}

// planHasExpose returns true if any stage step already emits an EXPOSE instruction.
func planHasExpose(plan *core.BuildPlan) bool {
	for _, stage := range plan.Stages {
		for _, step := range stage.Steps {
			if step.Type == core.StepExpose {
				return true
			}
		}
	}
	return false
}
