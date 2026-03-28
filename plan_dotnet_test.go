package docksmith

import (
	"strings"
	"testing"
)

func aspnetFramework() *Framework {
	return &Framework{
		Name:          "aspnet-core",
		BuildCommand:  "dotnet publish -c Release -o /app/publish",
		StartCommand:  "dotnet /app/publish/MyWebApp.dll",
		Port:          8080,
		DotnetVersion: "8.0",
	}
}

func dotnetWorkerFramework() *Framework {
	return &Framework{
		Name:          "dotnet-worker",
		BuildCommand:  "dotnet publish -c Release -o /app/publish",
		StartCommand:  "dotnet /app/publish/MyWorker.dll",
		Port:          0,
		DotnetVersion: "8.0",
	}
}

func mustPlanDotnet(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planDotnet(fw)
	if err != nil {
		t.Fatalf("planDotnet: %v", err)
	}
	return plan
}

// --- ASP.NET Core ---

func TestPlanDotnet_AspNet_TwoStages(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanDotnet_AspNet_BuilderUsesSdk(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	want := ResolveDockerTag("dotnet-sdk", "8.0")
	if plan.Stages[0].From != want {
		t.Errorf("builder from: got %q, want %q", plan.Stages[0].From, want)
	}
}

func TestPlanDotnet_AspNet_RuntimeUsesAspNet(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	want := ResolveDockerTag("dotnet-aspnet", "8.0")
	if plan.Stages[1].From != want {
		t.Errorf("runtime from: got %q, want %q", plan.Stages[1].From, want)
	}
}

func TestPlanDotnet_AspNet_BuilderRestores(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "dotnet restore") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should run dotnet restore")
	}
}

func TestPlanDotnet_AspNet_BuilderHasCacheMount(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil {
			found = true
		}
	}
	if !found {
		t.Error("builder should have a NuGet cache mount")
	}
}

func TestPlanDotnet_AspNet_RuntimeCopiesPublish(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCopyFrom && step.CopyFrom != nil && step.CopyFrom.Stage == "builder" {
			found = true
		}
	}
	if !found {
		t.Error("runtime should copy from builder stage")
	}
}

func TestPlanDotnet_AspNet_RuntimeHasEntrypoint(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepEntrypoint {
			for _, arg := range step.Args {
				if strings.HasSuffix(arg, ".dll") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("runtime should have ENTRYPOINT with .dll")
	}
}

func TestPlanDotnet_AspNet_ProjectNameExtracted(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	runtime := plan.Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == StepEntrypoint {
			for _, arg := range step.Args {
				if strings.HasSuffix(arg, ".dll") && !strings.Contains(arg, "MyWebApp") {
					t.Errorf("entrypoint dll should be MyWebApp.dll, got %q", arg)
				}
			}
		}
	}
}

func TestPlanDotnet_AspNet_ValidatesOK(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanDotnet_AspNet_ExposedPort(t *testing.T) {
	plan := mustPlanDotnet(t, aspnetFramework())
	if plan.Expose != 8080 {
		t.Errorf("expose: got %d, want 8080", plan.Expose)
	}
}

// --- Worker ---

func TestPlanDotnet_Worker_TwoStages(t *testing.T) {
	plan := mustPlanDotnet(t, dotnetWorkerFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanDotnet_Worker_RuntimeUsesRuntime(t *testing.T) {
	plan := mustPlanDotnet(t, dotnetWorkerFramework())
	want := ResolveDockerTag("dotnet-runtime", "8.0")
	if plan.Stages[1].From != want {
		t.Errorf("worker runtime from: got %q, want %q", plan.Stages[1].From, want)
	}
}

func TestPlanDotnet_Worker_ValidatesOK(t *testing.T) {
	plan := mustPlanDotnet(t, dotnetWorkerFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("worker should pass validate: %v", err)
	}
}

func TestPlanDotnet_ExtractProjectName(t *testing.T) {
	cases := []struct {
		cmd  string
		want string
	}{
		{"dotnet /app/publish/MyApp.dll", "MyApp"},
		{"dotnet /app/publish/WebApi.dll", "WebApi"},
		{"", "app"},
		{"dotnet run", "app"},
	}
	for _, c := range cases {
		got := extractDotnetProjectName(c.cmd)
		if got != c.want {
			t.Errorf("extractDotnetProjectName(%q): got %q, want %q", c.cmd, got, c.want)
		}
	}
}

func TestPlanDotnet_DefaultDotnetVersion(t *testing.T) {
	fw := &Framework{Name: "aspnet-core", Port: 8080, StartCommand: "dotnet /app/publish/App.dll"}
	plan := mustPlanDotnet(t, fw)
	if !strings.Contains(plan.Stages[0].From, "8.0") {
		t.Errorf("expected dotnet 8.0 in builder image, got %q", plan.Stages[0].From)
	}
}
