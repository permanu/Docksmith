package integration_test

import (
	"testing"

	"github.com/permanu/docksmith"
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
	var def docksmith.FrameworkDef
	if err := yaml.Unmarshal([]byte(sampleFrameworkYAML), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if def.Name != "nextjs" {
		t.Errorf("Name: got %q, want %q", def.Name, "nextjs")
	}
	if def.Runtime != "node" {
		t.Errorf("Runtime: got %q, want %q", def.Runtime, "node")
	}
	if def.Priority != 10 {
		t.Errorf("Priority: got %d, want 10", def.Priority)
	}
	if len(def.Detect.All) != 2 {
		t.Fatalf("Detect.All len: got %d, want 2", len(def.Detect.All))
	}
	if def.Detect.All[0].File != "package.json" {
		t.Errorf("Detect.All[0].File: got %q", def.Detect.All[0].File)
	}
	if def.Detect.All[1].Dependency != "next" {
		t.Errorf("Detect.All[1].Dependency: got %q", def.Detect.All[1].Dependency)
	}
	if len(def.Detect.None) != 1 || def.Detect.None[0].File != "Dockerfile" {
		t.Errorf("Detect.None unexpected")
	}
	if len(def.Version.Sources) != 2 {
		t.Fatalf("Version.Sources len: got %d, want 2", len(def.Version.Sources))
	}
	if def.Version.Default != "22" {
		t.Errorf("Version.Default: got %q, want %q", def.Version.Default, "22")
	}
	if len(def.PackageManager.Sources) != 3 {
		t.Fatalf("PackageManager.Sources len: got %d, want 3", len(def.PackageManager.Sources))
	}
	if def.PackageManager.Default != "npm" {
		t.Errorf("PackageManager.Default: got %q, want %q", def.PackageManager.Default, "npm")
	}
	if def.Plan.Port != 3000 {
		t.Errorf("Plan.Port: got %d, want 3000", def.Plan.Port)
	}
	if len(def.Plan.Stages) != 3 {
		t.Fatalf("Plan.Stages len: got %d, want 3", len(def.Plan.Stages))
	}
	if def.Defaults.Install["npm"] != "npm ci" {
		t.Errorf("Defaults.Install[npm]: got %q", def.Defaults.Install["npm"])
	}
	if def.Defaults.Build != "npm run build" {
		t.Errorf("Defaults.Build: got %q", def.Defaults.Build)
	}
	if len(def.Tests) != 2 {
		t.Fatalf("Tests len: got %d, want 2", len(def.Tests))
	}
	if !def.Tests[0].Expect.Detected {
		t.Error("Tests[0].Expect.Detected: got false, want true")
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
	var def docksmith.FrameworkDef
	if err := yaml.Unmarshal([]byte(minimal), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(def.Detect.Any) != 0 {
		t.Errorf("Detect.Any should be empty")
	}
	if len(def.Detect.None) != 0 {
		t.Errorf("Detect.None should be empty")
	}
	if def.Plan.Port != 8080 {
		t.Errorf("Plan.Port: got %d, want 8080", def.Plan.Port)
	}
}
