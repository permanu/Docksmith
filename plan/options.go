package plan

// PlanConfig holds user-specified overrides for Plan().
// A nil pointer means "not set — use default". A non-nil pointer (even &"")
// means "explicitly set — use this value, possibly disabling the feature".
type PlanConfig struct {
	User         *string
	Healthcheck  *string
	RuntimeImage *string
	BaseImage    *string
	Entrypoint   []string
	ExtraEnv     map[string]string
	Expose       *int
	InstallCmd   *string
	BuildCmd     *string
	StartCmd     *string
	SystemDeps   []string
	NoBuildCache bool
	ContextRoot  *string // app subdirectory relative to context root, e.g. "apps/frontend"
}

// planConfig is an internal alias kept for transition clarity.
type planConfig = PlanConfig

// PlanOption modifies a planConfig.
type PlanOption interface {
	apply(*planConfig)
}

type planOptionFunc func(*planConfig)

func (f planOptionFunc) apply(c *planConfig) { f(c) }

func WithUser(user string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.User = &user })
}

func WithHealthcheck(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.Healthcheck = &cmd })
}

func WithHealthcheckDisabled() PlanOption {
	empty := ""
	return planOptionFunc(func(c *planConfig) { c.Healthcheck = &empty })
}

func WithRuntimeImage(image string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.RuntimeImage = &image })
}

func WithBaseImage(image string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.BaseImage = &image })
}

func WithEntrypoint(args ...string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.Entrypoint = args })
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

// WithContextRoot sets the app subdirectory relative to the context root.
// E.g. "apps/frontend" when the build context is the repo root.
func WithContextRoot(appSubdir string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.ContextRoot = &appSubdir })
}

// ResolvePlanConfig resolves a slice of PlanOption into a PlanConfig.
func ResolvePlanConfig(opts []PlanOption) *PlanConfig {
	cfg := &planConfig{}
	for _, o := range opts {
		o.apply(cfg)
	}
	return cfg
}
