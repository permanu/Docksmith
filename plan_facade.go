package docksmith

import "github.com/permanu/docksmith/plan"

// Type aliases re-export plan types for backward compatibility.
type PlanOption = plan.PlanOption

// Plan converts a detected Framework into a BuildPlan.
func Plan(fw *Framework, opts ...PlanOption) (*BuildPlan, error) {
	return plan.Plan(fw, opts...)
}

// Option constructors re-exported for backward compatibility.
var (
	WithUser              = plan.WithUser
	WithHealthcheck       = plan.WithHealthcheck
	WithHealthcheckDisabled = plan.WithHealthcheckDisabled
	WithRuntimeImage      = plan.WithRuntimeImage
	WithBaseImage         = plan.WithBaseImage
	WithEntrypoint        = plan.WithEntrypoint
	WithExtraEnv          = plan.WithExtraEnv
	WithExpose            = plan.WithExpose
	WithInstallCommand    = plan.WithInstallCommand
	WithBuildCommand      = plan.WithBuildCommand
	WithStartCommand      = plan.WithStartCommand
	WithSystemDeps        = plan.WithSystemDeps
	WithBuildCacheDisabled = plan.WithBuildCacheDisabled
)

// ResolveDockerTag returns the default Docker image tag for a runtime.
var ResolveDockerTag = plan.ResolveDockerTag

// FrameworkDefaults returns sensible build and start commands for a framework.
var FrameworkDefaults = plan.FrameworkDefaults

// BuildkitCacheArgs returns --cache-from and --cache-to flags for buildctl/docker buildx.
var BuildkitCacheArgs = plan.BuildkitCacheArgs

// CacheDir returns the BuildKit cache directory for an app.
var CacheDir = plan.CacheDir
