package yamldef

import (
	"testing"

	"gopkg.in/yaml.v3"
)

const sampleFrameworkYAML = `
name: nextjs
runtime: node
priority: 10

detect:
  all:
    - file: package.json
    - dependency: next
  none:
    - file: Dockerfile

version:
  sources:
    - file: .nvmrc
    - json: package.json
      path: engines.node
  default: "22"

package_manager:
  sources:
    - json: package.json
      path: packageManager
    - file: pnpm-lock.yaml
      value: pnpm
    - file: yarn.lock
      value: yarn
  default: npm

plan:
  port: 3000
  stages:
    - name: deps
      base: node
      steps:
        - workdir: /app
        - copy: ["package.json", "package-lock.json*", "./"]
        - run: "npm ci"
          cache: /root/.npm
    - name: build
      from: deps
      steps:
        - copy: [".", "."]
        - env:
            NODE_ENV: production
        - run: npm run build
    - name: runtime
      base: node
      steps:
        - workdir: /app
        - copy_from:
            stage: build
            src: /app
            dst: /app
        - expose: "3000"
        - cmd: ["node", "server.js"]

defaults:
  install:
    npm: "npm ci"
    pnpm: "pnpm install --frozen-lockfile"
  build: "npm run build"
  start: "npm start"

tests:
  - name: basic nextjs detection
    fixture:
      package.json: '{"dependencies":{"next":"13.0.0"}}'
    expect:
      detected: true
      framework: nextjs
      port: 3000
  - name: no nextjs dependency
    fixture:
      package.json: '{"dependencies":{"express":"4.0.0"}}'
    expect:
      detected: false
`

func TestFrameworkDefParsesAllFields(t *testing.T) {
	var def FrameworkDef
	if err := yaml.Unmarshal([]byte(sampleFrameworkYAML), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Top-level fields
	if def.Name != "nextjs" {
		t.Errorf("Name: got %q, want %q", def.Name, "nextjs")
	}
	if def.Runtime != "node" {
		t.Errorf("Runtime: got %q, want %q", def.Runtime, "node")
	}
	if def.Priority != 10 {
		t.Errorf("Priority: got %d, want 10", def.Priority)
	}

	// DetectRules
	if len(def.Detect.All) != 2 {
		t.Fatalf("Detect.All len: got %d, want 2", len(def.Detect.All))
	}
	if def.Detect.All[0].File != "package.json" {
		t.Errorf("Detect.All[0].File: got %q, want %q", def.Detect.All[0].File, "package.json")
	}
	if def.Detect.All[1].Dependency != "next" {
		t.Errorf("Detect.All[1].Dependency: got %q, want %q", def.Detect.All[1].Dependency, "next")
	}
	if len(def.Detect.None) != 1 {
		t.Fatalf("Detect.None len: got %d, want 1", len(def.Detect.None))
	}
	if def.Detect.None[0].File != "Dockerfile" {
		t.Errorf("Detect.None[0].File: got %q, want %q", def.Detect.None[0].File, "Dockerfile")
	}
	if len(def.Detect.Any) != 0 {
		t.Errorf("Detect.Any: expected empty, got %d", len(def.Detect.Any))
	}

	// VersionConfig
	if len(def.Version.Sources) != 2 {
		t.Fatalf("Version.Sources len: got %d, want 2", len(def.Version.Sources))
	}
	if def.Version.Sources[0].File != ".nvmrc" {
		t.Errorf("Version.Sources[0].File: got %q, want %q", def.Version.Sources[0].File, ".nvmrc")
	}
	if def.Version.Sources[1].JSON != "package.json" {
		t.Errorf("Version.Sources[1].JSON: got %q, want %q", def.Version.Sources[1].JSON, "package.json")
	}
	if def.Version.Sources[1].Path != "engines.node" {
		t.Errorf("Version.Sources[1].Path: got %q, want %q", def.Version.Sources[1].Path, "engines.node")
	}
	if def.Version.Default != "22" {
		t.Errorf("Version.Default: got %q, want %q", def.Version.Default, "22")
	}

	// PMConfig
	if len(def.PackageManager.Sources) != 3 {
		t.Fatalf("PackageManager.Sources len: got %d, want 3", len(def.PackageManager.Sources))
	}
	if def.PackageManager.Sources[0].JSON != "package.json" {
		t.Errorf("PMSources[0].JSON: got %q, want %q", def.PackageManager.Sources[0].JSON, "package.json")
	}
	if def.PackageManager.Sources[0].Path != "packageManager" {
		t.Errorf("PMSources[0].Path: got %q, want %q", def.PackageManager.Sources[0].Path, "packageManager")
	}
	if def.PackageManager.Sources[1].File != "pnpm-lock.yaml" {
		t.Errorf("PMSources[1].File: got %q, want %q", def.PackageManager.Sources[1].File, "pnpm-lock.yaml")
	}
	if def.PackageManager.Sources[1].Value != "pnpm" {
		t.Errorf("PMSources[1].Value: got %q, want %q", def.PackageManager.Sources[1].Value, "pnpm")
	}
	if def.PackageManager.Default != "npm" {
		t.Errorf("PackageManager.Default: got %q, want %q", def.PackageManager.Default, "npm")
	}

	// PlanDef
	if def.Plan.Port != 3000 {
		t.Errorf("Plan.Port: got %d, want 3000", def.Plan.Port)
	}
	if len(def.Plan.Stages) != 3 {
		t.Fatalf("Plan.Stages len: got %d, want 3", len(def.Plan.Stages))
	}

	// Stage: deps
	deps := def.Plan.Stages[0]
	if deps.Name != "deps" {
		t.Errorf("stages[0].Name: got %q, want %q", deps.Name, "deps")
	}
	if deps.Base != "node" {
		t.Errorf("stages[0].Base: got %q, want %q", deps.Base, "node")
	}
	if len(deps.Steps) != 3 {
		t.Fatalf("stages[0].Steps len: got %d, want 3", len(deps.Steps))
	}
	if deps.Steps[0].Workdir != "/app" {
		t.Errorf("deps.Steps[0].Workdir: got %q, want %q", deps.Steps[0].Workdir, "/app")
	}
	if len(deps.Steps[1].Copy) != 3 {
		t.Fatalf("deps.Steps[1].Copy len: got %d, want 3", len(deps.Steps[1].Copy))
	}
	if deps.Steps[2].Run != "npm ci" {
		t.Errorf("deps.Steps[2].Run: got %q, want %q", deps.Steps[2].Run, "npm ci")
	}
	if deps.Steps[2].Cache != "/root/.npm" {
		t.Errorf("deps.Steps[2].Cache: got %q, want %q", deps.Steps[2].Cache, "/root/.npm")
	}

	// Stage: build
	build := def.Plan.Stages[1]
	if build.From != "deps" {
		t.Errorf("stages[1].From: got %q, want %q", build.From, "deps")
	}
	if len(build.Steps[1].Env) != 1 {
		t.Errorf("build.Steps[1].Env len: got %d, want 1", len(build.Steps[1].Env))
	}
	if build.Steps[1].Env["NODE_ENV"] != "production" {
		t.Errorf("build.Steps[1].Env[NODE_ENV]: got %q, want %q", build.Steps[1].Env["NODE_ENV"], "production")
	}

	// Stage: runtime
	runtime := def.Plan.Stages[2]
	if len(runtime.Steps) != 4 {
		t.Fatalf("runtime.Steps len: got %d, want 4", len(runtime.Steps))
	}
	cf := runtime.Steps[1].CopyFrom
	if cf == nil {
		t.Fatal("runtime.Steps[1].CopyFrom: got nil, want *CopyFromDef")
	}
	if cf.Stage != "build" {
		t.Errorf("CopyFrom.Stage: got %q, want %q", cf.Stage, "build")
	}
	if cf.Src != "/app" {
		t.Errorf("CopyFrom.Src: got %q, want %q", cf.Src, "/app")
	}
	if cf.Dst != "/app" {
		t.Errorf("CopyFrom.Dst: got %q, want %q", cf.Dst, "/app")
	}
	if runtime.Steps[2].Expose != "3000" {
		t.Errorf("runtime.Steps[2].Expose: got %q, want %q", runtime.Steps[2].Expose, "3000")
	}
	if len(runtime.Steps[3].Cmd) != 2 {
		t.Fatalf("runtime.Steps[3].Cmd len: got %d, want 2", len(runtime.Steps[3].Cmd))
	}

	// DefaultsDef
	if def.Defaults.Install["npm"] != "npm ci" {
		t.Errorf("Defaults.Install[npm]: got %q, want %q", def.Defaults.Install["npm"], "npm ci")
	}
	if def.Defaults.Install["pnpm"] != "pnpm install --frozen-lockfile" {
		t.Errorf("Defaults.Install[pnpm]: got %q", def.Defaults.Install["pnpm"])
	}
	if def.Defaults.Build != "npm run build" {
		t.Errorf("Defaults.Build: got %q, want %q", def.Defaults.Build, "npm run build")
	}
	if def.Defaults.Start != "npm start" {
		t.Errorf("Defaults.Start: got %q, want %q", def.Defaults.Start, "npm start")
	}

	// TestCases
	if len(def.Tests) != 2 {
		t.Fatalf("Tests len: got %d, want 2", len(def.Tests))
	}
	tc0 := def.Tests[0]
	if tc0.Name != "basic nextjs detection" {
		t.Errorf("Tests[0].Name: got %q", tc0.Name)
	}
	if tc0.Fixture["package.json"] == "" {
		t.Error("Tests[0].Fixture[package.json]: expected non-empty")
	}
	if !tc0.Expect.Detected {
		t.Error("Tests[0].Expect.Detected: got false, want true")
	}
	if tc0.Expect.Framework != "nextjs" {
		t.Errorf("Tests[0].Expect.Framework: got %q, want %q", tc0.Expect.Framework, "nextjs")
	}
	if tc0.Expect.Port != 3000 {
		t.Errorf("Tests[0].Expect.Port: got %d, want 3000", tc0.Expect.Port)
	}
	tc1 := def.Tests[1]
	if tc1.Expect.Detected {
		t.Error("Tests[1].Expect.Detected: got true, want false")
	}
}

func TestFrameworkDefEmptyRulesAreNil(t *testing.T) {
	const minimal = `
name: go-std
runtime: go
detect:
  all:
    - file: go.mod
plan:
  port: 8080
  stages:
    - name: build
      base: go
      steps:
        - run: go build -o app .
`
	var def FrameworkDef
	if err := yaml.Unmarshal([]byte(minimal), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(def.Detect.Any) != 0 {
		t.Errorf("Detect.Any should be empty for minimal YAML")
	}
	if len(def.Detect.None) != 0 {
		t.Errorf("Detect.None should be empty for minimal YAML")
	}
	if def.Plan.Port != 8080 {
		t.Errorf("Plan.Port: got %d, want 8080", def.Plan.Port)
	}
}
