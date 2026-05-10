// Package policy validates build artifacts against production supply-chain
// requirements without performing network calls.
package policy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/permanu/docksmith/core"
)

// ProductionPolicy describes the supply-chain gates expected before an image is
// promoted to production. Zero-value limits are strict: no critical or high
// vulnerabilities are allowed unless the caller sets a higher maximum.
type ProductionPolicy struct {
	RequireSBOM               bool     `json:"require_sbom"`
	RequireBaseDigest         bool     `json:"require_base_digest"`
	RequireProvenance         bool     `json:"require_provenance"`
	RequireSigning            bool     `json:"require_signing"`
	MaxCriticalVulns          int      `json:"max_critical_vulnerabilities"`
	MaxHighVulns              int      `json:"max_high_vulnerabilities"`
	AllowedRegistries         []string `json:"allowed_registries,omitempty"`
	AllowedImageFamilies      []string `json:"allowed_image_families,omitempty"`
	AllowedSignatureIssuers   []string `json:"allowed_signature_issuers,omitempty"`
	AllowedSignatureSubjects  []string `json:"allowed_signature_subjects,omitempty"`
	AllowedProvenanceBuilders []string `json:"allowed_provenance_builders,omitempty"`
}

// ImageEvidence is the minimal offline evidence needed for checks that are not
// represented in core.BuildManifest.
type ImageEvidence struct {
	Ref                 string `json:"ref"`
	Signed              bool   `json:"signed"`
	HasProvenance       bool   `json:"has_provenance"`
	CriticalVulns       int    `json:"critical_vulnerabilities"`
	HighVulns           int    `json:"high_vulnerabilities"`
	SignatureSubject    string `json:"signature_subject,omitempty"`
	SignatureIssuer     string `json:"signature_issuer,omitempty"`
	TransparencyLogID   string `json:"transparency_log_id,omitempty"`
	ProvenanceBuilderID string `json:"provenance_builder_id,omitempty"`
}

// Violation records one failed policy check.
type Violation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (v Violation) Error() string {
	return v.Message
}

// Validate checks whether the policy definition itself is well-formed.
func (p ProductionPolicy) Validate() []Violation {
	var violations []Violation
	if p.MaxCriticalVulns < 0 {
		violations = append(violations, violation("invalid_max_critical_vulnerabilities", "maximum critical vulnerabilities cannot be negative"))
	}
	if p.MaxHighVulns < 0 {
		violations = append(violations, violation("invalid_max_high_vulnerabilities", "maximum high vulnerabilities cannot be negative"))
	}
	violations = append(violations, validateList("allowed_registry", p.AllowedRegistries, normalizeRegistry)...)
	violations = append(violations, validateList("allowed_image_family", p.AllowedImageFamilies, normalizeFamilyForPolicy)...)
	violations = append(violations, validateList("allowed_signature_issuer", p.AllowedSignatureIssuers, normalizeIdentity)...)
	violations = append(violations, validateList("allowed_signature_subject", p.AllowedSignatureSubjects, normalizeIdentity)...)
	violations = append(violations, validateList("allowed_provenance_builder", p.AllowedProvenanceBuilders, normalizeIdentity)...)
	return violations
}

// Validate checks both the build manifest and image evidence.
func Validate(p ProductionPolicy, m core.BuildManifest, image ImageEvidence) []Violation {
	violations := ValidateManifest(p, m)
	violations = append(violations, ValidateImage(p, image)...)
	return violations
}

// ValidateManifest checks manifest-resident supply-chain metadata.
func ValidateManifest(p ProductionPolicy, m core.BuildManifest) []Violation {
	var violations []Violation
	if p.RequireSBOM && len(m.SBOM) == 0 {
		violations = append(violations, violation("missing_sbom", "SBOM is required"))
	}
	if len(m.SBOM) > 0 && !json.Valid(m.SBOM) {
		violations = append(violations, violation("invalid_sbom", "SBOM must be valid JSON"))
	}
	if p.RequireBaseDigest && !isSHA256Digest(m.BaseImage.Digest) {
		violations = append(violations, violation("missing_base_digest", "base image digest is required"))
	}
	if len(p.AllowedImageFamilies) > 0 {
		base := m.BaseImage.Image
		if base == "" {
			violations = append(violations, violation("missing_base_image", "base image is required for image family policy"))
		} else if !stringAllowed(imageFamily(base), allowedFamilies(p.AllowedImageFamilies)) {
			violations = append(violations, violation("disallowed_image_family", fmt.Sprintf("base image family %q is not allowed", imageFamily(base))))
		}
	}
	return violations
}

// ValidateImage checks image-reference, signing, provenance, and vulnerability
// evidence supplied by the caller. It does not resolve tags or query registries.
func ValidateImage(p ProductionPolicy, image ImageEvidence) []Violation {
	var violations []Violation
	if p.RequireSigning && !image.Signed {
		violations = append(violations, violation("unsigned_image", "image signature is required"))
	}
	if p.RequireSigning && image.Signed && strings.TrimSpace(image.TransparencyLogID) == "" {
		violations = append(violations, violation("missing_transparency_log", "transparency log entry is required for signed images"))
	}
	if image.Signed && len(p.AllowedSignatureIssuers) > 0 {
		issuer := normalizeIdentity(image.SignatureIssuer)
		if issuer == "" {
			violations = append(violations, violation("missing_signature_issuer", "signature issuer is required for signature issuer policy"))
		} else if !stringAllowed(issuer, allowedIdentities(p.AllowedSignatureIssuers)) {
			violations = append(violations, violation("disallowed_signature_issuer", fmt.Sprintf("signature issuer %q is not allowed", image.SignatureIssuer)))
		}
	}
	if image.Signed && len(p.AllowedSignatureSubjects) > 0 {
		subject := normalizeIdentity(image.SignatureSubject)
		if subject == "" {
			violations = append(violations, violation("missing_signature_subject", "signature subject is required for signature subject policy"))
		} else if !stringAllowed(subject, allowedIdentities(p.AllowedSignatureSubjects)) {
			violations = append(violations, violation("disallowed_signature_subject", fmt.Sprintf("signature subject %q is not allowed", image.SignatureSubject)))
		}
	}
	if p.RequireProvenance && !image.HasProvenance {
		violations = append(violations, violation("missing_provenance", "provenance is required"))
	}
	if image.HasProvenance && len(p.AllowedProvenanceBuilders) > 0 {
		builder := normalizeIdentity(image.ProvenanceBuilderID)
		if builder == "" {
			violations = append(violations, violation("missing_provenance_builder", "provenance builder is required for provenance builder policy"))
		} else if !stringAllowed(builder, allowedIdentities(p.AllowedProvenanceBuilders)) {
			violations = append(violations, violation("disallowed_provenance_builder", fmt.Sprintf("provenance builder %q is not allowed", image.ProvenanceBuilderID)))
		}
	}
	if image.CriticalVulns > p.MaxCriticalVulns {
		violations = append(violations, violation("critical_vulnerabilities", fmt.Sprintf("critical vulnerabilities %d exceeds maximum %d", image.CriticalVulns, p.MaxCriticalVulns)))
	}
	if image.HighVulns > p.MaxHighVulns {
		violations = append(violations, violation("high_vulnerabilities", fmt.Sprintf("high vulnerabilities %d exceeds maximum %d", image.HighVulns, p.MaxHighVulns)))
	}
	if len(p.AllowedRegistries) > 0 {
		registry := imageRegistry(image.Ref)
		if registry == "" {
			violations = append(violations, violation("missing_image_ref", "image reference is required for registry policy"))
		} else if !stringAllowed(registry, allowedRegistries(p.AllowedRegistries)) {
			violations = append(violations, violation("disallowed_registry", fmt.Sprintf("image registry %q is not allowed", registry)))
		}
	}
	if len(p.AllowedImageFamilies) > 0 && image.Ref != "" {
		family := imageFamily(image.Ref)
		if !stringAllowed(family, allowedFamilies(p.AllowedImageFamilies)) {
			violations = append(violations, violation("disallowed_image_family", fmt.Sprintf("image family %q is not allowed", family)))
		}
	}
	return violations
}

func violation(code, message string) Violation {
	return Violation{Code: code, Message: message}
}

func validateList(name string, entries []string, normalize func(string) string) []Violation {
	var violations []Violation
	seen := make(map[string]string, len(entries))
	for _, entry := range entries {
		normalized := normalize(entry)
		if normalized == "" {
			violations = append(violations, violation("blank_"+name, fmt.Sprintf("%s cannot contain blank entries", strings.ReplaceAll(name, "_", " "))))
			continue
		}
		if first, ok := seen[normalized]; ok {
			violations = append(violations, violation("duplicate_"+name, fmt.Sprintf("%s %q duplicates %q after normalization", strings.ReplaceAll(name, "_", " "), entry, first)))
			continue
		}
		seen[normalized] = entry
	}
	return violations
}

func isSHA256Digest(digest string) bool {
	if len(digest) != len("sha256:")+64 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	for _, r := range digest[len("sha256:"):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func allowedRegistries(registries []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(registries))
	for _, registry := range registries {
		registry = normalizeRegistry(registry)
		if registry != "" {
			allowed[registry] = struct{}{}
		}
	}
	return allowed
}

func allowedIdentities(identities []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(identities))
	for _, identity := range identities {
		identity = normalizeIdentity(identity)
		if identity != "" {
			allowed[identity] = struct{}{}
		}
	}
	return allowed
}

func allowedFamilies(families []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(families)*2)
	for _, family := range families {
		family = normalizeFamily(family)
		if family == "" {
			continue
		}
		allowed[family] = struct{}{}
		if strings.HasPrefix(family, "library/") {
			allowed[strings.TrimPrefix(family, "library/")] = struct{}{}
		} else if !strings.Contains(family, "/") {
			allowed["library/"+family] = struct{}{}
		}
	}
	return allowed
}

func stringAllowed(value string, allowed map[string]struct{}) bool {
	_, ok := allowed[value]
	return ok
}

func normalizeRegistry(registry string) string {
	registry = strings.ToLower(strings.TrimSpace(registry))
	if registry == "index.docker.io" {
		registry = "docker.io"
	}
	return registry
}

func normalizeIdentity(identity string) string {
	return strings.ToLower(strings.TrimSpace(identity))
}

func normalizeFamilyForPolicy(family string) string {
	return normalizeFamily(family)
}

func imageRegistry(ref string) string {
	registry, _ := splitRegistryAndRepo(ref)
	return registry
}

func imageFamily(ref string) string {
	_, repo := splitRegistryAndRepo(ref)
	return normalizeFamily(repo)
}

func splitRegistryAndRepo(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	ref = strings.TrimPrefix(ref, "docker://")
	ref = strings.SplitN(ref, "@", 2)[0]
	first, rest, hasSlash := strings.Cut(ref, "/")
	if hasSlash && (strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost") {
		registry := strings.ToLower(first)
		if registry == "index.docker.io" {
			registry = "docker.io"
		}
		return registry, rest
	}
	return "docker.io", ref
}

func normalizeFamily(ref string) string {
	ref = strings.ToLower(strings.TrimSpace(ref))
	ref = strings.TrimPrefix(ref, "docker://")
	ref = strings.SplitN(ref, "@", 2)[0]
	if idx := strings.LastIndex(ref, ":"); idx >= 0 && !strings.Contains(ref[idx+1:], "/") {
		ref = ref[:idx]
	}
	ref = strings.Trim(ref, "/")
	return ref
}
