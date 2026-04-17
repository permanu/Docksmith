// sbom.go — Wheel #1 Week 3 SBOM generation via syft.
//
// GenerateSBOM invokes the external `syft` CLI against a project directory
// and returns the resulting CycloneDX JSON document. If syft is not on PATH
// the function returns (nil, nil) — SBOM is best-effort in Week 3 and will
// become mandatory in a future enforcement pass.
//
// Timeouts are capped at 60s to avoid stalling the build pipeline on a
// misbehaving syft invocation.
package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// sbomTimeout caps the syft invocation. 60s is generous for typical mono-repos
// and keeps the build pipeline responsive on large scans.
const sbomTimeout = 60 * time.Second

// GenerateSBOM scans contextDir with syft and returns a CycloneDX JSON SBOM
// suitable for embedding in BuildManifest.SBOM. If syft is not installed on
// the host, returns (nil, nil) — the absence of an SBOM is an expected state
// in Week 3 and callers should treat it as "no SBOM for this build" rather
// than a build failure.
//
// On syft invocation failure (non-zero exit, timeout, invalid output) the
// function returns (nil, err) with a wrapped error so the caller can log the
// cause without aborting the build.
func GenerateSBOM(ctx context.Context, contextDir string) (json.RawMessage, error) {
	if contextDir == "" {
		return nil, errors.New("GenerateSBOM: empty contextDir")
	}

	// syft missing is an expected state — the caller should proceed without
	// an SBOM rather than fail the build.
	if _, err := exec.LookPath("syft"); err != nil {
		slog.Info("GenerateSBOM: syft not found on PATH, skipping SBOM generation",
			slog.String("context_dir", contextDir),
		)
		return nil, nil
	}

	runCtx, cancel := context.WithTimeout(ctx, sbomTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "syft", contextDir, "-o", "cyclonedx-json=-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return nil, fmt.Errorf("GenerateSBOM: syft timed out after %s: %w", sbomTimeout, runCtx.Err())
		}
		return nil, fmt.Errorf("GenerateSBOM: syft failed: %w: %s", err, stderr.String())
	}

	raw := stdout.Bytes()
	if len(raw) == 0 {
		return nil, errors.New("GenerateSBOM: syft returned empty output")
	}

	// Validate it's well-formed JSON — guards against callers embedding
	// garbage bytes into BuildManifest.SBOM.
	if !json.Valid(raw) {
		return nil, errors.New("GenerateSBOM: syft output is not valid JSON")
	}

	// Copy the bytes so the caller owns the slice independent of our buffer.
	out := make([]byte, len(raw))
	copy(out, raw)
	return json.RawMessage(out), nil
}
