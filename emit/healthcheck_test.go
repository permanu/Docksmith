package emit_test

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/emit"
)

// minimalPlan builds a single-stage BuildPlan with a StepHealthcheck.
func minimalPlan(step core.Step) *core.BuildPlan {
	return &core.BuildPlan{
		Framework: "static",
		Expose:    80,
		Stages: []core.Stage{
			{
				Name: "runtime",
				From: "nginx:alpine",
				Steps: []core.Step{
					{Type: core.StepExpose, Args: []string{"80"}},
					step,
					{Type: core.StepCmd, Args: []string{"nginx", "-g", "daemon off;"}},
				},
			},
		},
	}
}

// TestHealthcheckDefaultParams asserts that a StepHealthcheck with nil opts
// produces the exact hardcoded line that existed before configurable params
// were introduced.
func TestHealthcheckDefaultParams(t *testing.T) {
	step := core.Step{
		Type: core.StepHealthcheck,
		Args: []string{"curl -f http://localhost:80/"},
	}
	df := emit.EmitDockerfile(minimalPlan(step))
	want := "HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD curl -f http://localhost:80/"
	if !strings.Contains(df, want) {
		t.Errorf("expected default HEALTHCHECK line:\n  %q\nin Dockerfile:\n%s", want, df)
	}
	if strings.Contains(df, "--retries=") {
		t.Errorf("unexpected --retries flag in default output:\n%s", df)
	}
}

// TestHealthcheckConfiguredParams asserts that opts produce the configured
// stanza including --retries when Retries > 0.
func TestHealthcheckConfiguredParams(t *testing.T) {
	step := core.Step{
		Type: core.StepHealthcheck,
		Args: []string{"curl -f http://localhost:80/"},
		HealthcheckOpts: &core.HealthcheckOpts{
			Interval:    "30s",
			Timeout:     "3s",
			StartPeriod: "10s",
			Retries:     3,
		},
	}
	df := emit.EmitDockerfile(minimalPlan(step))
	want := "HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD curl -f http://localhost:80/"
	if !strings.Contains(df, want) {
		t.Errorf("expected configured HEALTHCHECK line:\n  %q\nin Dockerfile:\n%s", want, df)
	}
}

// TestHealthcheckPartialOpts ensures only explicitly set fields override
// the defaults — unset fields fall back to the hardcoded values.
func TestHealthcheckPartialOpts(t *testing.T) {
	step := core.Step{
		Type: core.StepHealthcheck,
		Args: []string{"curl -f http://localhost:80/"},
		HealthcheckOpts: &core.HealthcheckOpts{
			Timeout: "3s",
		},
	}
	df := emit.EmitDockerfile(minimalPlan(step))
	want := "HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD"
	if !strings.Contains(df, want) {
		t.Errorf("expected partial HEALTHCHECK line:\n  %q\nin Dockerfile:\n%s", want, df)
	}
	if strings.Contains(df, "--retries=") {
		t.Errorf("unexpected --retries flag when Retries=0:\n%s", df)
	}
}
