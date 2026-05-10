package policy

import (
	"encoding/json"
	"testing"

	"github.com/permanu/docksmith/core"
)

func TestValidateMissingSBOM(t *testing.T) {
	manifest := strictManifest()
	manifest.SBOM = nil

	violations := ValidateManifest(strictPolicy(), manifest)
	assertViolation(t, violations, "missing_sbom")
}

func TestValidateMissingBaseDigest(t *testing.T) {
	manifest := strictManifest()
	manifest.BaseImage.Digest = ""

	violations := ValidateManifest(strictPolicy(), manifest)
	assertViolation(t, violations, "missing_base_digest")
}

func TestValidateUnsignedAndMissingProvenance(t *testing.T) {
	image := strictImage()
	image.Signed = false
	image.HasProvenance = false

	violations := ValidateImage(strictPolicy(), image)
	assertViolation(t, violations, "unsigned_image")
	assertViolation(t, violations, "missing_provenance")
}

func TestValidateDisallowedRegistry(t *testing.T) {
	image := strictImage()
	image.Ref = "evil.example.com/team/app:1.0.0"

	violations := ValidateImage(strictPolicy(), image)
	assertViolation(t, violations, "disallowed_registry")
}

func TestProductionPolicyValidateInvalidPolicy(t *testing.T) {
	policy := strictPolicy()
	policy.MaxCriticalVulns = -1
	policy.MaxHighVulns = -2
	policy.AllowedRegistries = []string{"ghcr.io", " index.docker.io ", "docker.io", " "}
	policy.AllowedImageFamilies = []string{"team/app", "TEAM/APP", ""}
	policy.AllowedSignatureIssuers = []string{"https://issuer.example.com", " HTTPS://ISSUER.EXAMPLE.COM "}
	policy.AllowedSignatureSubjects = []string{"", "https://github.com/team/app/.github/workflows/release.yml@refs/heads/main"}
	policy.AllowedProvenanceBuilders = []string{"https://github.com/actions/runner", "HTTPS://GITHUB.COM/ACTIONS/RUNNER"}

	violations := policy.Validate()
	assertViolation(t, violations, "invalid_max_critical_vulnerabilities")
	assertViolation(t, violations, "invalid_max_high_vulnerabilities")
	assertViolation(t, violations, "blank_allowed_registry")
	assertViolation(t, violations, "duplicate_allowed_registry")
	assertViolation(t, violations, "blank_allowed_image_family")
	assertViolation(t, violations, "duplicate_allowed_image_family")
	assertViolation(t, violations, "duplicate_allowed_signature_issuer")
	assertViolation(t, violations, "blank_allowed_signature_subject")
	assertViolation(t, violations, "duplicate_allowed_provenance_builder")
}

func TestValidateMissingTransparencyLog(t *testing.T) {
	image := strictImage()
	image.TransparencyLogID = ""

	violations := ValidateImage(strictPolicy(), image)
	assertViolation(t, violations, "missing_transparency_log")
}

func TestValidateDisallowedSignatureIssuerAndBuilder(t *testing.T) {
	image := strictImage()
	image.SignatureIssuer = "https://issuer.evil.example.com"
	image.ProvenanceBuilderID = "https://builder.evil.example.com"

	violations := ValidateImage(strictPolicy(), image)
	assertViolation(t, violations, "disallowed_signature_issuer")
	assertViolation(t, violations, "disallowed_provenance_builder")
}

func TestValidatePassingStrictPolicy(t *testing.T) {
	violations := Validate(strictPolicy(), strictManifest(), strictImage())
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
}

func strictPolicy() ProductionPolicy {
	return ProductionPolicy{
		RequireSBOM:          true,
		RequireBaseDigest:    true,
		RequireProvenance:    true,
		RequireSigning:       true,
		MaxCriticalVulns:     0,
		MaxHighVulns:         0,
		AllowedRegistries:    []string{"ghcr.io"},
		AllowedImageFamilies: []string{"team/app", "node"},
		AllowedSignatureIssuers: []string{
			"https://token.actions.githubusercontent.com",
		},
		AllowedSignatureSubjects: []string{
			"https://github.com/team/app/.github/workflows/release.yml@refs/heads/main",
		},
		AllowedProvenanceBuilders: []string{
			"https://github.com/actions/runner",
		},
	}
}

func strictManifest() core.BuildManifest {
	return core.BuildManifest{
		BaseImage: core.BaseImageRef{
			Image:  "node:22-alpine",
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		SBOM: json.RawMessage(`{"bomFormat":"CycloneDX","specVersion":"1.5"}`),
	}
}

func strictImage() ImageEvidence {
	return ImageEvidence{
		Ref:                 "ghcr.io/team/app:1.0.0",
		Signed:              true,
		HasProvenance:       true,
		SignatureIssuer:     "https://token.actions.githubusercontent.com",
		SignatureSubject:    "https://github.com/team/app/.github/workflows/release.yml@refs/heads/main",
		TransparencyLogID:   "rekor:1234567890abcdef",
		ProvenanceBuilderID: "https://github.com/actions/runner",
	}
}

func assertViolation(t *testing.T, violations []Violation, code string) {
	t.Helper()
	for _, violation := range violations {
		if violation.Code == code {
			return
		}
	}
	t.Fatalf("missing violation %q in %#v", code, violations)
}
