// Package core defines the shared types used across all docksmith layers:
// Framework (detection result), BuildPlan (abstract build steps), Stage,
// Step, CacheMount, SecretMount, and BuildManifest (Permanu substrate contract).
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Framework holds the detection result for a project directory.
// Detect populates it; Plan consumes it to build a multi-stage Dockerfile.
// Zero-value fields are ignored during planning.
type Framework struct {
	Name           string   `json:"name"`
	BuildCommand   string   `json:"build_command"`
	StartCommand   string   `json:"start_command"`
	Port           int      `json:"port"`
	OutputDir      string   `json:"output_dir,omitempty"` // static asset dir (e.g. "dist", ".next"); empty for server frameworks
	NodeVersion    string   `json:"node_version,omitempty"`
	PackageManager string   `json:"package_manager,omitempty"` // npm, pnpm, yarn, bun — drives install commands and lockfile selection
	PythonVersion  string   `json:"python_version,omitempty"`
	PythonPM       string   `json:"python_pm,omitempty"` // pip, poetry, uv, pdm, pipenv — distinct from PackageManager (JS-only)
	GoVersion      string   `json:"go_version,omitempty"`
	SystemDeps     []string `json:"system_deps,omitempty"` // OS packages needed at build time (e.g. libpq-dev for psycopg2)
	PHPVersion     string   `json:"php_version,omitempty"`
	DotnetVersion  string   `json:"dotnet_version,omitempty"`
	JavaVersion    string   `json:"java_version,omitempty"`
	DenoVersion    string   `json:"deno_version,omitempty"`
	BunVersion     string   `json:"bun_version,omitempty"`
}

// DetectorFunc checks a directory and returns a Framework if detected, nil otherwise.
type DetectorFunc func(dir string) *Framework

// ToJSON serializes a Framework to JSON for transport or caching.
func (f *Framework) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}

// FrameworkFromJSON deserializes a Framework from JSON.
func FrameworkFromJSON(data []byte) (*Framework, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty framework data")
	}
	var fw Framework
	if err := json.Unmarshal(data, &fw); err != nil {
		return nil, fmt.Errorf("parse framework: %w", err)
	}
	return &fw, nil
}

// BuildManifest is the Permanu substrate contract record for a single image
// build. Docksmith emits it; Permanu persists it in build_manifests and stamps
// selected fields as OCI labels on the final image (see Wheel #2 for label
// emission). The contract is pinned in permanu/docs/substrate-contract.md —
// field names, JSON tags, and types here must match that document exactly.
type BuildManifest struct {
	SchemaVersion string    `json:"schema_version"` // "1.0"
	BuildID       string    `json:"build_id"`       // uuid-v7
	Commit        string    `json:"commit"`         // full git sha
	CommitShort   string    `json:"commit_short"`   // 8-char prefix
	ReleaseName   string    `json:"release_name"`   // e.g. "amber-otter-42" (Wheel #6)
	BuiltAt       time.Time `json:"built_at"`

	Framework    FrameworkSnapshot `json:"framework"`
	Runtime      RuntimeContract   `json:"runtime"`
	BaseImage    BaseImageRef      `json:"base_image"`
	Dependencies DependencyDigest  `json:"dependencies"`
	SBOM         json.RawMessage   `json:"sbom,omitempty"`  // CycloneDX JSON
	ImageDigest  string            `json:"image_digest"`    // sha256:... (post-build)
	Architectures []string         `json:"architectures"`   // e.g. ["linux/amd64","linux/arm64"]
}

// FrameworkSnapshot captures the detected framework at build time. It is a
// minimal, stable projection of Framework intended for the manifest — the full
// Framework struct is detection-time metadata and not part of the substrate
// contract.
type FrameworkSnapshot struct {
	Name     string `json:"name"`     // "nextjs", "django", "go"
	Version  string `json:"version"`  // detected runtime version
	Detector string `json:"detector"` // which docksmith detector matched
}

// RuntimeContract describes how the final image expects to be run. Dwaar and
// Permanu read this to wire health checks, shutdown handling, and required env.
type RuntimeContract struct {
	Port           int      `json:"port"`
	HealthPath     string   `json:"health_path"`          // "/healthz"
	HealthCmd      string   `json:"health_cmd,omitempty"`
	ShutdownSignal string   `json:"shutdown_signal"`      // "SIGTERM"
	ShutdownGraceS int      `json:"shutdown_grace_s"`     // default 10
	RequiredEnv    []string `json:"required_env"`         // e.g. ["DATABASE_URL"]
	OptionalEnv    []string `json:"optional_env,omitempty"`
}

// BaseImageRef pins the base image reference and its content digest. Digest is
// filled post-pull so that rebuilds are reproducible even if the upstream tag
// moves.
type BaseImageRef struct {
	Image  string `json:"image"`  // "node:22-alpine"
	Digest string `json:"digest"` // sha256:... pinned at build time
}

// DependencyDigest summarizes the dependency graph without shipping the full
// lockfile. LockfileHashes keys on the lockfile filename (e.g. "package-lock.json")
// and values are "sha256:<hex>" of the file contents.
type DependencyDigest struct {
	LockfileHashes map[string]string `json:"lockfile_hashes"`
	DirectCount    int               `json:"direct_count"`
	TotalCount     int               `json:"total_count"`
}

// ManifestSHA returns a sha256 digest of a compact JSON encoding of m, prefixed
// with "sha256:". The output feeds the io.permanu.manifest.sha OCI label emitted
// in Wheel #2 and is the canonical integrity check for the manifest blob.
//
// Callers must treat the returned string as opaque — the exact byte sequence is
// stable for a given BuildManifest value (Go's encoding/json sorts map keys and
// emits compact output with no trailing newline when used via Marshal).
func ManifestSHA(m BuildManifest) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshal manifest: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
