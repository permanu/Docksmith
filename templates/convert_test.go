// SPDX-License-Identifier: MPL-2.0

package templates_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/permanu/docksmith/blueprint"
	"github.com/permanu/docksmith/templates"
)

func TestConvertRailwayTemplate(t *testing.T) {
	spec := convertFixture(t, templates.FormatRailway, "railway/template.json")

	if spec.Name != "fixture-shop" {
		t.Fatalf("Name = %q", spec.Name)
	}
	if spec.SchemaVersion != "blueprint.docksmith.dev/v1" {
		t.Fatalf("SchemaVersion = %q", spec.SchemaVersion)
	}
	if spec.Environment != blueprint.EnvironmentDevelopment {
		t.Fatalf("Environment = %q", spec.Environment)
	}

	web := componentByName(t, spec, "web")
	if web.Kind != blueprint.ComponentApp {
		t.Fatalf("web kind = %q", web.Kind)
	}
	if web.Process.WorkingDir != "apps/web" {
		t.Fatalf("web working dir = %q, want the named monorepo service apps/web", web.Process.WorkingDir)
	}
	if strings.Join(web.Process.Command, " ") != "npm start" {
		t.Fatalf("web command = %#v", web.Process.Command)
	}
	if web.Health.Path != "/healthz" {
		t.Fatalf("web health = %q", web.Health.Path)
	}
	if web.Scaling.Replicas != 2 {
		t.Fatalf("web replicas = %d", web.Scaling.Replicas)
	}
	if len(web.Process.Ports) != 1 || web.Process.Ports[0].Container != 3000 {
		t.Fatalf("web ports = %#v", web.Process.Ports)
	}
	assertLiteralEnv(t, web, "NODE_ENV", "production")
	assertSecretEnv(t, web, "DATABASE_URL")
	if !dependsOn(web, "Postgres", blueprint.DependencyConditionHealthy) {
		t.Fatalf("web depends_on = %#v", web.Process.DependsOn)
	}

	pg := componentByName(t, spec, "Postgres")
	if pg.Kind != blueprint.ComponentStateful || pg.Image != "postgres:16" {
		t.Fatalf("postgres = kind %q image %q", pg.Kind, pg.Image)
	}
	if pg.Stateful == nil || pg.Stateful.DataPath != "/var/lib/postgresql/data" || pg.Stateful.VolumeName != "vol-pg" {
		t.Fatalf("postgres stateful = %#v", pg.Stateful)
	}
	assertNoLeak(t, spec, "${{Postgres.DATABASE_URL}}")
}

func TestConvertCompose(t *testing.T) {
	spec := convertFixture(t, templates.FormatCompose, "compose/compose.yaml")

	if spec.Name != "fixture-shop" {
		t.Fatalf("Name = %q", spec.Name)
	}
	if spec.Environment != blueprint.EnvironmentDevelopment {
		t.Fatalf("Environment = %q", spec.Environment)
	}

	web := componentByName(t, spec, "web")
	if web.Kind != blueprint.ComponentApp || web.Image != "ghcr.io/example/web:1" {
		t.Fatalf("web = kind %q image %q", web.Kind, web.Image)
	}
	if web.Process.WorkingDir != "/app" {
		t.Fatalf("web working dir = %q", web.Process.WorkingDir)
	}
	if strings.Join(web.Process.Command, " ") != "npm start" {
		t.Fatalf("web command = %#v", web.Process.Command)
	}
	if web.Scaling.Replicas != 2 {
		t.Fatalf("web replicas = %d", web.Scaling.Replicas)
	}
	if len(web.Process.Ports) != 1 || web.Process.Ports[0].Published != 8080 || web.Process.Ports[0].Container != 3000 {
		t.Fatalf("web ports = %#v", web.Process.Ports)
	}
	if !dependsOn(web, "db", blueprint.DependencyConditionHealthy) {
		t.Fatalf("web depends_on = %#v", web.Process.DependsOn)
	}
	if strings.Join(web.Health.Command, " ") != "CMD wget -qO- http://127.0.0.1:3000/healthz" {
		t.Fatalf("web health command = %#v", web.Health.Command)
	}
	if web.Health.Interval != "10s" || web.Health.Timeout != "3s" || web.Health.Retries != 3 {
		t.Fatalf("web health = %#v", web.Health)
	}
	assertLiteralEnv(t, web, "NODE_ENV", "production")
	assertSecretEnv(t, web, "API_TOKEN")

	db := componentByName(t, spec, "db")
	if db.Kind != blueprint.ComponentStateful || db.Image != "postgres:16" {
		t.Fatalf("db = kind %q image %q", db.Kind, db.Image)
	}
	if db.Stateful == nil || db.Stateful.DataPath != "/var/lib/postgresql/data" || db.Stateful.VolumeName != "pgdata" {
		t.Fatalf("db stateful = %#v", db.Stateful)
	}
	assertSecretEnv(t, db, "POSTGRES_PASSWORD")
	assertNoLeak(t, spec, "literal-secret-value")
}

func TestConvertHerokuAppJSON(t *testing.T) {
	spec := convertFixture(t, templates.FormatHeroku, "heroku/app.json")

	if spec.Name != "fixture-shop" {
		t.Fatalf("Name = %q", spec.Name)
	}
	if spec.Environment != blueprint.EnvironmentDevelopment {
		t.Fatalf("Environment = %q", spec.Environment)
	}

	web := componentByName(t, spec, "web")
	if web.Kind != blueprint.ComponentApp || web.Process.Type != blueprint.ProcessTypeWeb || web.Scaling.Replicas != 2 {
		t.Fatalf("web = kind %q type %q replicas %d", web.Kind, web.Process.Type, web.Scaling.Replicas)
	}
	assertLiteralEnv(t, web, "NODE_ENV", "production")
	assertSecretEnv(t, web, "DATABASE_URL")
	assertSecretEnv(t, web, "SECRET_TOKEN")
	if !dependsOn(web, "database", blueprint.DependencyConditionHealthy) {
		t.Fatalf("web depends_on = %#v", web.Process.DependsOn)
	}

	worker := componentByName(t, spec, "worker")
	if worker.Kind != blueprint.ComponentApp || worker.Process.Type != blueprint.ProcessTypeWorker || worker.Scaling.Replicas != 1 {
		t.Fatalf("worker = kind %q type %q replicas %d", worker.Kind, worker.Process.Type, worker.Scaling.Replicas)
	}

	db := componentByName(t, spec, "database")
	if db.Kind != blueprint.ComponentStateful || db.Image != "postgres:16" {
		t.Fatalf("database = kind %q image %q", db.Kind, db.Image)
	}
	if db.Stateful == nil || db.Stateful.DataPath != "/var/lib/postgresql/data" {
		t.Fatalf("database stateful = %#v", db.Stateful)
	}

	if len(spec.Release.Commands) != 1 {
		t.Fatalf("release commands = %#v", spec.Release.Commands)
	}
	release := spec.Release.Commands[0]
	if release.Name != "postdeploy" || strings.Join(release.Command, " ") != "npm run migrate" {
		t.Fatalf("release = %#v", release)
	}
}

func TestConvertRejectsHelmBeyondExperimental(t *testing.T) {
	data := readFixture(t, "helm/Chart.yaml")
	spec, err := templates.Convert(templates.FormatHelm, data)
	if err == nil {
		t.Fatal("Convert(helm) error = nil, want unsupported experimental rejection")
	}
	if !errors.Is(err, templates.ErrUnsupported) {
		t.Fatalf("errors.Is(ErrUnsupported) = false, err = %v", err)
	}
	if !errors.Is(err, templates.ErrExperimental) {
		t.Fatalf("errors.Is(ErrExperimental) = false, err = %v", err)
	}
	if !strings.Contains(err.Error(), "experimental") || !strings.Contains(err.Error(), "not a supported import") {
		t.Fatalf("error = %q, want experimental and not a supported import", err)
	}
	if spec.Name != "" || len(spec.Components) != 0 {
		t.Fatalf("helm import produced a blueprint: %#v", spec)
	}
}

func convertFixture(t *testing.T, kind, rel string) blueprint.DeploymentSpec {
	t.Helper()
	spec, err := templates.Convert(kind, readFixture(t, rel))
	if err != nil {
		t.Fatalf("Convert(%s) error = %v", kind, err)
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	return spec
}

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "testdata", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func componentByName(t *testing.T, spec blueprint.DeploymentSpec, name string) blueprint.Component {
	t.Helper()
	for i := range spec.Components {
		if spec.Components[i].Name == name {
			return spec.Components[i]
		}
	}
	t.Fatalf("component %q not found in %#v", name, spec.Components)
	return blueprint.Component{}
}

func envByName(t *testing.T, component blueprint.Component, name string) blueprint.Env {
	t.Helper()
	for i := range component.Env {
		if component.Env[i].Name == name {
			return component.Env[i]
		}
	}
	t.Fatalf("env %q not found", name)
	return blueprint.Env{}
}

func assertLiteralEnv(t *testing.T, component blueprint.Component, name, value string) {
	t.Helper()
	env := envByName(t, component, name)
	if env.Value != value || env.ValueFrom != "" {
		t.Fatalf("env %s = %#v, want literal %q", name, env, value)
	}
}

func assertSecretEnv(t *testing.T, component blueprint.Component, name string) {
	t.Helper()
	env := envByName(t, component, name)
	if env.Value != "" || !strings.HasPrefix(env.ValueFrom, "secret:") {
		t.Fatalf("env %s = %#v, want value_from secret", name, env)
	}
}

func dependsOn(component blueprint.Component, name string, condition blueprint.DependencyCondition) bool {
	for i := range component.Process.DependsOn {
		dep := component.Process.DependsOn[i]
		if dep.Component == name && dep.Condition == condition {
			return true
		}
	}
	return false
}

func assertNoLeak(t *testing.T, spec blueprint.DeploymentSpec, sentinel string) {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	if strings.Contains(string(raw), sentinel) {
		t.Fatalf("blueprint contains %q", sentinel)
	}
}
