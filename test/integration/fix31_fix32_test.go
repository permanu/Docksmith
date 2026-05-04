package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/config"
)

// TestExternalTool_BinaryFormat_MkdirPresent asserts that the emitted Dockerfile
// for an external_tool with format="binary" includes a mkdir -p step before the
// curl download. This prevents curl failing with "No such file or directory" when
// the install_path does not exist in the base image (closes #31).
func TestExternalTool_BinaryFormat_MkdirPresent(t *testing.T) {
	const installPath = "/app/bin"

	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "./app",
	}
	df, err := docksmith.GenerateDockerfile(fw,
		docksmith.WithExternalTools([]config.ExternalTool{
			{
				Name:        "atlas",
				URL:         "https://release.ariga.io/atlas/atlas-linux-${TARGETARCH}-v1.2.0",
				SHA256:      "c12b889e3349f0e5610aec32fe327e5a6911a0e472754a0c381c30c7c0630e88",
				InstallPath: installPath,
				Format:      "binary",
			},
		}),
	)
	if err != nil {
		t.Fatalf("GenerateDockerfile: %v", err)
	}

	wantMkdir := "mkdir -p " + installPath
	if !strings.Contains(df, wantMkdir) {
		t.Errorf("Dockerfile missing %q for binary format\nDockerfile:\n%s", wantMkdir, df)
	}

	// mkdir must appear before the curl line.
	mkdirIdx := strings.Index(df, wantMkdir)
	curlIdx := strings.Index(df, "curl -fsSL -o "+installPath+"/atlas")
	if mkdirIdx == -1 || curlIdx == -1 {
		t.Fatalf("expected both mkdir and curl lines; mkdir at %d, curl at %d\nDockerfile:\n%s", mkdirIdx, curlIdx, df)
	}
	if mkdirIdx > curlIdx {
		t.Errorf("mkdir must appear before curl in the Dockerfile (mkdir at %d, curl at %d)", mkdirIdx, curlIdx)
	}
}

// TestGoWorkspace_CopyStep asserts that WithGoWork causes the builder stage to
// include go.work* in the COPY step alongside go.mod/go.sum*. This ensures
// workspace replace directives are available when go mod download runs (closes #32).
func TestGoWorkspace_CopyStep(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "./app",
	}

	// Without WithGoWork — go.work* must NOT appear.
	dfWithout, err := docksmith.GenerateDockerfile(fw)
	if err != nil {
		t.Fatalf("GenerateDockerfile (without go work): %v", err)
	}
	if strings.Contains(dfWithout, "go.work") {
		t.Errorf("Dockerfile without WithGoWork must not contain go.work\nDockerfile:\n%s", dfWithout)
	}

	// With WithGoWork — go.work* must appear in the builder COPY step.
	dfWith, err := docksmith.GenerateDockerfile(fw, docksmith.WithGoWork())
	if err != nil {
		t.Fatalf("GenerateDockerfile (with go work): %v", err)
	}
	if !strings.Contains(dfWith, "go.work*") {
		t.Errorf("Dockerfile with WithGoWork missing go.work* in COPY step\nDockerfile:\n%s", dfWith)
	}

	// The go.work* copy must appear on the same COPY line as go.mod (dep-only stage).
	for _, line := range strings.Split(dfWith, "\n") {
		if strings.HasPrefix(line, "COPY") && strings.Contains(line, "go.mod") {
			if !strings.Contains(line, "go.work*") {
				t.Errorf("COPY line with go.mod must also include go.work*\ngot: %s", line)
			}
			return
		}
	}
	t.Errorf("no COPY line containing go.mod found\nDockerfile:\n%s", dfWith)
}
