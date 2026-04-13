package plan

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
)

// ApplySecretMounts detects private registry usage in dir and wires
// BuildKit secret mounts into the appropriate install step of the plan.
// Returns the list of secrets applied (empty if none detected).
func ApplySecretMounts(plan *core.BuildPlan, dir string) []detect.SecretDef {
	if plan == nil || len(plan.Stages) == 0 {
		return nil
	}

	var secrets []detect.SecretDef
	switch {
	case core.IsNodeFramework(plan.Framework) || core.IsBunFramework(plan.Framework):
		secrets = detect.DetectNodeSecrets(dir)
	case core.IsPythonFramework(plan.Framework):
		secrets = detect.DetectPythonSecrets(dir)
	case core.IsGoFramework(plan.Framework):
		secrets = detect.DetectGoSecrets(dir)
	case core.IsJavaFramework(plan.Framework):
		secrets = detect.DetectJavaSecrets(dir)
	case core.IsRubyFramework(plan.Framework):
		secrets = detect.DetectRubySecrets(dir)
	}

	if len(secrets) == 0 {
		return nil
	}

	for _, s := range secrets {
		if err := validateSecretTarget(s.Target); err != nil {
			slog.Warn("skipping secret with invalid target", "secret_id", s.ID, "target", s.Target, "error", err)
			continue
		}
		if !attachSecretToInstallStep(plan, core.SecretMount{ID: s.ID, Target: s.Target}) {
			slog.Warn("no install step found for secret mount", "secret_id", s.ID)
		}
	}
	return secrets
}

// attachSecretToInstallStep finds the first RUN step that looks like a
// dependency install command and attaches the secret mount to it.
// Searches the first stage (deps/builder) which is where install runs.
func attachSecretToInstallStep(plan *core.BuildPlan, mount core.SecretMount) bool {
	first := &plan.Stages[0]
	for i := range first.Steps {
		step := &first.Steps[i]
		if step.Type != core.StepRun {
			continue
		}
		if looksLikeInstall(strings.Join(step.Args, " ")) {
			step.SecretMounts = append(step.SecretMounts, mount)
			return true
		}
	}
	return false
}

// looksLikeInstall returns true if cmd appears to be a dependency install command.
func looksLikeInstall(cmd string) bool {
	installPatterns := []string{
		"npm ci", "npm install", "pnpm install", "yarn install", "bun install",
		"pip install", "poetry install", "pdm install", "pipenv install", "uv sync",
		"go mod download",
		"mvn dependency", "gradle dependencies",
		"bundle install",
	}
	for _, p := range installPatterns {
		if strings.Contains(cmd, p) {
			return true
		}
	}
	return false
}

// validateSecretTarget rejects paths that attempt traversal or are absolute
// outside expected mount locations.
func validateSecretTarget(target string) error {
	if target == "" {
		return fmt.Errorf("empty secret target path")
	}
	// Reject raw ".." components before cleaning — catches /root/../etc/passwd.
	if strings.Contains(target, "..") {
		return fmt.Errorf("secret target %q contains path traversal", target)
	}
	cleaned := filepath.Clean(target)
	if !filepath.IsAbs(cleaned) {
		return fmt.Errorf("secret target %q must be absolute", target)
	}
	return nil
}

// SecretBuildHint returns a Dockerfile comment documenting the docker build
// invocation needed to supply secrets. Returns "" when no secrets are present.
func SecretBuildHint(secrets []detect.SecretDef) string {
	if len(secrets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Build with secrets:\n")
	args := make([]string, 0, len(secrets))
	for _, s := range secrets {
		args = append(args, fmt.Sprintf("--secret id=%s,src=%s", s.ID, s.Src))
	}
	b.WriteString("# docker build " + strings.Join(args, " ") + " .\n")
	return b.String()
}

// SecretIgnoreFiles returns file patterns that should be added to .dockerignore
// when secrets are detected, preventing credential files from leaking into
// the build context.
func SecretIgnoreFiles() []string {
	return []string{
		".npmrc",
		".netrc",
		"pip.conf",
		"settings.xml",
		".bundle/config",
		"*.pem",
		"*.key",
	}
}
