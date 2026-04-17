package core

import (
	"encoding/json"
	"time"
)

// ManifestExtras carries caller-supplied fields that Docksmith cannot derive
// from a detected Framework alone. Permanu passes these into
// ManifestFromFramework when stamping a BuildManifest for a new build.
//
// Pure data — no pointers to mutable state, no I/O. The caller owns the values
// and is responsible for their correctness (e.g. BuildID must be uuid-v7,
// Commit must be a full git sha, BaseImageDigest must be the pinned sha256 of
// the resolved base image).
type ManifestExtras struct {
	Commit          string            // full git sha
	CommitShort     string            // 8-char prefix; if empty, derived from Commit
	ReleaseName     string            // e.g. "amber-otter-42" (Wheel #6)
	BuildID         string            // uuid-v7
	BuiltAt         time.Time         // build timestamp
	BaseImageDigest string            // sha256:... for the resolved base image
	Architectures   []string          // e.g. ["linux/amd64","linux/arm64"]
	LockfileHashes  map[string]string // {"package-lock.json": "sha256:..."}

	// Optional runtime overrides. When zero-valued, defaults are derived from
	// the Framework (Port) or the substrate contract defaults (SIGTERM, 10s,
	// /healthz).
	Port            int
	HealthPath      string
	HealthCmd       string
	ShutdownSignal  string
	ShutdownGraceS  int
	RequiredEnv     []string
	OptionalEnv     []string

	// Optional dependency counts. Zero values are preserved — the caller can
	// populate them from lockfile parsing.
	DirectDeps int
	TotalDeps  int

	// Optional framework detector name. When empty, falls back to Framework.Name.
	Detector string

	// Optional CycloneDX SBOM (JSON). Zero value serializes as omitted.
	SBOM json.RawMessage

	// Optional post-build image digest. Zero value is allowed — Permanu fills
	// it in after pushing the image to the registry.
	ImageDigest string
}

// ManifestFromFramework builds a BuildManifest from an existing Framework plus
// caller-supplied extras. Pure function — no I/O, no side effects. Fields that
// the caller did not set in extras fall back to sensible defaults pinned by
// the substrate contract (schema "1.0", SIGTERM, 10s grace, "/healthz").
//
// The returned manifest is ready for JSON serialization and for emission as
// OCI labels via emit.BuildLabels. Callers should compute ManifestSHA(m) after
// construction if they need the io.permanu.manifest.sha label.
func ManifestFromFramework(f Framework, extras ManifestExtras) BuildManifest {
	port := extras.Port
	if port == 0 {
		port = f.Port
	}
	healthPath := extras.HealthPath
	if healthPath == "" {
		healthPath = "/healthz"
	}
	shutdownSignal := extras.ShutdownSignal
	if shutdownSignal == "" {
		shutdownSignal = "SIGTERM"
	}
	shutdownGrace := extras.ShutdownGraceS
	if shutdownGrace == 0 {
		shutdownGrace = 10
	}
	detector := extras.Detector
	if detector == "" {
		detector = f.Name
	}
	commitShort := extras.CommitShort
	if commitShort == "" && len(extras.Commit) >= 8 {
		commitShort = extras.Commit[:8]
	}
	requiredEnv := extras.RequiredEnv
	if requiredEnv == nil {
		requiredEnv = []string{}
	}
	lockfileHashes := extras.LockfileHashes
	if lockfileHashes == nil {
		lockfileHashes = map[string]string{}
	}
	architectures := extras.Architectures
	if architectures == nil {
		architectures = []string{}
	}

	return BuildManifest{
		SchemaVersion: "1.0",
		BuildID:       extras.BuildID,
		Commit:        extras.Commit,
		CommitShort:   commitShort,
		ReleaseName:   extras.ReleaseName,
		BuiltAt:       extras.BuiltAt,
		Framework: FrameworkSnapshot{
			Name:     f.Name,
			Version:  frameworkVersion(f),
			Detector: detector,
		},
		Runtime: RuntimeContract{
			Port:           port,
			HealthPath:     healthPath,
			HealthCmd:      extras.HealthCmd,
			ShutdownSignal: shutdownSignal,
			ShutdownGraceS: shutdownGrace,
			RequiredEnv:    requiredEnv,
			OptionalEnv:    extras.OptionalEnv,
		},
		BaseImage: BaseImageRef{
			Image:  baseImageFor(f),
			Digest: extras.BaseImageDigest,
		},
		Dependencies: DependencyDigest{
			LockfileHashes: lockfileHashes,
			DirectCount:    extras.DirectDeps,
			TotalCount:     extras.TotalDeps,
		},
		SBOM:          extras.SBOM,
		ImageDigest:   extras.ImageDigest,
		Architectures: architectures,
	}
}

// frameworkVersion picks the most specific runtime version recorded on the
// framework. Priority order matches detector fidelity: language runtimes first,
// then package-manager-adjacent runtimes (bun, deno) which double as language
// hosts.
func frameworkVersion(f Framework) string {
	switch {
	case f.NodeVersion != "":
		return f.NodeVersion
	case f.PythonVersion != "":
		return f.PythonVersion
	case f.GoVersion != "":
		return f.GoVersion
	case f.PHPVersion != "":
		return f.PHPVersion
	case f.DotnetVersion != "":
		return f.DotnetVersion
	case f.JavaVersion != "":
		return f.JavaVersion
	case f.DenoVersion != "":
		return f.DenoVersion
	case f.BunVersion != "":
		return f.BunVersion
	default:
		return ""
	}
}

// baseImageFor returns the base image reference implied by the framework's
// runtime. This mirrors plan.ResolveDockerTag for the common cases; we keep it
// here (rather than importing plan) because core must not depend on plan.
// When no mapping exists the empty string is returned and callers should set
// BaseImageDigest.Image via extras later.
func baseImageFor(f Framework) string {
	// For the manifest we record the language runtime the framework targets,
	// not any Docker-ish transformation. Docksmith emitters may choose to use
	// distroless for the final stage — that digest is captured separately via
	// extras.BaseImageDigest when available, but the manifest base_image.image
	// tracks the language host. If f.Name directly matches a known runtime,
	// prefer that — otherwise infer from populated version fields.
	switch {
	case f.NodeVersion != "":
		return "node:" + f.NodeVersion + "-alpine"
	case f.PythonVersion != "":
		return "python:" + f.PythonVersion + "-slim"
	case f.GoVersion != "":
		return "golang:" + f.GoVersion + "-alpine"
	case f.PHPVersion != "":
		return "php:" + f.PHPVersion + "-fpm-alpine"
	case f.DotnetVersion != "":
		return "mcr.microsoft.com/dotnet/sdk:" + f.DotnetVersion
	case f.JavaVersion != "":
		return "eclipse-temurin:" + f.JavaVersion + "-jdk-alpine"
	case f.DenoVersion != "":
		return "denoland/deno:" + f.DenoVersion
	case f.BunVersion != "":
		return "oven/bun:" + f.BunVersion
	default:
		return ""
	}
}
