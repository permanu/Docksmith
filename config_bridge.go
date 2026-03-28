package docksmith

// ConfigToPlanOptions converts Config fields into a PlanOption slice.
func ConfigToPlanOptions(c *Config) ([]PlanOption, error) {
	var opts []PlanOption

	if c.Build.Command != "" {
		opts = append(opts, WithBuildCommand(c.Build.Command))
	}
	if c.Build.NoCache {
		opts = append(opts, WithBuildCacheDisabled())
	}
	if c.Start.Command != "" {
		opts = append(opts, WithStartCommand(c.Start.Command))
	}
	if len(c.Start.Entrypoint) > 0 {
		opts = append(opts, WithEntrypoint(c.Start.Entrypoint...))
	}
	if c.Install.Command != "" {
		opts = append(opts, WithInstallCommand(c.Install.Command))
	}
	if len(c.Install.SystemDeps) > 0 {
		opts = append(opts, WithSystemDeps(c.Install.SystemDeps...))
	}
	if len(c.Env) > 0 {
		opts = append(opts, WithExtraEnv(c.Env))
	}
	if c.RuntimeConfig.Image != "" {
		opts = append(opts, WithRuntimeImage(c.RuntimeConfig.Image))
	}
	if c.RuntimeConfig.Expose > 0 {
		opts = append(opts, WithExpose(c.RuntimeConfig.Expose))
	}
	if c.RuntimeConfig.UserSet {
		opts = append(opts, WithUser(c.RuntimeConfig.User))
	}
	if c.RuntimeConfig.HCSet {
		if c.RuntimeConfig.Healthcheck == "" {
			opts = append(opts, WithHealthcheckDisabled())
		} else {
			opts = append(opts, WithHealthcheck(c.RuntimeConfig.Healthcheck))
		}
	}

	return opts, nil
}
