package plan

import (
	"github.com/permanu/docksmith/core"
	"strings"
	"testing"
)

func goFramework() *core.Framework {
	return &core.Framework{
		Name:         "go-gin",
		GoVersion:    "1.22",
		Port:         8080,
		BuildCommand: "go build -o server .",
		StartCommand: "./server",
	}
}

func mustPlanGo(t *testing.T, fw *core.Framework) *core.BuildPlan {
	t.Helper()
	plan, err := planGo(fw)
	if err != nil {
		t.Fatalf("planGo: %v", err)
	}
	return plan
}

func TestPlanGo_TwoStages(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanGo_StageNames(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	if plan.Stages[0].Name != "builder" {
		t.Errorf("stage 0: got %q, want %q", plan.Stages[0].Name, "builder")
	}
	if plan.Stages[1].Name != "runtime" {
		t.Errorf("stage 1: got %q, want %q", plan.Stages[1].Name, "runtime")
	}
}

func TestPlanGo_BuilderUsesGolangAlpine(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	if plan.Stages[0].From != "golang:1.22-alpine" {
		t.Errorf("builder from: got %q, want %q", plan.Stages[0].From, "golang:1.22-alpine")
	}
}

func TestPlanGo_RuntimeUsesDistroless(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	from := plan.Stages[1].From
	if from != "gcr.io/distroless/static-debian12:nonroot" {
		t.Errorf("runtime should use distroless/static nonroot, got %q", from)
	}
}

func TestPlanGo_RuntimeHasNonRootUser(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	runtime := plan.Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepUser && step.Args[0] == "nonroot" {
			return
		}
	}
	t.Error("runtime should have USER nonroot step")
}

func TestPlanGo_RuntimeNoHealthcheck(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	runtime := plan.Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepHealthcheck {
			t.Error("go distroless runtime should not have HEALTHCHECK")
		}
	}
}

func TestPlanGo_BuilderStripsSymbols(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	builder := plan.Stages[0]
	for _, step := range builder.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "go build") && strings.Contains(arg, "-ldflags") {
					return
				}
			}
		}
	}
	t.Error("builder should use -ldflags for symbol stripping")
}

func TestPlanGo_DefaultVersion(t *testing.T) {
	fw := &core.Framework{
		Name:         "go-std",
		Port:         8080,
		StartCommand: "./server",
	}
	plan := mustPlanGo(t, fw)
	if plan.Stages[0].From != "golang:1.26-alpine" {
		t.Errorf("expected default go version, got %q", plan.Stages[0].From)
	}
}

func TestPlanGo_ValidatesOK(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanGo_ExposedPort(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	if plan.Expose != 8080 {
		t.Errorf("expose: got %d, want 8080", plan.Expose)
	}
}

func TestPlanGo_BuilderCopiesGoMod(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == core.StepCopy {
			for _, arg := range step.Args {
				if strings.Contains(arg, "go.mod") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should copy go.mod")
	}
}

func TestPlanGo_BuilderDownloadsCacheMount(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == core.StepRun && step.CacheMount != nil {
			if step.CacheMount.Target == "/go/pkg/mod" {
				found = true
			}
		}
	}
	if !found {
		t.Error("go mod download should have cache mount at /go/pkg/mod")
	}
}

func TestPlanGo_BuilderRunsBuild(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "CGO_ENABLED=0") && strings.Contains(arg, "go build") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should run CGO_ENABLED=0 go build")
	}
}

func TestPlanGo_BinaryExtractedFromBuildCommand(t *testing.T) {
	fw := &core.Framework{
		Name:         "go-chi",
		GoVersion:    "1.22",
		Port:         8080,
		BuildCommand: "go build -o myserver ./cmd/api",
		StartCommand: "./myserver",
	}
	plan := mustPlanGo(t, fw)
	builder := plan.Stages[0]
	for _, step := range builder.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "go build") {
					if !strings.Contains(arg, "myserver") {
						t.Errorf("build step should use binary name myserver, got: %q", arg)
					}
					return
				}
			}
		}
	}
	t.Error("no go build step found")
}

func TestPlanGo_RuntimeCopiesBinary(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == core.StepCopyFrom && step.CopyFrom != nil && step.CopyFrom.Stage == "builder" {
			found = true
		}
	}
	if !found {
		t.Error("runtime should copy binary from builder stage")
	}
}

func TestPlanGo_RuntimeHasCmd(t *testing.T) {
	plan := mustPlanGo(t, goFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == core.StepCmd {
			found = true
		}
	}
	if !found {
		t.Error("runtime stage must have a CMD step")
	}
}

func TestPlanGo_FrameworkName(t *testing.T) {
	fw := goFramework()
	plan := mustPlanGo(t, fw)
	if plan.Framework != "go-gin" {
		t.Errorf("framework name: got %q, want %q", plan.Framework, "go-gin")
	}
}

func TestPlanGo_DefaultBinaryName(t *testing.T) {
	// No -o flag — binary defaults to "app"
	fw := &core.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		BuildCommand: "go build .",
		StartCommand: "./app",
	}
	plan := mustPlanGo(t, fw)
	builder := plan.Stages[0]
	for _, step := range builder.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "go build") {
					if !strings.Contains(arg, "/app/app") {
						t.Errorf("default binary should be app, got: %q", arg)
					}
					return
				}
			}
		}
	}
}

func TestPlanGo_NoBuildCommand(t *testing.T) {
	fw := &core.Framework{
		Name:         "go-std",
		GoVersion:    "1.22",
		Port:         8080,
		StartCommand: "./server",
	}
	plan := mustPlanGo(t, fw)
	if err := plan.Validate(); err != nil {
		t.Errorf("plan with no BuildCommand should still validate: %v", err)
	}
}
