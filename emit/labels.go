// labels.go — Wheel #1 Week 2 OCI label emission.
//
// BuildLabels serializes a BuildManifest into Dockerfile LABEL lines under the
// io.permanu.* namespace. The full label set is pinned in
// permanu/docs/substrate-contract.md (Wheel #1 — OCI Label Conventions). Fields
// that are empty / zero-valued are omitted from the output so that downstream
// consumers (Permanu, Dwaar) can treat missing labels as "not set" rather than
// "set to empty string".
//
// Labels are emitted only on the final stage of a multi-stage build (handled
// by EmitDockerfile); intermediate stages carry no metadata so buildkit layer
// caching stays stable across unrelated manifest changes.
package emit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/permanu/docksmith/core"
)

// BuildLabels returns Dockerfile LABEL lines for the manifest. Each line
// already has a trailing newline; callers can join them via strings.Builder.
// Order is deterministic (matches the contract doc) so diffs between builds
// only reflect semantic changes.
//
// Empty / zero-valued fields are skipped with the following exceptions:
//   - io.permanu.manifest.schema is always emitted (constant "1.0" per contract).
//
// The io.permanu.manifest.json label carries a compact JSON encoding of the
// full manifest. Quotes inside JSON are escaped via strconv.Quote so the value
// is a single, safely-quoted Dockerfile string.
func BuildLabels(m core.BuildManifest) []string {
	var out []string

	// Schema version is a constant; always emit so consumers can gate on it.
	out = append(out, labelLine("io.permanu.manifest.schema", "1.0"))

	if m.BuildID != "" {
		out = append(out, labelLine("io.permanu.manifest.id", m.BuildID))
	}
	if sha, err := core.ManifestSHA(m); err != nil {
		// ManifestSHA only fails if json.Marshal fails, which shouldn't happen
		// for well-formed manifests. Log and continue — consumers without the
		// sha label can still parse io.permanu.manifest.json.
		slog.Warn("BuildLabels: ManifestSHA failed",
			slog.String("build_id", m.BuildID),
			slog.String("err", err.Error()),
		)
	} else if sha != "" {
		out = append(out, labelLine("io.permanu.manifest.sha", sha))
	}

	if m.Framework.Name != "" {
		out = append(out, labelLine("io.permanu.framework.name", m.Framework.Name))
	}
	if m.Framework.Version != "" {
		out = append(out, labelLine("io.permanu.framework.version", m.Framework.Version))
	}

	if m.Commit != "" {
		out = append(out, labelLine("io.permanu.build.commit", m.Commit))
	}
	if m.ReleaseName != "" {
		out = append(out, labelLine("io.permanu.build.release_name", m.ReleaseName))
	}

	if m.Runtime.Port != 0 {
		out = append(out, labelLine("io.permanu.runtime.port", strconv.Itoa(m.Runtime.Port)))
	}
	if m.Runtime.HealthPath != "" {
		out = append(out, labelLine("io.permanu.runtime.health_path", m.Runtime.HealthPath))
	}
	if m.Runtime.ShutdownSignal != "" {
		out = append(out, labelLine("io.permanu.runtime.shutdown_signal", m.Runtime.ShutdownSignal))
	}

	if m.BaseImage.Image != "" {
		out = append(out, labelLine("io.permanu.base.image", m.BaseImage.Image))
	}
	if m.BaseImage.Digest != "" {
		out = append(out, labelLine("io.permanu.base.digest", m.BaseImage.Digest))
	}

	// Full manifest JSON as the last label. Compact encoding, escaped via
	// strconv.Quote so embedded quotes do not terminate the Dockerfile string.
	if data, err := json.Marshal(m); err != nil {
		slog.Warn("BuildLabels: json.Marshal manifest failed",
			slog.String("build_id", m.BuildID),
			slog.String("err", err.Error()),
		)
	} else {
		out = append(out, fmt.Sprintf("LABEL io.permanu.manifest.json=%s\n", strconv.Quote(string(data))))
	}

	return out
}

// labelLine emits a single LABEL instruction. The value is sanitized (newlines
// stripped) and wrapped in double quotes. Embedded double quotes are escaped.
func labelLine(key, value string) string {
	clean := SanitizeDockerfileArg(value)
	// Use strconv.Quote to get a valid Go-style quoted string, which matches
	// Dockerfile's own quoting rules for LABEL values (backslash escapes).
	return fmt.Sprintf("LABEL %s=%s\n", key, strconv.Quote(clean))
}

// appendFinalStageLabels writes BuildLabels to b. Exposed to the package for
// EmitDockerfileWithManifest; unexported so external callers use BuildLabels
// directly.
func appendFinalStageLabels(b *strings.Builder, m core.BuildManifest) {
	for _, line := range BuildLabels(m) {
		b.WriteString(line)
	}
}
