package integration_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/plan"
)

func TestSecretMount_nodeNpmrc_emitsSecretInDockerfile(t *testing.T) {
	fw := &docksmith.Framework{
		Name:           "express",
		PackageManager: "npm",
		NodeVersion:    "22",
		Port:           3000,
		StartCommand:   "npm start",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	plan.ApplySecretMounts(p, "../../testdata/fixtures/node-npmrc-auth")
	df := docksmith.EmitDockerfile(p)

	assertContains(t, df, "--mount=type=secret,id=npm,target=/root/.npmrc")
}

func TestSecretMount_nodeNoNpmrc_noSecretMount(t *testing.T) {
	fw := &docksmith.Framework{
		Name:           "express",
		PackageManager: "npm",
		NodeVersion:    "22",
		Port:           3000,
		StartCommand:   "npm start",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	secrets := plan.ApplySecretMounts(p, "../../testdata/fixtures/node-no-npmrc")
	if len(secrets) != 0 {
		t.Error("expected no secrets for public project")
	}
	df := docksmith.EmitDockerfile(p)
	if strings.Contains(df, "--mount=type=secret") {
		t.Error("Dockerfile should not contain secret mounts for public project")
	}
}

func TestSecretMount_pythonPipConf_emitsSecretInDockerfile(t *testing.T) {
	fw := &docksmith.Framework{
		Name:          "flask",
		PythonVersion: "3.12",
		PythonPM:      "pip",
		Port:          5000,
		StartCommand:  "gunicorn app:app",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	plan.ApplySecretMounts(p, "../../testdata/fixtures/python-pip-conf")
	df := docksmith.EmitDockerfile(p)

	assertContains(t, df, "--mount=type=secret,id=pip,target=/root/.pip/pip.conf")
}

func TestSecretMount_goNetrc_emitsSecretInDockerfile(t *testing.T) {
	fw := &docksmith.Framework{
		Name:      "go",
		GoVersion: "1.22",
		Port:      8080,
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	plan.ApplySecretMounts(p, "../../testdata/fixtures/go-netrc")
	df := docksmith.EmitDockerfile(p)

	assertContains(t, df, "--mount=type=secret,id=netrc,target=/root/.netrc")
}

func TestSecretMount_javaSettings_emitsSecretInDockerfile(t *testing.T) {
	fw := &docksmith.Framework{
		Name:        "maven",
		JavaVersion: "21",
		Port:        8080,
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	plan.ApplySecretMounts(p, "../../testdata/fixtures/java-settings-xml")
	df := docksmith.EmitDockerfile(p)

	assertContains(t, df, "--mount=type=secret,id=maven,target=/root/.m2/settings.xml")
}

func TestSecretMount_rubyPrivateSource_emitsSecretInDockerfile(t *testing.T) {
	fw := &docksmith.Framework{
		Name:         "rails",
		Port:         3000,
		StartCommand: "rails server -b 0.0.0.0",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	plan.ApplySecretMounts(p, "../../testdata/fixtures/ruby-private-source")
	df := docksmith.EmitDockerfile(p)

	assertContains(t, df, "--mount=type=secret,id=bundle,target=/root/.bundle/config")
}

func TestSecretMount_noCredentialValuesInDockerfile(t *testing.T) {
	fw := &docksmith.Framework{
		Name:           "express",
		PackageManager: "npm",
		NodeVersion:    "22",
		Port:           3000,
		StartCommand:   "npm start",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatal(err)
	}
	plan.ApplySecretMounts(p, "../../testdata/fixtures/node-npmrc-auth")
	df := docksmith.EmitDockerfile(p)

	banned := []string{"_authToken", "NPM_TOKEN", "COMPANY_TOKEN", "ghp_PLACEHOLDER"}
	for _, s := range banned {
		if strings.Contains(df, s) {
			t.Errorf("Dockerfile leaks credential string %q", s)
		}
	}
}

func TestDockerignore_excludesCredentialFiles(t *testing.T) {
	runtimes := []string{"nextjs", "django", "go", "rails", "spring-boot", "static"}
	credFiles := []string{".npmrc", ".netrc", "pip.conf", "settings.xml", "*.pem", "*.key"}

	for _, name := range runtimes {
		got := docksmith.GenerateDockerignore(&docksmith.Framework{Name: name})
		for _, f := range credFiles {
			if !strings.Contains(got, f) {
				t.Errorf("framework %q: .dockerignore missing credential pattern %q", name, f)
			}
		}
	}
}

func TestSecretBuildHint_appearsInOutput(t *testing.T) {
	secrets := []docksmith.SecretDef{
		{ID: "npm", Target: "/root/.npmrc", Src: ".npmrc"},
	}
	hint := plan.SecretBuildHint(secrets)
	if !strings.Contains(hint, "docker build --secret id=npm,src=.npmrc") {
		t.Errorf("hint missing build command, got:\n%s", hint)
	}
}

func TestSecretBuildHint_multipleSecrets(t *testing.T) {
	secrets := []detect.SecretDef{
		{ID: "npm", Target: "/root/.npmrc", Src: ".npmrc"},
		{ID: "netrc", Target: "/root/.netrc", Src: ".netrc"},
	}
	hint := plan.SecretBuildHint(secrets)
	if !strings.Contains(hint, "--secret id=npm") || !strings.Contains(hint, "--secret id=netrc") {
		t.Errorf("hint missing one or more secrets, got:\n%s", hint)
	}
}
