package docksmith

import (
	"strings"
	"testing"
)

func makeNextJSFramework(pm string) *Framework {
	return &Framework{
		Name:           "nextjs",
		BuildCommand:   pmRunBuild(pm),
		StartCommand:   pmRunStart(pm),
		Port:           3000,
		OutputDir:      ".next",
		NodeVersion:    "22",
		PackageManager: pm,
	}
}

func mustPlanNode(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planNode(fw)
	if err != nil {
		t.Fatalf("planNode: %v", err)
	}
	return plan
}

func TestPlanNode_NextJS_ThreeStages(t *testing.T) {
	plan := mustPlanNode(t, makeNextJSFramework("npm"))
	if len(plan.Stages) != 3 {
		t.Fatalf("want 3 stages, got %d", len(plan.Stages))
	}
	if plan.Stages[0].Name != "deps" {
		t.Errorf("stage 0: want %q, got %q", "deps", plan.Stages[0].Name)
	}
	if plan.Stages[1].Name != "build" {
		t.Errorf("stage 1: want %q, got %q", "build", plan.Stages[1].Name)
	}
	if plan.Stages[2].Name != "runtime" {
		t.Errorf("stage 2: want %q, got %q", "runtime", plan.Stages[2].Name)
	}
}

func TestPlanNode_NextJS_BaseImages(t *testing.T) {
	plan := mustPlanNode(t, makeNextJSFramework("npm"))
	nodeImg := ResolveDockerTag("node", "22")
	if plan.Stages[0].From != nodeImg {
		t.Errorf("deps from: want %q, got %q", nodeImg, plan.Stages[0].From)
	}
	if plan.Stages[1].From != "deps" {
		t.Errorf("build from: want %q, got %q", "deps", plan.Stages[1].From)
	}
	if plan.Stages[2].From != nodeImg {
		t.Errorf("runtime from: want %q, got %q", nodeImg, plan.Stages[2].From)
	}
}

func TestPlanNode_CacheMount_ByPM(t *testing.T) {
	cases := []struct {
		pm    string
		cache string
	}{
		{"npm", "/root/.npm"},
		{"pnpm", "/root/.local/share/pnpm/store"},
		{"yarn", "/usr/local/share/.cache/yarn"},
		{"bun", "/root/.bun/install/cache"},
	}
	for _, tc := range cases {
		plan := mustPlanNode(t, makeNextJSFramework(tc.pm))
		deps := plan.Stages[0]
		var found *CacheMount
		for _, s := range deps.Steps {
			if s.Type == StepRun && s.CacheMount != nil {
				found = s.CacheMount
				break
			}
		}
		if found == nil {
			t.Errorf("pm=%s: no cache mount in deps stage", tc.pm)
			continue
		}
		if found.Target != tc.cache {
			t.Errorf("pm=%s: cache target want %q, got %q", tc.pm, tc.cache, found.Target)
		}
	}
}

func TestPlanNode_StaticFramework_NginxRuntime(t *testing.T) {
	staticFrameworks := []struct {
		name      string
		outputDir string
		port      int
	}{
		{"vite", "dist", 3000},
		{"create-react-app", "build", 3000},
		{"gatsby", "public", 9000},
		{"angular", "dist", 4200},
		{"vue-cli", "dist", 8080},
		{"astro", "dist", 4321},
	}
	for _, tc := range staticFrameworks {
		fw := &Framework{
			Name:           tc.name,
			BuildCommand:   "npm run build",
			StartCommand:   "npx serve dist",
			Port:           tc.port,
			OutputDir:      tc.outputDir,
			NodeVersion:    "22",
			PackageManager: "npm",
		}
		plan := mustPlanNode(t, fw)
		runtime := plan.Stages[len(plan.Stages)-1]
		if runtime.From != "nginx:alpine" {
			t.Errorf("%s: runtime from want nginx:alpine, got %q", tc.name, runtime.From)
		}
	}
}

func TestPlanNode_DynamicFramework_NodeRuntime(t *testing.T) {
	fw := &Framework{
		Name:           "express",
		BuildCommand:   "npm install",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	nodeImg := ResolveDockerTag("node", "22")
	if runtime.From != nodeImg {
		t.Errorf("express runtime from: want %q, got %q", nodeImg, runtime.From)
	}
}

func TestPlanNode_Validate(t *testing.T) {
	pms := []string{"npm", "pnpm", "yarn", "bun"}
	for _, pm := range pms {
		plan := mustPlanNode(t, makeNextJSFramework(pm))
		if err := plan.Validate(); err != nil {
			t.Errorf("pm=%s: Validate() error: %v", pm, err)
		}
	}
}

func TestPlanNode_Expose(t *testing.T) {
	plan := mustPlanNode(t, makeNextJSFramework("npm"))
	if plan.Expose != 3000 {
		t.Errorf("expose: want 3000, got %d", plan.Expose)
	}
}

func TestPlanNode_Framework(t *testing.T) {
	plan := mustPlanNode(t, makeNextJSFramework("npm"))
	if plan.Framework != "nextjs" {
		t.Errorf("framework: want %q, got %q", "nextjs", plan.Framework)
	}
}

// hasCorepackEnable reports whether the deps stage contains a "corepack enable" RUN step.
func hasCorepackEnable(plan *BuildPlan) bool {
	for _, step := range plan.Stages[0].Steps {
		if step.Type == StepRun && len(step.Args) == 1 && step.Args[0] == "corepack enable" {
			return true
		}
	}
	return false
}

func makeNodeFramework(pm, nodeVersion string) *Framework {
	return &Framework{
		Name:           "nextjs",
		BuildCommand:   pmRunBuild(pm),
		StartCommand:   pmRunStart(pm),
		Port:           3000,
		OutputDir:      ".next",
		NodeVersion:    nodeVersion,
		PackageManager: pm,
	}
}

func TestPlanNode_Corepack_PnpmNode22(t *testing.T) {
	plan := mustPlanNode(t, makeNodeFramework("pnpm", "22"))
	if !hasCorepackEnable(plan) {
		t.Error("pnpm + Node 22: expected 'corepack enable' step in deps stage")
	}
}

func TestPlanNode_Corepack_YarnNode22(t *testing.T) {
	plan := mustPlanNode(t, makeNodeFramework("yarn", "22"))
	if !hasCorepackEnable(plan) {
		t.Error("yarn + Node 22: expected 'corepack enable' step in deps stage")
	}
}

func TestPlanNode_Corepack_NpmNode22_NotInjected(t *testing.T) {
	plan := mustPlanNode(t, makeNodeFramework("npm", "22"))
	if hasCorepackEnable(plan) {
		t.Error("npm + Node 22: 'corepack enable' must not be injected for npm")
	}
}

func TestPlanNode_Corepack_PnpmNode20_NotInjected(t *testing.T) {
	plan := mustPlanNode(t, makeNodeFramework("pnpm", "20"))
	if hasCorepackEnable(plan) {
		t.Error("pnpm + Node 20: 'corepack enable' must not be injected for Node < 22")
	}
}

func TestPlanNode_Corepack_PnpmEmptyVersion(t *testing.T) {
	plan := mustPlanNode(t, makeNodeFramework("pnpm", ""))
	if !hasCorepackEnable(plan) {
		t.Error("pnpm + empty version: expected 'corepack enable' (default is latest >= 22)")
	}
}

func TestPlanNode_Corepack_StepOrder(t *testing.T) {
	// corepack enable must appear between WORKDIR and COPY.
	plan := mustPlanNode(t, makeNodeFramework("pnpm", "22"))
	steps := plan.Stages[0].Steps
	workdirIdx := -1
	corepackIdx := -1
	copyIdx := -1
	for i, s := range steps {
		switch {
		case s.Type == StepWorkdir:
			workdirIdx = i
		case s.Type == StepRun && len(s.Args) == 1 && s.Args[0] == "corepack enable":
			corepackIdx = i
		case s.Type == StepCopy && copyIdx == -1:
			copyIdx = i
		}
	}
	if workdirIdx < 0 || corepackIdx < 0 || copyIdx < 0 {
		t.Fatalf("missing steps: workdir=%d corepack=%d copy=%d", workdirIdx, corepackIdx, copyIdx)
	}
	if !(workdirIdx < corepackIdx && corepackIdx < copyIdx) {
		t.Errorf("step order wrong: want WORKDIR < corepack enable < COPY, got indices %d %d %d",
			workdirIdx, corepackIdx, copyIdx)
	}
}

func TestNodeVersionAtLeast(t *testing.T) {
	cases := []struct {
		ver  string
		min  int
		want bool
	}{
		{"22", 22, true},
		{"22.1.0", 22, true},
		{"20", 22, false},
		{"20.11.0", 22, false},
		{"", 22, true},
		{"23", 22, true},
		{"invalid", 22, true},
	}
	for _, tc := range cases {
		got := nodeVersionAtLeast(tc.ver, tc.min)
		if got != tc.want {
			t.Errorf("nodeVersionAtLeast(%q, %d) = %v, want %v", tc.ver, tc.min, got, tc.want)
		}
	}
}

func TestPlanNode_ServerRuntime_HasTini(t *testing.T) {
	fw := &Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	var hasEntrypoint bool
	for _, s := range runtime.Steps {
		if s.Type == StepEntrypoint {
			joined := strings.Join(s.Args, " ")
			if strings.Contains(joined, "tini") {
				hasEntrypoint = true
			}
		}
	}
	if !hasEntrypoint {
		t.Error("server runtime should have tini ENTRYPOINT")
	}
}

func TestPlanNode_ServerRuntime_HasNodeUser(t *testing.T) {
	fw := &Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, s := range runtime.Steps {
		if s.Type == StepUser && s.Args[0] == "node" {
			return
		}
	}
	t.Error("server runtime should have USER node step")
}

func TestPlanNode_ServerRuntime_HasHealthcheck(t *testing.T) {
	fw := &Framework{
		Name:           "nextjs",
		BuildCommand:   "npm run build",
		StartCommand:   "npm start",
		Port:           3000,
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, s := range runtime.Steps {
		if s.Type == StepHealthcheck {
			if strings.Contains(s.Args[0], "3000") {
				return
			}
		}
	}
	t.Error("server runtime should have a healthcheck on port 3000")
}

func TestPlanNode_StaticRuntime_HasNginxUser(t *testing.T) {
	fw := &Framework{
		Name:           "vite",
		BuildCommand:   "npm run build",
		Port:           80,
		OutputDir:      "dist",
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, s := range runtime.Steps {
		if s.Type == StepUser && s.Args[0] == "nginx" {
			return
		}
	}
	t.Error("static runtime should have USER nginx step")
}

func TestPlanNode_StaticRuntime_CopyUsesAbsolutePath(t *testing.T) {
	fw := &Framework{
		Name:           "vite",
		BuildCommand:   "npm run build",
		Port:           80,
		OutputDir:      "dist",
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, s := range runtime.Steps {
		if s.Type == StepCopyFrom && s.CopyFrom != nil && s.CopyFrom.Stage == "build" {
			if s.CopyFrom.Src != "/app/dist" {
				t.Errorf("static COPY src: got %q, want /app/dist (absolute path)", s.CopyFrom.Src)
			}
			return
		}
	}
	t.Error("no COPY --from=build step found in static runtime")
}

func TestPlanNode_StaticRuntime_HasHealthcheck(t *testing.T) {
	fw := &Framework{
		Name:           "vite",
		BuildCommand:   "npm run build",
		Port:           80,
		OutputDir:      "dist",
		NodeVersion:    "22",
		PackageManager: "npm",
	}
	plan := mustPlanNode(t, fw)
	runtime := plan.Stages[len(plan.Stages)-1]
	for _, s := range runtime.Steps {
		if s.Type == StepHealthcheck {
			if strings.Contains(s.Args[0], ":80/") {
				return
			}
		}
	}
	t.Error("static runtime should have a healthcheck on port 80")
}
