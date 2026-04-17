package docksmith

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuildWithManifestSmoke exercises the Wheel #1 facade end-to-end on a
// trivial Go project: detect -> plan -> emit, with io.permanu.* labels
// stamped on the final stage.
func TestBuildWithManifestSmoke(t *testing.T) {
	dir := t.TempDir()
	// Seed a minimal Go project so Detect picks up the framework.
	must(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module smoke\n\ngo 1.26\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pass a pre-resolved BaseImageDigest to skip the network call; the
	// facade must respect the caller-supplied digest.
	extras := ManifestExtras{
		BuildID:         "018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f99",
		Commit:          "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		ReleaseName:     "smoke-test-1",
		BuiltAt:         time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
		BaseImageDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Architectures:   []string{"linux/amd64"},
	}

	df, m, err := BuildWithManifest(ctx, dir, extras, DetectOptions{})
	if err != nil {
		t.Fatalf("BuildWithManifest: %v", err)
	}
	if df == "" {
		t.Fatal("empty dockerfile")
	}
	if m.BuildID != extras.BuildID {
		t.Errorf("manifest BuildID = %q, want %q", m.BuildID, extras.BuildID)
	}
	if m.Commit != extras.Commit {
		t.Errorf("manifest Commit = %q, want %q", m.Commit, extras.Commit)
	}
	if m.BaseImage.Digest != extras.BaseImageDigest {
		t.Errorf("manifest BaseImage.Digest = %q, want %q", m.BaseImage.Digest, extras.BaseImageDigest)
	}

	// io.permanu labels must be present in the Dockerfile output.
	mustContain := []string{
		`LABEL io.permanu.manifest.schema="1.0"`,
		`LABEL io.permanu.manifest.id="018f2b1a-7a3f-7c2b-9e1a-4c1f2d3e4f99"`,
		`LABEL io.permanu.build.release_name="smoke-test-1"`,
		`LABEL io.permanu.manifest.json=`,
	}
	for _, want := range mustContain {
		if !strings.Contains(df, want) {
			t.Errorf("missing label %q in dockerfile:\n%s", want, df)
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
}
