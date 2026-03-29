package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
)

func TestApplySecretMounts_nodeWithNpmrc(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "node-npmrc-auth")
	fw := &core.Framework{
		Name:           "express",
		PackageManager: "npm",
		NodeVersion:    "22",
		Port:           3000,
		StartCommand:   "npm start",
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := ApplySecretMounts(p, dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}

	// Verify the install step got a secret mount.
	found := false
	for _, stage := range p.Stages {
		for _, step := range stage.Steps {
			if step.SecretMount != nil && step.SecretMount.ID == "npm" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected secret mount on install step")
	}
}

func TestApplySecretMounts_nodeNoNpmrc(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "node-no-npmrc")
	fw := &core.Framework{
		Name:           "express",
		PackageManager: "npm",
		NodeVersion:    "22",
		Port:           3000,
		StartCommand:   "npm start",
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := ApplySecretMounts(p, dir)
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets for public project, got %d", len(secrets))
	}
}

func TestApplySecretMounts_pythonWithPipConf(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "python-pip-conf")
	fw := &core.Framework{
		Name:          "flask",
		PythonVersion: "3.12",
		PythonPM:      "pip",
		Port:          5000,
		StartCommand:  "gunicorn app:app",
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := ApplySecretMounts(p, dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "pip" {
		t.Errorf("expected pip secret, got %q", secrets[0].ID)
	}
}

func TestApplySecretMounts_goWithNetrc(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "go-netrc")
	fw := &core.Framework{
		Name:      "go",
		GoVersion: "1.22",
		Port:      8080,
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := ApplySecretMounts(p, dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "netrc" {
		t.Errorf("expected netrc secret, got %q", secrets[0].ID)
	}
}

func TestApplySecretMounts_javaWithSettings(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "java-settings-xml")
	fw := &core.Framework{
		Name:        "maven",
		JavaVersion: "21",
		Port:        8080,
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := ApplySecretMounts(p, dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "maven" {
		t.Errorf("expected maven secret, got %q", secrets[0].ID)
	}
}

func TestApplySecretMounts_rubyPrivateSource(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "ruby-private-source")
	fw := &core.Framework{
		Name:         "rails",
		Port:         3000,
		StartCommand: "rails server -b 0.0.0.0",
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := ApplySecretMounts(p, dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "bundle" {
		t.Errorf("expected bundle secret, got %q", secrets[0].ID)
	}
}

func TestApplySecretMounts_nilPlan(t *testing.T) {
	secrets := ApplySecretMounts(nil, "/nonexistent")
	if len(secrets) != 0 {
		t.Errorf("expected nil for nil plan, got %d", len(secrets))
	}
}

func TestValidateSecretTarget_pathTraversal(t *testing.T) {
	tests := []struct {
		target  string
		wantErr bool
	}{
		{"/root/.npmrc", false},
		{"/root/.pip/pip.conf", false},
		{"../../../etc/passwd", true},
		{"/root/../etc/passwd", true},
		{"relative/path", true},
		{"", true},
	}
	for _, tt := range tests {
		err := validateSecretTarget(tt.target)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateSecretTarget(%q) error=%v, wantErr=%v", tt.target, err, tt.wantErr)
		}
	}
}

func TestSecretBuildHint_withSecrets(t *testing.T) {
	secrets := []detect.SecretDef{
		{ID: "npm", Target: "/root/.npmrc", Src: ".npmrc"},
	}
	hint := SecretBuildHint(secrets)
	if !strings.Contains(hint, "--secret id=npm,src=.npmrc") {
		t.Errorf("expected build hint to contain --secret flag, got:\n%s", hint)
	}
	if !strings.HasPrefix(hint, "#") {
		t.Error("hint should be a comment")
	}
}

func TestSecretBuildHint_empty(t *testing.T) {
	hint := SecretBuildHint(nil)
	if hint != "" {
		t.Errorf("expected empty hint for no secrets, got %q", hint)
	}
}

func TestNoSecretValuesInDockerfile(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "node-npmrc-auth")
	fw := &core.Framework{
		Name:           "express",
		PackageManager: "npm",
		NodeVersion:    "22",
		Port:           3000,
		StartCommand:   "npm start",
	}
	p, err := Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	ApplySecretMounts(p, dir)

	// Read the actual .npmrc to get credential patterns.
	npmrc, err := os.ReadFile(filepath.Join(dir, ".npmrc"))
	if err != nil {
		t.Fatal(err)
	}

	df := emitForTest(p)
	for _, line := range strings.Split(string(npmrc), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(df, line) {
			t.Errorf("Dockerfile contains credential line: %q", line)
		}
	}

	// Verify no ENV/ARG/COPY with credential files.
	for _, banned := range []string{"_authToken", "NPM_TOKEN", "COMPANY_TOKEN"} {
		if strings.Contains(df, banned) {
			t.Errorf("Dockerfile contains banned string %q", banned)
		}
	}
}

func emitForTest(p *core.BuildPlan) string {
	var b strings.Builder
	for _, stage := range p.Stages {
		for _, step := range stage.Steps {
			b.WriteString(strings.Join(step.Args, " "))
			b.WriteByte('\n')
		}
	}
	return b.String()
}
