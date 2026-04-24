package docksmith

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

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

// Error types
type DetectionError = core.DetectionError
type NearMiss = core.NearMiss

// ---------------------------------------------------------------------------
// Function aliases — re-export from subpackages
// ---------------------------------------------------------------------------

var FrameworkFromJSON = core.FrameworkFromJSON
var ValidateContextRoot = config.ValidateContextRoot

// Wheel #1 substrate contract re-exports.
type BuildManifest = core.BuildManifest
type FrameworkSnapshot = core.FrameworkSnapshot
type RuntimeContract = core.RuntimeContract
type BaseImageRef = core.BaseImageRef
type DependencyDigest = core.DependencyDigest
type ManifestExtras = core.ManifestExtras

var ManifestFromFramework = core.ManifestFromFramework
var ManifestSHA = core.ManifestSHA
var BuildLabels = emit.BuildLabels
var ResolveBaseImageDigest = plan.ResolveBaseImageDigest
var GenerateSBOM = plan.GenerateSBOM

// ErrBaseImageUnresolvable is returned by ResolveBaseImageDigest on registry
// failure. Callers can errors.Is against this to decide whether to proceed
// with an empty digest or abort the build.
var ErrBaseImageUnresolvable = plan.ErrBaseImageUnresolvable

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
var BuildkitMultiArchCacheArgs = plan.BuildkitMultiArchCacheArgs
var CacheDir = plan.CacheDir
var DefaultArchitectures = plan.DefaultArchitectures
var BuildxMultiArchArgs = plan.BuildxMultiArchArgs
var BuildxPushArgs = plan.BuildxPushArgs
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
	df := EmitDockerfile(p)
	if df == "" {
		return "", fmt.Errorf("generate dockerfile: build plan produced no stages")
	}
	return df, nil
}

// Build runs the full pipeline for dir: detect -> plan -> emit.
func Build(dir string, opts ...PlanOption) (string, *Framework, error) {
	return BuildWithOptions(dir, DetectOptions{}, opts...)
}

// BuildWithOptions runs the pipeline with custom detection options.
// When private registry indicators are found, secret mounts are wired
// into the plan and a build hint comment is prepended to the Dockerfile.
func BuildWithOptions(dir string, detectOpts DetectOptions, planOpts ...PlanOption) (string, *Framework, error) {
	ctx := context.Background()
	ctx, span := tracer().Start(ctx, "docksmith.build")
	defer span.End()

	fw, err := detectWithSpan(ctx, dir, detectOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", nil, fmt.Errorf("build: %w", err)
	}
	if fw.Name == "dockerfile" {
		return "", fw, nil
	}

	p, err := planWithSpan(ctx, fw, planOpts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", fw, err
	}
	secrets := ApplySecretMounts(p, dir)
	df := EmitDockerfile(p)
	if df == "" {
		err := fmt.Errorf("build: plan produced no stages for framework %s", fw.Name)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", fw, err
	}
	if hint := SecretBuildHint(secrets); hint != "" {
		df = hint + df
	}
	span.SetAttributes(
		attribute.String("docksmith.detected_language", fw.Name),
		attribute.String("docksmith.dockerfile_path", dir),
	)
	return df, fw, nil
}

// BuildWithManifest runs the full pipeline and returns the Dockerfile string
// together with a BuildManifest that includes io.permanu.* labels in the
// rendered Dockerfile. Callers supply ManifestExtras carrying fields
// Docksmith cannot derive (Commit, ReleaseName, BuildID, BuiltAt) and
// optional overrides.
//
// Side effects:
//   - Resolves base-image digest via ResolveBaseImageDigest when
//     extras.BaseImageDigest is empty (network round-trip; failures are logged
//     and the digest is left empty rather than failing the build).
//   - Generates a CycloneDX SBOM via GenerateSBOM when extras.SBOM is nil
//     (skipped silently when syft is not installed).
//
// The Dockerfile includes LABEL io.permanu.* lines on the final stage only,
// so intermediate stages remain cache-stable across manifest churn.
func BuildWithManifest(ctx context.Context, dir string, extras ManifestExtras, detectOpts DetectOptions, planOpts ...PlanOption) (string, BuildManifest, error) {
	ctx, span := tracer().Start(ctx, "docksmith.build")
	defer span.End()

	fw, err := detectWithSpan(ctx, dir, detectOpts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", BuildManifest{}, fmt.Errorf("build: %w", err)
	}
	if fw.Name == "dockerfile" {
		err := fmt.Errorf("build: framework=%q has its own Dockerfile; manifest-mode not supported", fw.Name)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", BuildManifest{}, err
	}

	p, err := planWithSpan(ctx, fw, planOpts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", BuildManifest{}, err
	}
	secrets := ApplySecretMounts(p, dir)

	// Resolve base-image digest when the caller didn't supply one. Registry
	// failures are non-fatal — the manifest records an empty digest, which
	// downstream consumers can treat as "unresolved at build time".
	if extras.BaseImageDigest == "" {
		// Pick the final stage's base image (the runtime layer) for pinning.
		// Intermediate build stages use their own base images and are not
		// part of the shipped artifact.
		if baseRef := finalStageImage(p); baseRef != "" && core.IsImageRef(baseRef) {
			digest, derr := resolveBaseImageDigestWithSpan(ctx, baseRef)
			if derr != nil {
				// Non-fatal: log and continue with empty digest. Downstream
				// consumers treat empty digest as "unresolved at build time".
				slog.Warn("BuildWithManifest: base image digest unresolved",
					slog.String("image", baseRef),
					slog.String("err", derr.Error()),
				)
			} else {
				extras.BaseImageDigest = digest
			}
		}
	}

	// Generate SBOM when the caller didn't supply one. syft-missing is
	// silently skipped (Week 3 best-effort).
	if extras.SBOM == nil {
		sbom, serr := GenerateSBOM(ctx, dir)
		if serr != nil {
			slog.Warn("BuildWithManifest: SBOM generation failed",
				slog.String("context_dir", dir),
				slog.String("err", serr.Error()),
			)
		} else if sbom != nil {
			extras.SBOM = sbom
		}
	}

	m := ManifestFromFramework(*fw, extras)
	df := emit.EmitDockerfileWithManifest(p, &m)
	if df == "" {
		err := fmt.Errorf("build: plan produced no stages for framework %s", fw.Name)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", BuildManifest{}, err
	}
	if hint := SecretBuildHint(secrets); hint != "" {
		df = hint + df
	}
	span.SetAttributes(
		attribute.String("docksmith.detected_language", fw.Name),
		attribute.String("docksmith.dockerfile_path", dir),
		attribute.Int64("docksmith.layers_count", int64(len(p.Stages))),
	)
	return df, m, nil
}

// finalStageImage returns the base image of the final stage in the plan, or
// empty string if the plan has no stages. The final stage is the one that
// produces the shipped image; intermediate stages are build-only.
func finalStageImage(p *BuildPlan) string {
	if p == nil || len(p.Stages) == 0 {
		return ""
	}
	return p.Stages[len(p.Stages)-1].From
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
