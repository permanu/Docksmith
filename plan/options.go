package plan

import (
	"strings"

	"github.com/permanu/docksmith/core"
)

// PlanConfig holds user-specified overrides for Plan().
// A nil pointer means "not set — use default". A non-nil pointer (even &"")
// means "explicitly set — use this value, possibly disabling the feature".
type PlanConfig struct {
	User             *string
	Healthcheck      *string
	RuntimeImage     *string
	BaseImage        *string
	ImageFamily      string // "alpine", "slim", or "distroless"; empty = use runtime default
	Entrypoint       []string
	ExtraEnv         map[string]string
	Expose           *int
	InstallCmd       *string
	BuildCmd         *string
	StartCmd         *string
	SystemDeps       []string
	NoBuildCache     bool
	Secrets          []core.SecretMount
	ContextRoot      *string
	LdFlags          map[string]string
	EntrypointScript string // path relative to build context; empty = not set
	RuntimeAssets    []core.AssetCopy
}

// planConfig is an internal alias kept for transition clarity.
type planConfig = PlanConfig

// PlanOption modifies a planConfig.
type PlanOption interface {
	apply(*planConfig)
}

type planOptionFunc func(*planConfig)

func (f planOptionFunc) apply(c *planConfig) { f(c) }

// WithUser sets the USER directive. Empty string removes USER entirely (container runs as root).
func WithUser(user string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.User = &user })
}

func WithHealthcheck(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.Healthcheck = &cmd })
}

// WithHealthcheckDisabled emits HEALTHCHECK NONE, suppressing the default curl/wget check.
func WithHealthcheckDisabled() PlanOption {
	empty := ""
	return planOptionFunc(func(c *planConfig) { c.Healthcheck = &empty })
}

// WithRuntimeImage overrides the final stage's FROM (where the app runs).
func WithRuntimeImage(image string) PlanOption {
	return planOptionFunc(func(c *planConfig) {
		if isValidImageRef(image) {
			c.RuntimeImage = &image
		}
	})
}

// WithBaseImage overrides the build stage's FROM (where deps are installed and code compiles).
func WithBaseImage(image string) PlanOption {
	return planOptionFunc(func(c *planConfig) {
		if isValidImageRef(image) {
			c.BaseImage = &image
		}
	})
}

func isValidImageRef(s string) bool {
	for _, bad := range []string{"&&", "||", "|", ";", "`", "$("} {
		if strings.Contains(s, bad) {
			return false
		}
	}
	return s != ""
}

func WithEntrypoint(args ...string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.Entrypoint = args })
}

// WithEntrypointScript wires a host-side shell script into the image as
// /entrypoint.sh. The plan emits COPY + chmod + ENTRYPOINT steps into the
// runtime stage and demotes Start.Command to CMD (so the script receives it
// as positional arguments). Tini wrapping is bypassed when this option is
// set — the script takes responsibility for PID 1 signal forwarding and
// zombie reaping. path must be relative to the build context (no leading /,
// no .. components).
func WithEntrypointScript(path string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.EntrypointScript = path })
}

func WithExtraEnv(env map[string]string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.ExtraEnv = env })
}

func WithExpose(port int) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.Expose = &port })
}

func WithInstallCommand(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.InstallCmd = &cmd })
}

// WithBuildCommand overrides the build step command. This is a no-op for
// single-stage plans that have no explicit "build" stage.
func WithBuildCommand(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.BuildCmd = &cmd })
}

func WithStartCommand(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.StartCmd = &cmd })
}

func WithSystemDeps(deps ...string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.SystemDeps = deps })
}

func WithBuildCacheDisabled() PlanOption {
	return planOptionFunc(func(c *planConfig) { c.NoBuildCache = true })
}

func WithSecrets(secrets []core.SecretMount) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.Secrets = secrets })
}

func WithContextRoot(appSubdir string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.ContextRoot = &appSubdir })
}

// WithLdFlags sets Go -X linker flags injected into the build step.
// Keys must match ^[a-zA-Z_][a-zA-Z0-9_./]*$ and values must not contain '"' or newline.
// Only honoured by the Go plan builder; ignored for all other runtimes.
func WithLdFlags(flags map[string]string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.LdFlags = flags })
}

// WithImageFamily sets the runtime-stage base image family.
// Accepted values: "alpine", "slim", "distroless".
// When unset the runtime's default is used (Go→distroless, Node→alpine, Python→slim).
func WithImageFamily(family string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.ImageFamily = family })
}

// WithRuntimeAssets appends COPY <src> <dst> steps into the runtime stage.
func WithRuntimeAssets(assets []core.AssetCopy) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.RuntimeAssets = assets })
}

// ResolvePlanConfig resolves a slice of PlanOption into a PlanConfig.
func ResolvePlanConfig(opts []PlanOption) *PlanConfig {
	cfg := &planConfig{}
	for _, o := range opts {
		o.apply(cfg)
	}
	return cfg
}
