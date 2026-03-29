package docksmith

import (
	"fmt"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/emit"
	"github.com/permanu/docksmith/plan"
	"github.com/permanu/docksmith/registry"
	"github.com/permanu/docksmith/yamldef"
)

// ---------------------------------------------------------------------------
// Type aliases — re-export core types for backward compatibility
// ---------------------------------------------------------------------------

type Framework = core.Framework
type DetectorFunc = core.DetectorFunc

// Plan types
type StepType = core.StepType

const (
	StepWorkdir     = core.StepWorkdir
	StepCopy        = core.StepCopy
	StepCopyFrom    = core.StepCopyFrom
	StepRun         = core.StepRun
	StepEnv         = core.StepEnv
	StepArg         = core.StepArg
	StepExpose      = core.StepExpose
	StepCmd         = core.StepCmd
	StepEntrypoint  = core.StepEntrypoint
	StepUser        = core.StepUser
	StepHealthcheck = core.StepHealthcheck
)

type BuildPlan = core.BuildPlan
type Stage = core.Stage
type Step = core.Step
type CacheMount = core.CacheMount
type SecretMount = core.SecretMount
type CopyFrom = core.CopyFrom

// Config types
type Config = config.Config
type BuildConfig = config.BuildConfig
type StartConfig = config.StartConfig
type InstallConfig = config.InstallConfig
type RuntimeCfg = config.RuntimeCfg
type SecretConfig = config.SecretConfig

// Detect types
type NamedDetector = detect.NamedDetector
type DetectOptions = detect.DetectOptions
type SecretDef = detect.SecretDef

// Plan option type
type PlanOption = plan.PlanOption

// YAML definition types
type FrameworkDef = yamldef.FrameworkDef
type DetectRules = yamldef.DetectRules
type DetectRule = yamldef.DetectRule
type VersionConfig = yamldef.VersionConfig
type VersionSource = yamldef.VersionSource
type PMConfig = yamldef.PMConfig
type PMSource = yamldef.PMSource
type PlanDef = yamldef.PlanDef
type StageDef = yamldef.StageDef
type StepDef = yamldef.StepDef
type CopyFromDef = yamldef.CopyFromDef
type DefaultsDef = yamldef.DefaultsDef
type TestCase = yamldef.TestCase
type TestExpect = yamldef.TestExpect
type TestResult = yamldef.TestResult

// Registry types
const DefaultRegistryURL = registry.DefaultRegistryURL

type RegistryIndex = registry.Index
type RegistryEntry = registry.Entry

// ---------------------------------------------------------------------------
// Sentinel errors — re-export from core
// ---------------------------------------------------------------------------

var (
	ErrNotDetected   = core.ErrNotDetected
	ErrInvalidConfig = core.ErrInvalidConfig
	ErrInvalidPlan   = core.ErrInvalidPlan
)

// ---------------------------------------------------------------------------
// Function aliases — re-export from subpackages
// ---------------------------------------------------------------------------

var FrameworkFromJSON = core.FrameworkFromJSON
var ValidateContextRoot = config.ValidateContextRoot

// Plan option constructors
var (
	WithUser                = plan.WithUser
	WithHealthcheck         = plan.WithHealthcheck
	WithHealthcheckDisabled = plan.WithHealthcheckDisabled
	WithRuntimeImage        = plan.WithRuntimeImage
	WithBaseImage           = plan.WithBaseImage
	WithEntrypoint          = plan.WithEntrypoint
	WithExtraEnv            = plan.WithExtraEnv
	WithExpose              = plan.WithExpose
	WithInstallCommand      = plan.WithInstallCommand
	WithBuildCommand        = plan.WithBuildCommand
	WithStartCommand        = plan.WithStartCommand
	WithSystemDeps          = plan.WithSystemDeps
	WithBuildCacheDisabled  = plan.WithBuildCacheDisabled
	WithSecrets             = plan.WithSecrets
	WithContextRoot         = plan.WithContextRoot
)

var ResolveDockerTag = plan.ResolveDockerTag
var FrameworkDefaults = plan.FrameworkDefaults
var BuildkitCacheArgs = plan.BuildkitCacheArgs
var CacheDir = plan.CacheDir
var ApplySecretMounts = plan.ApplySecretMounts
var SecretBuildHint = plan.SecretBuildHint
var SecretIgnoreFiles = plan.SecretIgnoreFiles

// Registry function aliases
var FetchRegistryIndex = registry.FetchIndex
var SearchRegistry = registry.Search
var InstallFramework = registry.InstallFramework

// ---------------------------------------------------------------------------
// Facade functions — main pipeline
// ---------------------------------------------------------------------------

// GenerateDockerfile runs Plan + EmitDockerfile for the given framework.
// Returns ("", nil) when fw.Name == "dockerfile" (user already has one).
func GenerateDockerfile(fw *Framework, opts ...PlanOption) (string, error) {
	if fw == nil || fw.Name == "dockerfile" {
		return "", nil
	}
	p, err := Plan(fw, opts...)
	if err != nil {
		return "", fmt.Errorf("generate dockerfile: %w", err)
	}
	return EmitDockerfile(p), nil
}

// Build runs the full pipeline for dir: detect -> plan -> emit.
func Build(dir string, opts ...PlanOption) (string, *Framework, error) {
	return BuildWithOptions(dir, DetectOptions{}, opts...)
}

// BuildWithOptions runs the pipeline with custom detection options.
// When private registry indicators are found, secret mounts are wired
// into the plan and a build hint comment is prepended to the Dockerfile.
func BuildWithOptions(dir string, detectOpts DetectOptions, planOpts ...PlanOption) (string, *Framework, error) {
	fw, err := DetectWithOptions(dir, detectOpts)
	if err != nil {
		return "", nil, fmt.Errorf("build: %w", err)
	}
	if fw.Name == "dockerfile" {
		return "", fw, nil
	}
	p, err := Plan(fw, planOpts...)
	if err != nil {
		return "", fw, err
	}
	secrets := ApplySecretMounts(p, dir)
	df := EmitDockerfile(p)
	if hint := SecretBuildHint(secrets); hint != "" {
		df = hint + df
	}
	return df, fw, nil
}

// ---------------------------------------------------------------------------
// Detect facade
// ---------------------------------------------------------------------------

// Detect analyzes dir and returns the detected framework.
func Detect(dir string) (*Framework, error) {
	return detect.Detect(dir)
}

// DetectWithOptions runs detection with alternate config sources or auto-fetch behavior.
func DetectWithOptions(dir string, opts DetectOptions) (*Framework, error) {
	return detect.DetectWithOptions(dir, opts)
}

// RegisterDetector prepends d to the registry.
func RegisterDetector(name string, d DetectorFunc) {
	detect.RegisterDetector(name, d)
}

// RegisterDetectorBefore inserts d immediately before the named detector.
func RegisterDetectorBefore(before, name string, d DetectorFunc) {
	detect.RegisterDetectorBefore(before, name, d)
}

// ConfigToFramework converts a Config to a Framework for Dockerfile generation.
func ConfigToFramework(c *config.Config) *Framework {
	return detect.ConfigToFramework(c)
}

// ---------------------------------------------------------------------------
// Plan facade
// ---------------------------------------------------------------------------

// Plan converts a detected Framework into a BuildPlan.
func Plan(fw *Framework, opts ...PlanOption) (*BuildPlan, error) {
	return plan.Plan(fw, opts...)
}

// ---------------------------------------------------------------------------
// Emit facade
// ---------------------------------------------------------------------------

// EmitDockerfile serializes a BuildPlan into a Dockerfile string.
func EmitDockerfile(p *BuildPlan) string {
	return emit.EmitDockerfile(p)
}

// GenerateDockerignore returns .dockerignore file content tailored to the framework.
func GenerateDockerignore(fw *Framework) string {
	return emit.GenerateDockerignore(fw)
}

// ---------------------------------------------------------------------------
// Config facade
// ---------------------------------------------------------------------------

// LoadConfig reads the first matching config file from dir.
// Returns (nil, nil) if no config file exists.
func LoadConfig(dir string) (*Config, error) {
	return config.Load(dir)
}

// LoadPlanOptions reads the config from dir and converts it to a PlanOption slice.
// Returns nil (not an error) when no config file exists.
func LoadPlanOptions(dir string) ([]PlanOption, error) {
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return ConfigToPlanOptions(cfg)
}

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

	if len(c.Secrets) > 0 {
		opts = append(opts, WithSecrets(configSecretsToMounts(c.Secrets)))
	}

	if c.ContextRoot != "" {
		opts = append(opts, WithContextRoot(c.ContextRoot))
	}

	return opts, nil
}

func configSecretsToMounts(secrets map[string]SecretConfig) []core.SecretMount {
	mounts := make([]core.SecretMount, 0, len(secrets))
	for id, sec := range secrets {
		mounts = append(mounts, core.SecretMount{
			ID:     id,
			Target: sec.Target,
			Env:    sec.Env,
		})
	}
	return mounts
}
