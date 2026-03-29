package plan

import (
	"github.com/permanu/docksmith/core"
	"strings"
	"testing"
)

func djangoFramework() *core.Framework {
	return &core.Framework{
		Name:          "django",
		PythonVersion: "3.12",
		PythonPM:      "pip",
		Port:          8000,
		StartCommand:  "gunicorn myapp.wsgi:application",
		SystemDeps:    []string{"libpq-dev"},
	}
}

func mustPlanPython(t *testing.T, fw *core.Framework) *core.BuildPlan {
	t.Helper()
	plan, err := planPython(fw)
	if err != nil {
		t.Fatalf("planPython: %v", err)
	}
	return plan
}

func TestPlanPython_TwoStages(t *testing.T) {
	plan := mustPlanPython(t, djangoFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
	if plan.Stages[0].Name != "builder" {
		t.Errorf("stage 0 name: got %q, want %q", plan.Stages[0].Name, "builder")
	}
	if plan.Stages[1].Name != "runtime" {
		t.Errorf("stage 1 name: got %q, want %q", plan.Stages[1].Name, "runtime")
	}
}

func TestPlanPython_BaseImages(t *testing.T) {
	plan := mustPlanPython(t, djangoFramework())
	if plan.Stages[0].From != "python:3.12-slim" {
		t.Errorf("builder from: got %q, want %q", plan.Stages[0].From, "python:3.12-slim")
	}
	if plan.Stages[1].From != "python:3.12-slim" {
		t.Errorf("runtime from: got %q, want %q", plan.Stages[1].From, "python:3.12-slim")
	}
}

func TestPlanPython_DefaultVersion(t *testing.T) {
	fw := &core.Framework{Name: "flask", Port: 5000, StartCommand: "gunicorn app:app"}
	plan := mustPlanPython(t, fw)
	if plan.Stages[0].From != "python:3.12-slim" {
		t.Errorf("expected default python version, got %q", plan.Stages[0].From)
	}
}

func TestPlanPython_ValidatesOK(t *testing.T) {
	if err := mustPlanPython(t, djangoFramework()).Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanPython_ExposedPort(t *testing.T) {
	plan := mustPlanPython(t, djangoFramework())
	if plan.Expose != 8000 {
		t.Errorf("expose: got %d, want 8000", plan.Expose)
	}
}

func TestPlanPython_BuilderHasSystemDeps(t *testing.T) {
	fw := &core.Framework{
		Name:         "django",
		Port:         8000,
		StartCommand: "gunicorn app.wsgi",
		SystemDeps:   []string{"libpq-dev", "libffi-dev"},
	}
	plan := mustPlanPython(t, fw)
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "libpq-dev") && strings.Contains(arg, "libffi-dev") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("builder stage should install system deps")
	}
}

func TestPlanPython_RuntimeHasNoDevDeps(t *testing.T) {
	fw := &core.Framework{
		Name:         "django",
		Port:         8000,
		StartCommand: "gunicorn app.wsgi",
		SystemDeps:   []string{"libpq-dev"},
	}
	runtime := mustPlanPython(t, fw).Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "libpq-dev") {
					t.Error("runtime stage must not install -dev packages")
				}
			}
		}
	}
}

func TestPlanPython_VenvCopied(t *testing.T) {
	runtime := mustPlanPython(t, djangoFramework()).Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepCopyFrom && step.CopyFrom != nil && step.CopyFrom.Src == "/app/.venv" {
			return
		}
	}
	t.Error("runtime stage must copy /app/.venv from builder")
}

func TestPlanPython_PathEnvSet(t *testing.T) {
	runtime := mustPlanPython(t, djangoFramework()).Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepEnv && len(step.Args) >= 2 && step.Args[0] == "PATH" &&
			strings.Contains(step.Args[1], "/app/.venv/bin") {
			return
		}
	}
	t.Error("runtime stage must set PATH to include /app/.venv/bin")
}

func TestPlanPython_GunicornBindFixup(t *testing.T) {
	fw := &core.Framework{Name: "django", Port: 8000, StartCommand: "gunicorn myapp.wsgi:application"}
	runtime := mustPlanPython(t, fw).Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepCmd {
			cmd := strings.Join(step.Args, " ")
			if !strings.Contains(cmd, "--bind") && !strings.Contains(cmd, "-b ") {
				t.Errorf("gunicorn start command should have --bind, got: %q", cmd)
			}
			return
		}
	}
	t.Error("no CMD step found in runtime stage")
}

func TestPlanPython_GunicornBindNotDuplicated(t *testing.T) {
	fw := &core.Framework{Name: "django", Port: 8000, StartCommand: "gunicorn myapp.wsgi:application --bind 0.0.0.0:8000"}
	runtime := mustPlanPython(t, fw).Stages[1]
	for _, step := range runtime.Steps {
		if step.Type == core.StepCmd {
			cmd := strings.Join(step.Args, " ")
			if count := strings.Count(cmd, "--bind"); count != 1 {
				t.Errorf("--bind appeared %d times, expected exactly 1", count)
			}
			return
		}
	}
	t.Error("no CMD step found in runtime stage")
}

func TestPlanPython_BuilderHasBuildEssential(t *testing.T) {
	builder := mustPlanPython(t, djangoFramework()).Stages[0]
	for _, step := range builder.Steps {
		if step.Type == core.StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "build-essential") {
					return
				}
			}
		}
	}
	t.Error("builder stage should install build-essential")
}

func TestPlanPython_PackageManagers(t *testing.T) {
	cases := []struct {
		pm           string
		wantInstall  string
		wantCache    string
		wantCopyFile string
	}{
		{
			pm:           "pip",
			wantInstall:  "pip install",
			wantCache:    "/root/.cache/pip",
			wantCopyFile: "requirements.txt",
		},
		{
			pm:           "poetry",
			wantInstall:  "poetry install",
			wantCache:    "/root/.cache/pypoetry",
			wantCopyFile: "pyproject.toml",
		},
		{
			pm:           "uv",
			wantInstall:  "uv sync",
			wantCache:    "/root/.cache/uv",
			wantCopyFile: "pyproject.toml",
		},
		{
			pm:           "pdm",
			wantInstall:  "pdm install",
			wantCache:    "/root/.cache/pip",
			wantCopyFile: "pyproject.toml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.pm, func(t *testing.T) {
			fw := &core.Framework{
				Name:         "django",
				PythonPM:     tc.pm,
				Port:         8000,
				StartCommand: "gunicorn app.wsgi",
			}
			builder := mustPlanPython(t, fw).Stages[0]

			var foundInstall, foundCache, foundCopy bool
			for _, step := range builder.Steps {
				switch step.Type {
				case core.StepRun:
					for _, arg := range step.Args {
						if strings.Contains(arg, tc.wantInstall) {
							foundInstall = true
						}
					}
					if step.CacheMount != nil && step.CacheMount.Target == tc.wantCache {
						foundCache = true
					}
				case core.StepCopy:
					for _, arg := range step.Args {
						if strings.Contains(arg, tc.wantCopyFile) {
							foundCopy = true
						}
					}
				}
			}
			if !foundInstall {
				t.Errorf("%s: expected install command containing %q", tc.pm, tc.wantInstall)
			}
			if !foundCache {
				t.Errorf("%s: expected cache mount at %q", tc.pm, tc.wantCache)
			}
			if !foundCopy {
				t.Errorf("%s: expected COPY step with %q", tc.pm, tc.wantCopyFile)
			}
		})
	}
}

func TestPlanPython_Runtime_HasTini(t *testing.T) {
	plan := mustPlanPython(t, djangoFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == core.StepEntrypoint {
			if strings.Contains(strings.Join(s.Args, " "), "tini") {
				return
			}
		}
	}
	t.Error("python runtime should have tini ENTRYPOINT")
}

func TestPlanPython_Runtime_HasAppUser(t *testing.T) {
	plan := mustPlanPython(t, djangoFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == core.StepUser && s.Args[0] == "appuser" {
			return
		}
	}
	t.Error("python runtime should have USER appuser step")
}

func TestPlanPython_Runtime_HasHealthcheck(t *testing.T) {
	plan := mustPlanPython(t, djangoFramework())
	runtime := plan.Stages[1]
	for _, s := range runtime.Steps {
		if s.Type == core.StepHealthcheck {
			if strings.Contains(s.Args[0], "8000") {
				return
			}
		}
	}
	t.Error("python runtime should have a healthcheck on port 8000")
}
