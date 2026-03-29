package emit

import (
	"strings"
	"testing"

	"github.com/permanu/docksmith/core"
)

func secretPlan(steps ...core.Step) *core.BuildPlan {
	return &core.BuildPlan{
		Framework: "python",
		Expose:    8000,
		Stages: []core.Stage{
			{
				Name:  "build",
				From:  "python:3.12-slim",
				Steps: steps,
			},
		},
	}
}

func TestEmitSecretMount_TargetForm(t *testing.T) {
	plan := secretPlan(core.Step{
		Type: core.StepRun,
		Args: []string{"pip install -r requirements.txt"},
		SecretMounts: []core.SecretMount{
			{ID: "pip-conf", Target: "/root/.pip/pip.conf"},
		},
	})
	out := EmitDockerfile(plan)
	if !strings.Contains(out, "--mount=type=secret,id=pip-conf,target=/root/.pip/pip.conf") {
		t.Errorf("expected target-form secret mount in:\n%s", out)
	}
}

func TestEmitSecretMount_EnvForm(t *testing.T) {
	plan := secretPlan(core.Step{
		Type: core.StepRun,
		Args: []string{"pip install -r requirements.txt"},
		SecretMounts: []core.SecretMount{
			{ID: "license", Env: "LICENSE_KEY"},
		},
	})
	out := EmitDockerfile(plan)
	if !strings.Contains(out, "--mount=type=secret,id=license,env=LICENSE_KEY") {
		t.Errorf("expected env-form secret mount in:\n%s", out)
	}
}

func TestEmitSecretMount_Multiple(t *testing.T) {
	plan := secretPlan(core.Step{
		Type: core.StepRun,
		Args: []string{"pip install -r requirements.txt"},
		SecretMounts: []core.SecretMount{
			{ID: "pip-conf", Target: "/root/.pip/pip.conf"},
			{ID: "token", Env: "API_TOKEN"},
		},
	})
	out := EmitDockerfile(plan)
	if !strings.Contains(out, "--mount=type=secret,id=pip-conf,target=/root/.pip/pip.conf") {
		t.Errorf("missing pip-conf mount in:\n%s", out)
	}
	if !strings.Contains(out, "--mount=type=secret,id=token,env=API_TOKEN") {
		t.Errorf("missing token mount in:\n%s", out)
	}
	// Both should be on the same RUN line.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "RUN") && strings.Contains(line, "pip-conf") {
			if !strings.Contains(line, "token") {
				t.Error("both secret mounts should be on the same RUN line")
			}
		}
	}
}

func TestEmitSecretMount_NoSecrets_CleanOutput(t *testing.T) {
	plan := secretPlan(core.Step{
		Type: core.StepRun,
		Args: []string{"pip install -r requirements.txt"},
	})
	out := EmitDockerfile(plan)
	if strings.Contains(out, "--mount=type=secret") {
		t.Errorf("expected no secret mounts in:\n%s", out)
	}
}
