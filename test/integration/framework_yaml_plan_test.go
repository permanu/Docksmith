package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/yamldef"
)

func minimalNodeDef() *docksmith.FrameworkDef {
	return &docksmith.FrameworkDef{
		Name:    "express",
		Runtime: "node",
		Version: docksmith.VersionConfig{Default: "22"},
		PackageManager: docksmith.PMConfig{Default: "npm"},
		Plan: docksmith.PlanDef{
			Port: 3000,
			Stages: []docksmith.StageDef{
				{
					Name: "deps",
					Base: "node",
					Steps: []docksmith.StepDef{
						{Workdir: "/app"},
						{Copy: []string{"package.json", "package-lock.json*", "./"}},
						{Run: "{{install_command}}", Cache: "/root/.npm"},
					},
				},
				{
					Name: "build",
					From: "deps",
					Steps: []docksmith.StepDef{
						{Copy: []string{".", "."}},
						{Run: "{{build_command}}"},
					},
				},
				{
					Name: "runtime",
					Base: "node",
					Steps: []docksmith.StepDef{
						{Workdir: "/app"},
						{CopyFrom: &docksmith.CopyFromDef{Stage: "build", Src: "/app", Dst: "/app"}},
						{Cmd: []string{"node", "server.js"}},
					},
				},
			},
		},
		Defaults: docksmith.DefaultsDef{
			Install: map[string]string{
				"npm":  "npm ci",
				"pnpm": "pnpm install --frozen-lockfile",
			},
			Build: "npm run build",
			Start: "npm start",
		},
	}
}

func TestBuildPlanFromDefBasic(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	if plan.Framework != "express" {
		t.Errorf("Framework: got %q, want %q", plan.Framework, "express")
	}
	if plan.Expose != 3000 {
		t.Errorf("Expose: got %d, want 3000", plan.Expose)
	}
	if len(plan.Stages) != 3 {
		t.Fatalf("Stages len: got %d, want 3", len(plan.Stages))
	}
}

func TestBuildPlanFromDefStageNames(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	names := []string{"deps", "build", "runtime"}
	for i, want := range names {
		if plan.Stages[i].Name != want {
			t.Errorf("Stages[%d].Name: got %q, want %q", i, plan.Stages[i].Name, want)
		}
	}
}

func TestBuildPlanFromDefBaseResolvesToDockerTag(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	if plan.Stages[0].From != "node:22-alpine" {
		t.Errorf("deps From: got %q, want %q", plan.Stages[0].From, "node:22-alpine")
	}
	if plan.Stages[2].From != "node:22-alpine" {
		t.Errorf("runtime From: got %q, want %q", plan.Stages[2].From, "node:22-alpine")
	}
}

func TestBuildPlanFromDefFromLiteral(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	if plan.Stages[1].From != "deps" {
		t.Errorf("build From: got %q, want %q", plan.Stages[1].From, "deps")
	}
}

func TestBuildPlanFromDefInstallCommandVariable(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[0].Steps[2]
	if step.Type != docksmith.StepRun {
		t.Fatalf("step type: got %d, want StepRun", step.Type)
	}
	if len(step.Args) == 0 || step.Args[0] != "npm ci" {
		t.Errorf("install_command: got %v, want [npm ci]", step.Args)
	}
}

func TestBuildPlanFromDefBuildCommandVariable(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[1].Steps[1]
	if step.Args[0] != "npm run build" {
		t.Errorf("build_command: got %q, want %q", step.Args[0], "npm run build")
	}
}

func TestBuildPlanFromDefCacheMount(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[0].Steps[2]
	if step.CacheMount == nil {
		t.Fatal("expected cache mount on deps install step, got nil")
	}
	if step.CacheMount.Target != "/root/.npm" {
		t.Errorf("cache mount target: got %q, want %q", step.CacheMount.Target, "/root/.npm")
	}
}

func TestBuildPlanFromDefCopyFrom(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[2].Steps[1]
	if step.Type != docksmith.StepCopyFrom {
		t.Fatalf("step type: got %d, want StepCopyFrom", step.Type)
	}
	if step.CopyFrom == nil {
		t.Fatal("expected CopyFrom to be non-nil")
	}
	if step.CopyFrom.Stage != "build" {
		t.Errorf("CopyFrom.Stage: got %q, want %q", step.CopyFrom.Stage, "build")
	}
}

func TestBuildPlanFromDefWorkdir(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[0].Steps[0]
	if step.Type != docksmith.StepWorkdir {
		t.Fatalf("step type: got %d, want StepWorkdir", step.Type)
	}
	if step.Args[0] != "/app" {
		t.Errorf("workdir: got %q, want /app", step.Args[0])
	}
}

func TestBuildPlanFromDefCmd(t *testing.T) {
	dir := t.TempDir()
	plan, err := docksmith.BuildPlanFromDefDir(minimalNodeDef(), dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[2].Steps[2]
	if step.Type != docksmith.StepCmd {
		t.Fatalf("step type: got %d, want StepCmd", step.Type)
	}
	if len(step.Args) != 2 || step.Args[0] != "node" || step.Args[1] != "server.js" {
		t.Errorf("cmd: got %v, want [node server.js]", step.Args)
	}
}

// --- Version detection ---

func TestBuildPlanFromDefVersionFromFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("20\n"), 0o644)
	def := minimalNodeDef()
	def.Version.Sources = []docksmith.VersionSource{{File: ".nvmrc"}}
	def.Version.Default = "22"

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	if plan.Stages[0].From != "node:20-alpine" {
		t.Errorf("deps From: got %q, want node:20-alpine", plan.Stages[0].From)
	}
}

func TestBuildPlanFromDefVersionFromJSONPath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"engines":{"node":"18"}}`), 0o644)
	def := minimalNodeDef()
	def.Version.Sources = []docksmith.VersionSource{
		{JSON: "package.json", Path: "engines.node"},
	}

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	if plan.Stages[0].From != "node:18-alpine" {
		t.Errorf("deps From: got %q, want node:18-alpine", plan.Stages[0].From)
	}
}

func TestBuildPlanFromDefVersionFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()
	def.Version.Sources = []docksmith.VersionSource{{File: ".nvmrc"}}
	def.Version.Default = "20"

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	if plan.Stages[0].From != "node:20-alpine" {
		t.Errorf("deps From: got %q, want node:20-alpine", plan.Stages[0].From)
	}
}

// --- PM detection ---

func TestBuildPlanFromDefPMFromLockfile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644)
	def := minimalNodeDef()
	def.PackageManager.Sources = []docksmith.PMSource{
		{File: "pnpm-lock.yaml", Value: "pnpm"},
		{File: "yarn.lock", Value: "yarn"},
	}
	def.PackageManager.Default = "npm"

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[0].Steps[2]
	if step.Args[0] != "pnpm install --frozen-lockfile" {
		t.Errorf("pnpm install_command: got %q", step.Args[0])
	}
}

func TestBuildPlanFromDefPMFromJSONPath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"packageManager":"yarn@3.6.0"}`), 0o644)
	def := minimalNodeDef()
	def.PackageManager.Sources = []docksmith.PMSource{
		{JSON: "package.json", Path: "packageManager"},
	}
	def.PackageManager.Default = "npm"
	def.Defaults.Install["yarn"] = "yarn install --frozen-lockfile"

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[0].Steps[2]
	if step.Args[0] != "yarn install --frozen-lockfile" {
		t.Errorf("yarn install_command: got %q", step.Args[0])
	}
}

// --- Variable substitution ---

func TestBuildPlanFromDefUnknownVariableLeftInPlace(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()
	def.Plan.Stages[1].Steps[1] = docksmith.StepDef{Run: "{{unknown_var}}"}

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[1].Steps[1]
	if step.Args[0] != "{{unknown_var}}" {
		t.Errorf("unknown var: got %q, want {{unknown_var}}", step.Args[0])
	}
}

func TestBuildPlanFromDefPortVariable(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()
	def.Plan.Stages[2].Steps = []docksmith.StepDef{{Expose: "{{port}}"}}

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[2].Steps[0]
	if step.Type != docksmith.StepExpose {
		t.Fatalf("step type: got %d, want StepExpose", step.Type)
	}
	if step.Args[0] != "3000" {
		t.Errorf("expose port: got %q, want 3000", step.Args[0])
	}
}

func TestBuildPlanFromDefLockfileVariable(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()
	def.Plan.Stages[0].Steps = []docksmith.StepDef{
		{Copy: []string{"package.json", "{{lockfile}}", "./"}},
	}

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[0].Steps[0]
	if step.Args[1] != "package-lock.json" {
		t.Errorf("lockfile for npm: got %q, want package-lock.json", step.Args[1])
	}
}

func TestBuildPlanFromDefEnvStep(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()
	def.Plan.Stages[1].Steps = []docksmith.StepDef{
		{Env: map[string]string{"NODE_ENV": "production"}},
	}

	plan, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err != nil {
		t.Fatalf("BuildPlanFromDefDir: %v", err)
	}
	step := plan.Stages[1].Steps[0]
	if step.Type != docksmith.StepEnv {
		t.Fatalf("step type: got %d, want StepEnv", step.Type)
	}
	if len(step.Args) != 2 || step.Args[0] != "NODE_ENV" || step.Args[1] != "production" {
		t.Errorf("env args: got %v", step.Args)
	}
}

func TestBuildPlanFromDefNilReturnsError(t *testing.T) {
	_, err := docksmith.BuildPlanFromDefDir(nil, t.TempDir())
	if err == nil {
		t.Error("expected error for nil def")
	}
}

func TestBuildPlanFromDefStageMissingFromAndBase(t *testing.T) {
	dir := t.TempDir()
	def := minimalNodeDef()
	def.Plan.Stages[0].Base = ""
	def.Plan.Stages[0].From = ""

	_, err := docksmith.BuildPlanFromDefDir(def, dir)
	if err == nil {
		t.Error("expected error for stage with neither base nor from")
	}
}

// --- PMLockfileName ---

func TestPMLockfileName(t *testing.T) {
	tests := []struct{ pm, want string }{
		{"npm", "package-lock.json"}, {"pnpm", "pnpm-lock.yaml"},
		{"yarn", "yarn.lock"}, {"bun", "bun.lockb"},
		{"pip", "requirements.txt"}, {"poetry", "poetry.lock"},
		{"cargo", "Cargo.lock"}, {"bundler", "Gemfile.lock"},
		{"composer", "composer.lock"}, {"unknown", "package-lock.json"},
	}
	for _, tt := range tests {
		got := yamldef.PMLockfileName(tt.pm)
		if got != tt.want {
			t.Errorf("PMLockfileName(%q): got %q, want %q", tt.pm, got, tt.want)
		}
	}
}
