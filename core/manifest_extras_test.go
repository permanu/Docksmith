package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestManifestFromFrameworkFullFields(t *testing.T) {
	built := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	fw := Framework{
		Name:           "nextjs",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	extras := ManifestExtras{
		Commit:          "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
		ReleaseName:     "amber-otter-42",
		BuildID:         "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f50",
		BuiltAt:         built,
		BaseImageDigest: "sha256:deadbeef",
		Architectures:   []string{"linux/amd64", "linux/arm64"},
		LockfileHashes: map[string]string{
			"package-lock.json": "sha256:cafebabe",
		},
		DirectDeps: 42,
		TotalDeps:  317,
	}

	m := ManifestFromFramework(fw, extras)

	if m.SchemaVersion != "1.0" {
		t.Errorf("SchemaVersion = %q, want 1.0", m.SchemaVersion)
	}
	if m.BuildID != extras.BuildID {
		t.Errorf("BuildID = %q, want %q", m.BuildID, extras.BuildID)
	}
	if m.Commit != extras.Commit {
		t.Errorf("Commit = %q, want %q", m.Commit, extras.Commit)
	}
	if m.CommitShort != "a1b2c3d4" {
		t.Errorf("CommitShort = %q, want derived a1b2c3d4", m.CommitShort)
	}
	if m.ReleaseName != extras.ReleaseName {
		t.Errorf("ReleaseName = %q, want %q", m.ReleaseName, extras.ReleaseName)
	}
	if !m.BuiltAt.Equal(built) {
		t.Errorf("BuiltAt = %v, want %v", m.BuiltAt, built)
	}
	if m.Framework.Name != "nextjs" {
		t.Errorf("Framework.Name = %q, want nextjs", m.Framework.Name)
	}
	if m.Framework.Version != "22" {
		t.Errorf("Framework.Version = %q, want 22 (from NodeVersion)", m.Framework.Version)
	}
	if m.Framework.Detector != "nextjs" {
		t.Errorf("Framework.Detector = %q, want nextjs (fallback from Name)", m.Framework.Detector)
	}
	if m.Runtime.Port != 3000 {
		t.Errorf("Runtime.Port = %d, want 3000", m.Runtime.Port)
	}
	if m.Runtime.HealthPath != "/healthz" {
		t.Errorf("Runtime.HealthPath = %q, want /healthz (default)", m.Runtime.HealthPath)
	}
	if m.Runtime.ShutdownSignal != "SIGTERM" {
		t.Errorf("Runtime.ShutdownSignal = %q, want SIGTERM (default)", m.Runtime.ShutdownSignal)
	}
	if m.Runtime.ShutdownGraceS != 10 {
		t.Errorf("Runtime.ShutdownGraceS = %d, want 10 (default)", m.Runtime.ShutdownGraceS)
	}
	if m.BaseImage.Image != "node:22-alpine" {
		t.Errorf("BaseImage.Image = %q, want node:22-alpine (derived from NodeVersion)", m.BaseImage.Image)
	}
	if m.BaseImage.Digest != "sha256:deadbeef" {
		t.Errorf("BaseImage.Digest = %q, want sha256:deadbeef", m.BaseImage.Digest)
	}
	if m.Dependencies.DirectCount != 42 {
		t.Errorf("Dependencies.DirectCount = %d, want 42", m.Dependencies.DirectCount)
	}
	if m.Dependencies.TotalCount != 317 {
		t.Errorf("Dependencies.TotalCount = %d, want 317", m.Dependencies.TotalCount)
	}
	if m.Dependencies.LockfileHashes["package-lock.json"] != "sha256:cafebabe" {
		t.Errorf("LockfileHashes missing entry: %+v", m.Dependencies.LockfileHashes)
	}
	if len(m.Architectures) != 2 {
		t.Errorf("Architectures = %v, want 2 entries", m.Architectures)
	}
}

func TestManifestFromFrameworkOverrides(t *testing.T) {
	fw := Framework{Name: "go", GoVersion: "1.26", Port: 8080}
	extras := ManifestExtras{
		Commit:          "1234567890abcdef1234567890abcdef12345678",
		CommitShort:     "CUSTOM12",
		BuildID:         "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f51",
		Port:            9090,
		HealthPath:      "/status",
		HealthCmd:       "curl -fsS http://127.0.0.1:9090/status",
		ShutdownSignal:  "SIGINT",
		ShutdownGraceS:  30,
		RequiredEnv:     []string{"DATABASE_URL"},
		OptionalEnv:     []string{"SENTRY_DSN"},
		Detector:        "go-main",
		BaseImageDigest: "sha256:1111",
		LockfileHashes:  map[string]string{"go.sum": "sha256:2222"},
		Architectures:   []string{"linux/amd64"},
	}
	m := ManifestFromFramework(fw, extras)

	if m.CommitShort != "CUSTOM12" {
		t.Errorf("CommitShort override failed: %q", m.CommitShort)
	}
	if m.Runtime.Port != 9090 {
		t.Errorf("Port override failed: %d", m.Runtime.Port)
	}
	if m.Runtime.HealthPath != "/status" {
		t.Errorf("HealthPath override failed: %q", m.Runtime.HealthPath)
	}
	if m.Runtime.ShutdownSignal != "SIGINT" {
		t.Errorf("ShutdownSignal override failed: %q", m.Runtime.ShutdownSignal)
	}
	if m.Runtime.ShutdownGraceS != 30 {
		t.Errorf("ShutdownGraceS override failed: %d", m.Runtime.ShutdownGraceS)
	}
	if m.Runtime.HealthCmd != extras.HealthCmd {
		t.Errorf("HealthCmd = %q", m.Runtime.HealthCmd)
	}
	if m.Framework.Detector != "go-main" {
		t.Errorf("Detector override failed: %q", m.Framework.Detector)
	}
	if m.BaseImage.Image != "golang:1.26-alpine" {
		t.Errorf("BaseImage.Image = %q, want golang:1.26-alpine (derived from GoVersion)", m.BaseImage.Image)
	}
}

func TestManifestFromFrameworkOmitsEmptyOptionals(t *testing.T) {
	fw := Framework{Name: "go", GoVersion: "1.26", Port: 8080}
	extras := ManifestExtras{
		Commit:          "1234567890abcdef1234567890abcdef12345678",
		BuildID:         "bid",
		BaseImageDigest: "sha256:aaaa",
	}
	m := ManifestFromFramework(fw, extras)

	// Required slices/maps must be non-nil (not null) when marshaled.
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)

	// These omitempty fields should NOT appear in the JSON.
	mustNotContain := []string{
		`"sbom":`,
		`"health_cmd":`,
		`"optional_env":`,
	}
	for _, bad := range mustNotContain {
		if contains(s, bad) {
			t.Errorf("unexpected omitempty field %q in output:\n%s", bad, s)
		}
	}

	// These required fields must be present, even when zero/empty.
	mustContain := []string{
		`"required_env":[]`,
		`"lockfile_hashes":{}`,
		`"architectures":[]`,
	}
	for _, want := range mustContain {
		if !contains(s, want) {
			t.Errorf("missing required field %q in output:\n%s", want, s)
		}
	}
}

func TestManifestFromFrameworkShortCommitHandlesShort(t *testing.T) {
	// Commit shorter than 8 chars must not panic; CommitShort stays empty.
	fw := Framework{Name: "go", GoVersion: "1.26"}
	extras := ManifestExtras{Commit: "abc"}
	m := ManifestFromFramework(fw, extras)
	if m.CommitShort != "" {
		t.Errorf("CommitShort = %q, want empty for short commit", m.CommitShort)
	}
}

func TestManifestFromFrameworkSBOMPassthrough(t *testing.T) {
	fw := Framework{Name: "go", GoVersion: "1.26"}
	sbom := json.RawMessage(`{"bomFormat":"CycloneDX","specVersion":"1.5"}`)
	extras := ManifestExtras{SBOM: sbom}
	m := ManifestFromFramework(fw, extras)
	if string(m.SBOM) != string(sbom) {
		t.Errorf("SBOM = %s, want %s", m.SBOM, sbom)
	}
}

// contains is a local helper to avoid pulling strings into this test file
// just for Contains — keeps imports minimal.
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
