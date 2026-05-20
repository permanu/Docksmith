package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerfileCommandAppliesConfigPlanOptions(t *testing.T) {
	dir := writeCLIConfigFixture(t)

	cmd := exec.Command("go", "run", ".", "dockerfile", dir)
	cmd.Dir = "."
	var out, errw bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errw
	if err := cmd.Run(); err != nil {
		t.Fatalf("dockerfile command failed: %v\nstderr:\n%s", err, errw.String())
	}

	want := "HEALTHCHECK --interval=5s --timeout=2s --start-period=1s --retries=4 CMD wget --spider -q http://localhost:4173/health || exit 1"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("dockerfile command did not apply config healthcheck options.\nwant line:\n%s\nstdout:\n%s", want, out.String())
	}
}

func TestPlanCommandAppliesConfigPlanOptions(t *testing.T) {
	dir := writeCLIConfigFixture(t)

	cmd := exec.Command("go", "run", ".", "-format", "json", "plan", dir)
	cmd.Dir = "."
	var out, errw bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errw
	if err := cmd.Run(); err != nil {
		t.Fatalf("plan command failed: %v\nstderr:\n%s", err, errw.String())
	}

	if !strings.Contains(out.String(), `"healthcheck_opts"`) {
		t.Fatalf("plan command did not include config-derived healthcheck options:\n%s", out.String())
	}
	if !strings.Contains(out.String(), `"retries": 4`) {
		t.Fatalf("plan command did not include config-derived healthcheck retries:\n%s", out.String())
	}
}

func TestBuildTOMLTemplateNilFramework(t *testing.T) {
	got := buildTOMLTemplate(nil)

	for _, want := range []string{
		"# runtime = \"\"",
		"# build = \"\"",
		"# start = \"\"",
		"# port = 8080",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("nil framework template missing %q:\n%s", want, got)
		}
	}
}

func writeCLIConfigFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := `runtime = "go"
version = "1.22"

[build]
command = "go build -o /app/server ."

[start]
command = "./server"

[runtime_config]
expose = 4173
healthcheck = "wget --spider -q http://localhost:4173/health || exit 1"

[runtime_config.healthcheck_opts]
interval = "5s"
timeout = "2s"
start_period = "1s"
retries = 4
`
	if err := os.WriteFile(filepath.Join(dir, "docksmith.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
