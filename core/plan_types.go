package core

import (
	"fmt"
	"strings"
)

// StepType identifies the kind of build instruction.
type StepType int

const (
	StepWorkdir StepType = iota + 1
	StepCopy
	StepCopyFrom
	StepRun
	StepEnv
	StepArg
	StepExpose
	StepCmd
	StepEntrypoint
	StepUser
	StepHealthcheck
	StepFetchTool
)

// BuildPlan is the complete abstract build description produced by Layer 2
// and consumed by Layer 3. It is the contract between planning and emission.
type BuildPlan struct {
	Framework    string   `json:"framework"`
	Stages       []Stage  `json:"stages"`
	Expose       int      `json:"expose"`
	Dockerignore []string `json:"dockerignore,omitempty"`
}

// Stage is a single build stage (e.g., deps, build, runtime).
type Stage struct {
	Name  string `json:"name"`
	From  string `json:"from"` // base image tag or a prior stage name
	Steps []Step `json:"steps"`
}

// HealthcheckOpts carries optional HEALTHCHECK timing parameters.
// Zero values mean "use the hardcoded default". Interval/Timeout/StartPeriod
// must match ^\d+(ms|s|m|h)$ when set. Retries must be 0..10 (0 = omit flag).
type HealthcheckOpts struct {
	Interval    string `json:"interval,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
	StartPeriod string `json:"start_period,omitempty"`
	Retries     int    `json:"retries,omitempty"`
}

// Step is one build instruction within a stage.
type Step struct {
	Type            StepType         `json:"type"`
	Args            []string         `json:"args,omitempty"`
	CacheMount      *CacheMount      `json:"cache_mount,omitempty"`
	SecretMounts    []SecretMount    `json:"secret_mounts,omitempty"`
	CopyFrom        *CopyFrom        `json:"copy_from,omitempty"`
	Link            bool             `json:"link,omitempty"`
	ShellForm       bool             `json:"shell_form,omitempty"` // emit CMD/ENTRYPOINT as shell-form (supports env-var expansion)
	HealthcheckOpts *HealthcheckOpts `json:"healthcheck_opts,omitempty"`
}

// CacheMount describes a BuildKit cache mount for a RUN step.
type CacheMount struct {
	Target string `json:"target"`
}

// SecretMount describes a BuildKit secret mount for a RUN step.
// At least one of Target or Env must be set. When Target is set, the secret
// is mounted as a file. When Env is set, it is injected as an environment variable.
type SecretMount struct {
	ID     string `json:"id"`
	Target string `json:"target,omitempty"`
	Env    string `json:"env,omitempty"`
}

// CopyFrom copies a path from a named prior stage.
type CopyFrom struct {
	Stage string `json:"stage"`
	Src   string `json:"src"`
	Dst   string `json:"dst"`
}

// AssetCopy describes a host file or directory to COPY into the runtime stage.
type AssetCopy struct {
	Src   string `toml:"src"   yaml:"src"   json:"src"`
	Dst   string `toml:"dst"   yaml:"dst"   json:"dst"`
	Chown string `toml:"chown" yaml:"chown" json:"chown,omitempty"` // optional, e.g. "permanu:permanu" or "1000:1000"
}

// Validate checks plan invariants before emission.
func (p *BuildPlan) Validate() error {
	if len(p.Stages) == 0 {
		return fmt.Errorf("%w: no stages defined", ErrInvalidPlan)
	}

	// static sites don't need a port; everything else does.
	if p.Expose <= 0 && p.Framework != "static" {
		return fmt.Errorf("%w: expose port must be > 0 for framework %q", ErrInvalidPlan, p.Framework)
	}

	stageNames := map[string]bool{}
	for _, s := range p.Stages {
		stageNames[s.Name] = true
	}

	for i, s := range p.Stages {
		if len(s.Steps) == 0 {
			return fmt.Errorf("%w: stage %q (index %d) has no steps", ErrInvalidPlan, s.Name, i)
		}
		// A "from" value is either a Docker image reference (contains ":" or "/")
		// or a prior stage name. If it looks like neither, it's invalid.
		if !IsImageRef(s.From) && !stageNames[s.From] {
			return fmt.Errorf("%w: stage %q references unknown from %q", ErrInvalidPlan, s.Name, s.From)
		}
	}

	return nil
}

// IsImageRef returns true if s looks like a Docker image reference rather than
// a stage name. A stage name never contains ":" or "/".
func IsImageRef(s string) bool {
	return strings.ContainsAny(s, ":/")
}
