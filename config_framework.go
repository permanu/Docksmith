package docksmith

// ToPlanOptions converts Config fields into a PlanOption slice.
func (c *Config) ToPlanOptions() ([]PlanOption, error) {
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

// ToFramework converts a Config to a Framework for Dockerfile generation.
func (c *Config) ToFramework() *Framework {
	if c.Dockerfile != "" {
		return &Framework{
			Name:      "dockerfile",
			OutputDir: c.Dockerfile,
		}
	}

	fw := &Framework{
		Name:         c.runtimeToFrameworkName(),
		BuildCommand: c.Build.Command,
		StartCommand: c.Start.Command,
		Port:         c.RuntimeConfig.Expose,
		SystemDeps:   c.Install.SystemDeps,
	}

	switch c.Runtime {
	case "node":
		fw.NodeVersion = c.Version
		fw.PackageManager = c.PackageManager
	case "python":
		fw.PythonVersion = c.Version
		fw.PythonPM = c.PackageManager
	case "go":
		fw.GoVersion = c.Version
		if fw.BuildCommand == "" {
			fw.BuildCommand = "go build -o app ."
		}
	case "php":
		fw.PHPVersion = c.Version
	case "java":
		fw.JavaVersion = c.Version
	case "dotnet":
		fw.DotnetVersion = c.Version
	case "deno":
		fw.DenoVersion = c.Version
	case "bun":
		fw.BunVersion = c.Version
		fw.PackageManager = "bun"
	}

	return fw
}

func (c *Config) runtimeToFrameworkName() string {
	switch c.Runtime {
	case "node":
		return "express"
	case "python":
		return "flask"
	case "go":
		return "go-std"
	case "php":
		return "php"
	case "java":
		return "maven"
	case "dotnet":
		return "aspnet-core"
	case "rust":
		return "rust-generic"
	case "ruby":
		return "rails"
	case "elixir":
		return "elixir-phoenix"
	case "deno":
		return "deno"
	case "bun":
		return "bun"
	case "static":
		return "static"
	default:
		return c.Runtime
	}
}
