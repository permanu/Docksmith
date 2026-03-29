package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectNodeSecrets_withNpmrcAuth(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "node-npmrc-auth")
	secrets := DetectNodeSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "npm" {
		t.Errorf("expected id=npm, got %q", secrets[0].ID)
	}
	if secrets[0].Target != "/root/.npmrc" {
		t.Errorf("expected target=/root/.npmrc, got %q", secrets[0].Target)
	}
}

func TestDetectNodeSecrets_noNpmrc(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "node-no-npmrc")
	secrets := DetectNodeSecrets(dir)
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets for public project, got %d", len(secrets))
	}
}

func TestDetectNodeSecrets_publishConfig(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{
		"name": "@corp/lib",
		"publishConfig": { "registry": "https://npm.corp.com/" }
	}`)
	secrets := DetectNodeSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret for publishConfig, got %d", len(secrets))
	}
}

func TestDetectPythonSecrets_withPipConf(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "python-pip-conf")
	secrets := DetectPythonSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "pip" {
		t.Errorf("expected id=pip, got %q", secrets[0].ID)
	}
}

func TestDetectPythonSecrets_poetrySource(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "pyproject.toml", `[tool.poetry]
name = "myapp"

[tool.poetry.source]
name = "private"
url = "https://pypi.corp.com/simple/"
`)
	secrets := DetectPythonSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret for poetry source, got %d", len(secrets))
	}
}

func TestDetectPythonSecrets_noPipConf(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "requirements.txt", "flask==3.0.0\n")
	secrets := DetectPythonSecrets(dir)
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestDetectGoSecrets_withNetrc(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-netrc")
	secrets := DetectGoSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "netrc" {
		t.Errorf("expected id=netrc, got %q", secrets[0].ID)
	}
}

func TestDetectGoSecrets_goprivateInGoEnv(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "go.env", "GOPRIVATE=github.com/mycompany/*\n")
	secrets := DetectGoSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret for GOPRIVATE, got %d", len(secrets))
	}
}

func TestDetectGoSecrets_noNetrc(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "go.mod", "module example.com/app\ngo 1.22\n")
	secrets := DetectGoSecrets(dir)
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestDetectJavaSecrets_withSettingsXml(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "java-settings-xml")
	secrets := DetectJavaSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "maven" {
		t.Errorf("expected id=maven, got %q", secrets[0].ID)
	}
}

func TestDetectJavaSecrets_mvnSubdir(t *testing.T) {
	dir := t.TempDir()
	mvnDir := filepath.Join(dir, ".mvn")
	if err := os.Mkdir(mvnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, mvnDir, "settings.xml", "<settings><servers><server><id>x</id></server></servers></settings>")
	secrets := DetectJavaSecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret from .mvn/settings.xml, got %d", len(secrets))
	}
}

func TestDetectJavaSecrets_noSettings(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "pom.xml", "<project></project>")
	secrets := DetectJavaSecrets(dir)
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestDetectRubySecrets_privateSource(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "ruby-private-source")
	secrets := DetectRubySecrets(dir)
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].ID != "bundle" {
		t.Errorf("expected id=bundle, got %q", secrets[0].ID)
	}
}

func TestDetectRubySecrets_publicOnly(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "Gemfile", `source "https://rubygems.org"
gem "rails"
`)
	secrets := DetectRubySecrets(dir)
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets for public-only Gemfile, got %d", len(secrets))
	}
}

func writeFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
