package docksmith

// planConfig holds user-specified overrides for Plan().
// A nil pointer means "not set — use default". A non-nil pointer (even &"")
// means "explicitly set — use this value, possibly disabling the feature".
type planConfig struct {
	user         *string
	healthcheck  *string
	runtimeImage *string
	baseImage    *string
	entrypoint   []string
	extraEnv     map[string]string
	expose       *int
	installCmd   *string
	buildCmd     *string
	startCmd     *string
	systemDeps   []string
	noBuildCache bool
}

// PlanOption modifies a planConfig.
type PlanOption interface {
	apply(*planConfig)
}

type planOptionFunc func(*planConfig)

func (f planOptionFunc) apply(c *planConfig) { f(c) }

func WithUser(user string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.user = &user })
}

func WithHealthcheck(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.healthcheck = &cmd })
}

func WithHealthcheckDisabled() PlanOption {
	empty := ""
	return planOptionFunc(func(c *planConfig) { c.healthcheck = &empty })
}

func WithRuntimeImage(image string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.runtimeImage = &image })
}

func WithBaseImage(image string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.baseImage = &image })
}

func WithEntrypoint(args ...string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.entrypoint = args })
}

func WithExtraEnv(env map[string]string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.extraEnv = env })
}

func WithExpose(port int) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.expose = &port })
}

func WithInstallCommand(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.installCmd = &cmd })
}

func WithBuildCommand(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.buildCmd = &cmd })
}

func WithStartCommand(cmd string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.startCmd = &cmd })
}

func WithSystemDeps(deps ...string) PlanOption {
	return planOptionFunc(func(c *planConfig) { c.systemDeps = deps })
}

func WithBuildCacheDisabled() PlanOption {
	return planOptionFunc(func(c *planConfig) { c.noBuildCache = true })
}

func resolvePlanConfig(opts []PlanOption) *planConfig {
	cfg := &planConfig{}
	for _, o := range opts {
		o.apply(cfg)
	}
	return cfg
}
