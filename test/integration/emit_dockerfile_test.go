package integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/permanu/docksmith"
)

// helpers -----------------------------------------------------------------

func mustNodePlan(t *testing.T) *docksmith.BuildPlan {
	t.Helper()
	fw := &docksmith.Framework{
		Name:           "nextjs",
		NodeVersion:    "22",
		PackageManager: "npm",
		Port:           3000,
		StartCommand:   "node server.js",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatalf("Plan(nextjs): %v", err)
	}
	return p
}

func mustPythonPlan(t *testing.T) *docksmith.BuildPlan {
	t.Helper()
	fw := &docksmith.Framework{
		Name:          "django",
		PythonVersion: "3.12",
		PythonPM:      "pip",
		Port:          8000,
		StartCommand:  "gunicorn myapp.wsgi:application",
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatalf("Plan(django): %v", err)
	}
	return p
}

func mustGoPlan(t *testing.T) *docksmith.BuildPlan {
	t.Helper()
	fw := &docksmith.Framework{
		Name:      "go",
		GoVersion: "1.26",
		Port:      8080,
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatalf("Plan(go): %v", err)
	}
	return p
}

func mustStaticPlan(t *testing.T) *docksmith.BuildPlan {
	t.Helper()
	fw := &docksmith.Framework{
		Name:      "static",
		OutputDir: "public",
		Port:      0,
	}
	p, err := docksmith.Plan(fw)
	if err != nil {
		t.Fatalf("Plan(static): %v", err)
	}
	return p
}

// Node.js plan ------------------------------------------------------------

func TestEmitDockerfile_NodeJS_MultiStage(t *testing.T) {
	plan := mustNodePlan(t)
	out := docksmith.EmitDockerfile(plan)

	assertContains(t, out, "FROM node:22-alpine AS deps")
	assertContains(t, out, "FROM deps AS build")
	assertContains(t, out, "FROM node:22-alpine AS runtime")

	fromCount := strings.Count(out, "FROM ")
	if fromCount < 3 {
		t.Errorf("expected >=3 FROM lines, got %d\nDockerfile:\n%s", fromCount, out)
	}
}

func TestEmitDockerfile_NodeJS_CopyFrom(t *testing.T) {
	plan := mustNodePlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "COPY --from=build --link /app /app")
}

func TestEmitDockerfile_NodeJS_CMD(t *testing.T) {
	plan := mustNodePlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, `CMD ["node", "server.js"]`)
}

func TestEmitDockerfile_NodeJS_CacheMount(t *testing.T) {
	plan := mustNodePlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "--mount=type=cache,target=/root/.npm")
	assertContains(t, out, "npm ci")
}

// Python plan -------------------------------------------------------------

func TestEmitDockerfile_Python_VenvCopy(t *testing.T) {
	plan := mustPythonPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "COPY --from=builder /app/.venv /app/.venv")
}

func TestEmitDockerfile_Python_EnvPath(t *testing.T) {
	plan := mustPythonPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "ENV PATH /app/.venv/bin:$PATH")
}

func TestEmitDockerfile_Python_CMD(t *testing.T) {
	plan := mustPythonPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "CMD gunicorn")
}

// Go plan -----------------------------------------------------------------

func TestEmitDockerfile_Go_CGODisabled(t *testing.T) {
	plan := mustGoPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "CGO_ENABLED=0")
}

func TestEmitDockerfile_Go_DistrolessRuntime(t *testing.T) {
	plan := mustGoPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "FROM gcr.io/distroless/static-debian12:nonroot AS runtime")
}

func TestEmitDockerfile_Go_BinaryCopy(t *testing.T) {
	plan := mustGoPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "COPY --from=builder /app/app ./app")
}

// Static plan -------------------------------------------------------------

func TestEmitDockerfile_Static_NginxBase(t *testing.T) {
	plan := mustStaticPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "FROM nginx:alpine AS runtime")
}

func TestEmitDockerfile_Static_CopyToHtmlRoot(t *testing.T) {
	plan := mustStaticPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "COPY public /usr/share/nginx/html")
}

// Step type coverage ------------------------------------------------------

func TestEmitDockerfile_StepWorkdir(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepWorkdir, Args: []string{"/app"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "WORKDIR /app")
}

func TestEmitDockerfile_StepCopy(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepCopy, Args: []string{"src", "dst"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "COPY src dst")
}

func TestEmitDockerfile_StepCopyWithLink(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepCopy, Args: []string{"src", "dst"}, Link: true})
	assertContains(t, docksmith.EmitDockerfile(plan), "COPY --link src dst")
}

func TestEmitDockerfile_StepCopyFrom(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{
		Type:     docksmith.StepCopyFrom,
		CopyFrom: &docksmith.CopyFrom{Stage: "build", Src: "/app/bin", Dst: "/usr/local/bin"},
	})
	assertContains(t, docksmith.EmitDockerfile(plan), "COPY --from=build /app/bin /usr/local/bin")
}

func TestEmitDockerfile_StepCopyFromWithLink(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{
		Type:     docksmith.StepCopyFrom,
		CopyFrom: &docksmith.CopyFrom{Stage: "build", Src: "/app", Dst: "/app"},
		Link:     true,
	})
	assertContains(t, docksmith.EmitDockerfile(plan), "COPY --from=build --link /app /app")
}

func TestEmitDockerfile_StepRun(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepRun, Args: []string{"echo hello"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "RUN echo hello")
}

func TestEmitDockerfile_StepRun_CacheMount(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{
		Type:       docksmith.StepRun,
		Args:       []string{"npm ci"},
		CacheMount: &docksmith.CacheMount{Target: "/root/.npm"},
	})
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "RUN --mount=type=cache,target=/root/.npm npm ci")
}

func TestEmitDockerfile_StepRun_SecretMount(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{
		Type:        docksmith.StepRun,
		Args:        []string{"pip install -r requirements.txt"},
		SecretMount: &docksmith.SecretMount{ID: "pip-conf", Target: "/root/.pip/pip.conf"},
	})
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "--mount=type=secret,id=pip-conf,target=/root/.pip/pip.conf")
}

func TestEmitDockerfile_StepEnv(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepEnv, Args: []string{"NODE_ENV", "production"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "ENV NODE_ENV production")
}

func TestEmitDockerfile_StepArg(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepArg, Args: []string{"BUILD_VERSION"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "ARG BUILD_VERSION")
}

func TestEmitDockerfile_StepExpose(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepExpose, Args: []string{"8080"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "EXPOSE 8080")
}

func TestEmitDockerfile_StepCmd(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepCmd, Args: []string{"node", "index.js"}})
	assertContains(t, docksmith.EmitDockerfile(plan), `CMD ["node", "index.js"]`)
}

func TestEmitDockerfile_StepEntrypoint(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepEntrypoint, Args: []string{"/docker-entrypoint.sh"}})
	assertContains(t, docksmith.EmitDockerfile(plan), `ENTRYPOINT ["/docker-entrypoint.sh"]`)
}

func TestEmitDockerfile_StepUser(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepUser, Args: []string{"nobody"}})
	assertContains(t, docksmith.EmitDockerfile(plan), "USER nobody")
}

func TestEmitDockerfile_StepHealthcheck(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{Type: docksmith.StepHealthcheck, Args: []string{"curl -f http://localhost/health || exit 1"}})
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD curl -f http://localhost/health || exit 1")
}

// EXPOSE from plan.Expose -------------------------------------------------

func TestEmitDockerfile_ExposeFromPlan(t *testing.T) {
	plan := mustGoPlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "EXPOSE 8080")
}

// Edge cases --------------------------------------------------------------

func TestEmitDockerfile_EmptyPlan_ReturnsEmpty(t *testing.T) {
	plan := &docksmith.BuildPlan{Framework: "go", Expose: 8080}
	out := docksmith.EmitDockerfile(plan)
	if out != "" {
		t.Errorf("expected empty string for empty plan, got:\n%s", out)
	}
}

func TestEmitDockerfile_InjectionSafety_NewlineInArg(t *testing.T) {
	plan := singleStepPlan(docksmith.Step{
		Type: docksmith.StepRun,
		Args: []string{"echo hello\nRUN rm -rf /"},
	})
	out := docksmith.EmitDockerfile(plan)

	lines := strings.Split(out, "\n")
	runCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "RUN ") {
			runCount++
		}
	}
	if runCount > 1 {
		t.Errorf("injection not sanitized — found %d RUN lines:\n%s", runCount, out)
	}
}

func TestEmitDockerfile_BlankLineBetweenStages(t *testing.T) {
	plan := mustNodePlan(t)
	out := docksmith.EmitDockerfile(plan)

	lines := strings.Split(out, "\n")
	fromCount := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "FROM ") {
			fromCount++
			if fromCount > 1 && i > 0 && lines[i-1] != "" {
				t.Errorf("expected blank line before FROM at line %d; previous line: %q", i+1, lines[i-1])
			}
		}
	}
	if fromCount < 2 {
		t.Errorf("expected at least 2 FROM lines to test blank-line separation, got %d", fromCount)
	}
}

func TestEmitDockerfile_SyntaxComment(t *testing.T) {
	plan := mustNodePlan(t)
	out := docksmith.EmitDockerfile(plan)
	assertContains(t, out, "# syntax=docker/dockerfile:1")
}

// helpers -----------------------------------------------------------------

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected Dockerfile to contain:\n  %q\ngot:\n%s", needle, haystack)
	}
}

func singleStepPlan(step docksmith.Step) *docksmith.BuildPlan {
	return &docksmith.BuildPlan{
		Framework: "go",
		Expose:    8080,
		Stages: []docksmith.Stage{
			{
				Name:  "runtime",
				From:  fmt.Sprintf("golang:%s-alpine", "1.26"),
				Steps: []docksmith.Step{step},
			},
		},
	}
}
