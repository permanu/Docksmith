// Package emit serializes a BuildPlan into a Dockerfile string.
// The output is standard Dockerfile syntax with multi-stage builds,
// BuildKit cache mounts, and no proprietary extensions.
package emit

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
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
	if plan == nil || len(plan.Stages) == 0 {
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
			fmt.Fprintf(b, "ENV %s=%s\n",
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
		fmt.Fprintf(b, "HEALTHCHECK %s CMD %s\n", healthcheckFlags(step.HealthcheckOpts), cmd)

	case core.StepFetchTool:
		if len(step.Args) >= 4 {
			format := ""
			if len(step.Args) >= 5 {
				format = step.Args[4]
			}
			if len(step.SHA256Map) > 0 {
				fmt.Fprintf(b, "RUN %s\n", fetchToolRunMultiArch(step.Args[0], step.Args[1], step.SHA256Map, step.Args[3], format))
			} else {
				fmt.Fprintf(b, "RUN %s\n", fetchToolRun(step.Args[0], step.Args[1], step.Args[2], step.Args[3], format))
			}
		}
	}
}

// healthcheckFlags returns the HEALTHCHECK flag string for a step.
// When opts is nil or all zero, it returns the hardcoded defaults so output is
// byte-identical to the pre-configurable behavior.
func healthcheckFlags(opts *core.HealthcheckOpts) string {
	interval := "30s"
	timeout := "5s"
	startPeriod := "10s"
	retries := 0

	if opts != nil {
		if opts.Interval != "" {
			interval = opts.Interval
		}
		if opts.Timeout != "" {
			timeout = opts.Timeout
		}
		if opts.StartPeriod != "" {
			startPeriod = opts.StartPeriod
		}
		if opts.Retries > 0 {
			retries = opts.Retries
		}
	}

	flags := fmt.Sprintf("--interval=%s --timeout=%s --start-period=%s", interval, timeout, startPeriod)
	if retries > 0 {
		flags += fmt.Sprintf(" --retries=%d", retries)
	}
	return flags
}

// fetchToolRun builds the multi-line shell command for a StepFetchTool step.
// format must be "tar.gz", "zip", "binary", or "" (defaults to "tar.gz").
func fetchToolRun(name, url, sha256, installPath, format string) string {
	if format == "" {
		format = "tar.gz"
	}

	if format == "binary" {
		dest := filepath.Join(installPath, name)
		lines := []string{
			`set -eux; \`,
			`  arch="$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"; \`,
			fmt.Sprintf(`  url="%s"; \`, url),
			fmt.Sprintf(`  mkdir -p %s; \`, installPath),
			fmt.Sprintf(`  curl -fsSL -o %s "$url"; \`, dest),
			fmt.Sprintf(`  echo "%s  %s" | sha256sum -c -; \`, sha256, dest),
			fmt.Sprintf(`  chmod +x %s`, dest),
		}
		return strings.Join(lines, "\n")
	}

	// archive formats: "tar.gz" or "zip"
	var ext string
	if format == "zip" {
		ext = ".zip"
	} else {
		ext = ".tar.gz"
	}
	archive := fmt.Sprintf("/tmp/%s%s", name, ext)

	var extractLine string
	if format == "zip" {
		extractLine = fmt.Sprintf(`  unzip -o %s -d %s; \`, archive, installPath)
	} else {
		extractLine = fmt.Sprintf(`  tar -xzf %s -C %s; \`, archive, installPath)
	}

	lines := []string{
		`set -eux; \`,
		`  arch="$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"; \`,
		fmt.Sprintf(`  url="%s"; \`, url),
		fmt.Sprintf(`  curl -fsSL -o %s "$url"; \`, archive),
		fmt.Sprintf(`  echo "%s  %s" | sha256sum -c -; \`, sha256, archive),
		fmt.Sprintf(`  mkdir -p %s; \`, installPath),
		extractLine,
		fmt.Sprintf(`  rm %s`, archive),
	}
	return strings.Join(lines, "\n")
}

// fetchToolRunMultiArch builds the shell command for a StepFetchTool step when
// per-arch sha256 values are provided. It emits a case statement keyed on
// $(uname -m) → normalised arch, then selects the matching sha256 at build time.
// Arch keys are sorted alphabetically for deterministic output.
func fetchToolRunMultiArch(name, url string, sha256Map map[string]string, installPath, format string) string {
	if format == "" {
		format = "tar.gz"
	}

	// Sort arch keys for determinism.
	archs := make([]string, 0, len(sha256Map))
	for arch := range sha256Map {
		archs = append(archs, arch)
	}
	sort.Strings(archs)

	// Build case arms.
	caseArms := make([]string, 0, len(archs)+1)
	for _, arch := range archs {
		caseArms = append(caseArms, fmt.Sprintf(`    %s) sha256="%s" ;;`, arch, sha256Map[arch]))
	}
	caseArms = append(caseArms, `    *) echo "unsupported arch $arch" >&2; exit 1 ;;`)

	header := []string{
		`set -eux; \`,
		`  arch="$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"; \`,
		`  case "$arch" in \`,
	}
	armLines := make([]string, len(caseArms))
	for i, arm := range caseArms {
		armLines[i] = "  " + arm + ` \`
	}
	footer := []string{
		`  esac; \`,
		fmt.Sprintf(`  url="%s"; \`, url),
	}

	if format == "binary" {
		dest := filepath.Join(installPath, name)
		tail := []string{
			fmt.Sprintf(`  mkdir -p %s; \`, installPath),
			fmt.Sprintf(`  curl -fsSL -o %s "$url"; \`, dest),
			fmt.Sprintf(`  echo "$sha256  %s" | sha256sum -c -; \`, dest),
			fmt.Sprintf(`  chmod +x %s`, dest),
		}
		all := append(header, armLines...)
		all = append(all, footer...)
		all = append(all, tail...)
		return strings.Join(all, "\n")
	}

	// archive formats
	var ext string
	if format == "zip" {
		ext = ".zip"
	} else {
		ext = ".tar.gz"
	}
	archive := fmt.Sprintf("/tmp/%s%s", name, ext)

	var extractLine string
	if format == "zip" {
		extractLine = fmt.Sprintf(`  unzip -o %s -d %s; \`, archive, installPath)
	} else {
		extractLine = fmt.Sprintf(`  tar -xzf %s -C %s; \`, archive, installPath)
	}

	tail := []string{
		fmt.Sprintf(`  curl -fsSL -o %s "$url"; \`, archive),
		fmt.Sprintf(`  echo "$sha256  %s" | sha256sum -c -; \`, archive),
		fmt.Sprintf(`  mkdir -p %s; \`, installPath),
		extractLine,
		fmt.Sprintf(`  rm %s`, archive),
	}
	all := append(header, armLines...)
	all = append(all, footer...)
	all = append(all, tail...)
	return strings.Join(all, "\n")
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
