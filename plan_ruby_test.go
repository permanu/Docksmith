package docksmith

import (
	"strings"
	"testing"
)

func railsFramework() *Framework {
	return &Framework{
		Name:         "rails",
		BuildCommand: "bundle install",
		StartCommand: "rails server -b 0.0.0.0 -p 3000",
		Port:         3000,
	}
}

func sinatraFramework() *Framework {
	return &Framework{
		Name:         "sinatra",
		BuildCommand: "bundle install",
		StartCommand: "ruby app.rb -o 0.0.0.0 -p 4567",
		Port:         4567,
	}
}

func mustPlanRuby(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planRuby(fw)
	if err != nil {
		t.Fatalf("planRuby: %v", err)
	}
	return plan
}

func TestPlanRuby_TwoStages(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanRuby_StageNames(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	if plan.Stages[0].Name != "builder" {
		t.Errorf("stage 0: got %q, want %q", plan.Stages[0].Name, "builder")
	}
	if plan.Stages[1].Name != "runtime" {
		t.Errorf("stage 1: got %q, want %q", plan.Stages[1].Name, "runtime")
	}
}

func TestPlanRuby_BaseImageFromResolveDockerTag(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	want := ResolveDockerTag("ruby", "")
	if plan.Stages[0].From != want {
		t.Errorf("builder from: got %q, want %q", plan.Stages[0].From, want)
	}
	if plan.Stages[1].From != want {
		t.Errorf("runtime from: got %q, want %q", plan.Stages[1].From, want)
	}
}

func TestPlanRuby_BuilderInstallsGems(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "bundle install") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should run bundle install")
	}
}

func TestPlanRuby_BuilderHasCacheMount(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil {
			found = true
		}
	}
	if !found {
		t.Error("builder should have a cache mount for gems")
	}
}

func TestPlanRuby_BuilderCopiesGemfile(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepCopy {
			for _, arg := range step.Args {
				if strings.Contains(arg, "Gemfile") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder should copy Gemfile")
	}
}

func TestPlanRuby_RailsRunsAssetPrecompile(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "assets:precompile") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("rails builder should run assets:precompile")
	}
}

func TestPlanRuby_SinatraNoAssetPrecompile(t *testing.T) {
	plan := mustPlanRuby(t, sinatraFramework())
	builder := plan.Stages[0]
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "assets:precompile") {
					t.Error("sinatra builder should not run assets:precompile")
				}
			}
		}
	}
}

func TestPlanRuby_RuntimeCopiesGemsFromBuilder(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
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

func TestPlanRuby_RuntimeHasCmd(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCmd {
			found = true
		}
	}
	if !found {
		t.Error("runtime stage must have a CMD step")
	}
}

func TestPlanRuby_ExposedPort(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	if plan.Expose != 3000 {
		t.Errorf("expose: got %d, want 3000", plan.Expose)
	}
}

func TestPlanRuby_ValidatesOK(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanRuby_DefaultPort(t *testing.T) {
	fw := &Framework{Name: "rails", BuildCommand: "bundle install", StartCommand: "rails server"}
	plan := mustPlanRuby(t, fw)
	if plan.Expose != 3000 {
		t.Errorf("default port: got %d, want 3000", plan.Expose)
	}
}

func TestPlanRuby_FrameworkName(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	if plan.Framework != "rails" {
		t.Errorf("framework: got %q, want %q", plan.Framework, "rails")
	}
}

func TestPlanRuby_Runtime_HasBundleBinInPath(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == StepEnv && s.Args[0] == "PATH" {
			if !strings.Contains(s.Args[1], "/usr/local/bundle/bin") {
				t.Errorf("PATH should include /usr/local/bundle/bin, got: %s", s.Args[1])
			}
			return
		}
	}
	t.Error("ruby runtime should have ENV PATH with bundle bin")
}

func TestPlanRuby_Runtime_HasAppUser(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == StepUser && s.Args[0] == "appuser" {
			return
		}
	}
	t.Error("ruby runtime should have USER appuser step")
}

func TestPlanRuby_Runtime_HasHealthcheck(t *testing.T) {
	plan := mustPlanRuby(t, railsFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == StepHealthcheck {
			return
		}
	}
	t.Error("ruby runtime should have a HEALTHCHECK step")
}
