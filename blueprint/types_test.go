package blueprint

import (
	"strings"
	"testing"
)

func TestDeploymentSpecValidateAcceptsProductionAppAndStateful(t *testing.T) {
	spec := DeploymentSpec{
		Name:        "shop",
		Environment: EnvironmentProduction,
		Mode:        ModeSwarm,
		Components: []Component{
			{
				Name:  "web",
				Kind:  ComponentApp,
				Image: "example.com/shop/web:1",
				Process: Process{
					Type:  ProcessTypeWeb,
					Ports: []Port{{Name: "http", Container: 8080}},
					DependsOn: []DependsOn{{
						Component: "postgres",
						Condition: DependencyConditionHealthy,
					}},
				},
				Scaling: Scaling{Replicas: 2},
				Env: []Env{
					{Name: "DATABASE_URL", ValueFrom: "secret:database-url", Required: true},
					{Name: "LOG_LEVEL", Value: "info"},
				},
				Health: Health{Path: "/healthz"},
			},
			{
				Name:     "postgres",
				Kind:     ComponentStateful,
				Image:    "postgres:16",
				Stateful: &Stateful{DataPath: "/var/lib/postgresql/data", VolumeName: "pgdata"},
				Backup: Backup{
					Enabled:        true,
					Schedule:       "0 2 * * *",
					Retention:      "168h",
					DestinationRef: "secret:backup-bucket",
				},
			},
		},
		Release: Release{Commands: []ReleaseCommand{{
			Name:    "migrate",
			Command: []string{"bin/migrate"},
			DependsOn: []DependsOn{{
				Component: "postgres",
				Condition: DependencyConditionHealthy,
			}},
		}}},
		Secrets: []Secret{
			{Name: "database-url", Required: true},
		},
	}

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestComponentValidateRejectsSensitiveLiteralEnv(t *testing.T) {
	c := Component{
		Name: "web",
		Kind: ComponentApp,
		Env:  []Env{{Name: "API_KEY", Value: "literal-secret"}},
	}

	err := c.Validate(EnvironmentProduction)
	if err == nil {
		t.Fatal("Validate() error = nil, want sensitive literal rejection")
	}
	if !strings.Contains(err.Error(), "sensitive env must use value_from") {
		t.Fatalf("Validate() error = %q, want sensitive env message", err)
	}
}

func TestComponentValidateRejectsUnsafeStatefulScaling(t *testing.T) {
	c := Component{
		Name:     "postgres",
		Kind:     ComponentStateful,
		Stateful: &Stateful{DataPath: "/data"},
		Scaling: Scaling{
			Replicas:    2,
			Autoscaling: Autoscaling{Enabled: true, MinReplicas: 1, MaxReplicas: 3},
		},
		Backup: Backup{Enabled: true},
	}

	err := c.Validate(EnvironmentProduction)
	if err == nil {
		t.Fatal("Validate() error = nil, want unsafe stateful scaling rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "stateful autoscaling requires explicit capability") {
		t.Fatalf("Validate() error = %q, want autoscaling capability message", msg)
	}
	if !strings.Contains(msg, "stateful replicas > 1 require explicit HA support") {
		t.Fatalf("Validate() error = %q, want HA support message", msg)
	}
}

func TestComponentValidateAllowsExplicitStatefulHAAndAutoscaling(t *testing.T) {
	c := Component{
		Name:     "queue",
		Kind:     ComponentStateful,
		Stateful: &Stateful{DataPath: "/data", HA: true},
		Scaling: Scaling{
			Replicas:    3,
			Autoscaling: Autoscaling{Enabled: true, MinReplicas: 2, MaxReplicas: 5},
		},
		Capabilities: Capabilities{StatefulHA: true, StatefulAutoscaling: true},
		Backup:       Backup{Enabled: true},
	}

	if err := c.Validate(EnvironmentProduction); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProcessValidateRequiresCronSchedule(t *testing.T) {
	c := Component{
		Name:    "digest",
		Kind:    ComponentApp,
		Process: Process{Type: ProcessTypeCron},
	}

	err := c.Validate(EnvironmentProduction)
	if err == nil {
		t.Fatal("Validate() error = nil, want cron schedule rejection")
	}
	if !strings.Contains(err.Error(), "cron process requires schedule") {
		t.Fatalf("Validate() error = %q, want cron schedule message", err)
	}
}

func TestProcessValidateRequiresProductionWebPortUnlessInternalOnly(t *testing.T) {
	t.Run("external web", func(t *testing.T) {
		c := Component{
			Name:    "web",
			Kind:    ComponentApp,
			Process: Process{Type: ProcessTypeWeb},
		}

		err := c.Validate(EnvironmentProduction)
		if err == nil {
			t.Fatal("Validate() error = nil, want web port rejection")
		}
		if !strings.Contains(err.Error(), "production web process requires a port unless internal-only") {
			t.Fatalf("Validate() error = %q, want web port message", err)
		}
	})

	t.Run("internal web", func(t *testing.T) {
		c := Component{
			Name:    "admin",
			Kind:    ComponentApp,
			Process: Process{Type: ProcessTypeWeb, InternalOnly: true},
		}

		if err := c.Validate(EnvironmentProduction); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
}

func TestComponentValidateRejectsScaledReleaseProcess(t *testing.T) {
	c := Component{
		Name:    "migrate",
		Kind:    ComponentApp,
		Process: Process{Type: ProcessTypeRelease},
		Scaling: Scaling{
			Replicas:    2,
			Autoscaling: Autoscaling{Enabled: true},
		},
	}

	err := c.Validate(EnvironmentProduction)
	if err == nil {
		t.Fatal("Validate() error = nil, want release scale rejection")
	}
	if !strings.Contains(err.Error(), "release commands cannot scale") {
		t.Fatalf("Validate() error = %q, want release scale message", err)
	}
}

func TestDependsOnValidateRejectsUnsupportedCondition(t *testing.T) {
	c := Component{
		Name: "web",
		Kind: ComponentApp,
		Process: Process{
			Type: ProcessTypeWorker,
			DependsOn: []DependsOn{{
				Component: "redis",
				Condition: DependencyCondition("ready"),
			}},
		},
	}

	err := c.Validate(EnvironmentProduction)
	if err == nil {
		t.Fatal("Validate() error = nil, want depends_on condition rejection")
	}
	if !strings.Contains(err.Error(), `condition must be "started" or "healthy"`) {
		t.Fatalf("Validate() error = %q, want depends_on condition message", err)
	}
}

func TestReleaseValidateChecksCommandDependencies(t *testing.T) {
	spec := DeploymentSpec{
		Name:        "shop",
		Environment: EnvironmentProduction,
		Mode:        ModeSwarm,
		Components: []Component{{
			Name: "web",
			Kind: ComponentApp,
			Process: Process{
				Type:  ProcessTypeWeb,
				Ports: []Port{{Container: 8080}},
			},
		}},
		Release: Release{Commands: []ReleaseCommand{{
			Name: "migrate",
			DependsOn: []DependsOn{{
				Component: "postgres",
				Condition: DependencyCondition("ready"),
			}},
		}}},
	}

	err := spec.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want release dependency rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "blueprint: release") {
		t.Fatalf("Validate() error = %q, want release context", msg)
	}
	if !strings.Contains(msg, `condition must be "started" or "healthy"`) {
		t.Fatalf("Validate() error = %q, want depends_on condition message", msg)
	}
}

func TestDeploymentSpecValidateRejectsMissingProcessDependency(t *testing.T) {
	spec := validGraphSpec()
	spec.Components[0].Process.DependsOn[0].Component = "cache"

	err := spec.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing dependency rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, `component "web"`) {
		t.Fatalf("Validate() error = %q, want component context", msg)
	}
	if !strings.Contains(msg, `depends_on "cache" references unknown component`) {
		t.Fatalf("Validate() error = %q, want missing dependency message", msg)
	}
}

func TestDeploymentSpecValidateRejectsMissingReleaseDependency(t *testing.T) {
	spec := validGraphSpec()
	spec.Release.Commands[0].DependsOn[0].Component = "cache"

	err := spec.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing release dependency rejection")
	}
	if !strings.Contains(err.Error(), `release command "migrate": depends_on "cache" references unknown component`) {
		t.Fatalf("Validate() error = %q, want missing release dependency message", err)
	}
}

func TestDeploymentSpecValidateRejectsMissingEnvSecret(t *testing.T) {
	spec := validGraphSpec()
	spec.Components[0].Env[0].ValueFrom = "secret:missing-database-url"

	err := spec.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing env secret rejection")
	}
	if !strings.Contains(err.Error(), `env "DATABASE_URL" value_from references unknown secret "missing-database-url"`) {
		t.Fatalf("Validate() error = %q, want missing env secret message", err)
	}
}

func TestDeploymentSpecValidateRejectsMissingSecretMount(t *testing.T) {
	spec := validGraphSpec()
	spec.Components[0].Secrets[0].Name = "missing-tls-key"

	err := spec.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing secret mount rejection")
	}
	if !strings.Contains(err.Error(), `secret mount "missing-tls-key" references unknown secret`) {
		t.Fatalf("Validate() error = %q, want missing secret mount message", err)
	}
}

func TestDeploymentSpecValidateAcceptsValidGraph(t *testing.T) {
	spec := validGraphSpec()

	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestComponentValidateRequiresProductionStatefulBackup(t *testing.T) {
	c := Component{
		Name:     "redis",
		Kind:     ComponentStateful,
		Stateful: &Stateful{DataPath: "/data"},
	}

	err := c.Validate(EnvironmentProduction)
	if err == nil {
		t.Fatal("Validate() error = nil, want backup requirement")
	}
	if !strings.Contains(err.Error(), "production stateful components require backup") {
		t.Fatalf("Validate() error = %q, want backup message", err)
	}
}

func validGraphSpec() DeploymentSpec {
	return DeploymentSpec{
		Name:        "shop",
		Environment: EnvironmentProduction,
		Mode:        ModeSwarm,
		Components: []Component{
			{
				Name:  "web",
				Kind:  ComponentApp,
				Image: "example.com/shop/web:1",
				Process: Process{
					Type:  ProcessTypeWeb,
					Ports: []Port{{Name: "http", Container: 8080}},
					DependsOn: []DependsOn{{
						Component: "postgres",
						Condition: DependencyConditionHealthy,
					}},
				},
				Env: []Env{{
					Name:      "DATABASE_URL",
					ValueFrom: "secret:database-url",
					Required:  true,
				}},
				Secrets: []SecretMount{{
					Name:    "tls-key",
					EnvName: "TLS_KEY",
				}},
			},
			{
				Name:     "postgres",
				Kind:     ComponentStateful,
				Image:    "postgres:16",
				Stateful: &Stateful{DataPath: "/var/lib/postgresql/data", VolumeName: "pgdata"},
				Backup: Backup{
					Enabled:        true,
					Schedule:       "0 2 * * *",
					Retention:      "168h",
					DestinationRef: "secret:backup-bucket",
				},
			},
		},
		Release: Release{Commands: []ReleaseCommand{{
			Name:    "migrate",
			Command: []string{"bin/migrate"},
			DependsOn: []DependsOn{{
				Component: "postgres",
				Condition: DependencyConditionHealthy,
			}},
		}}},
		Secrets: []Secret{
			{Name: "database-url", Required: true},
			{Name: "tls-key", Required: true},
		},
	}
}

func TestDeploymentPlanValidateChecksResolvedComponents(t *testing.T) {
	plan := DeploymentPlan{
		Spec: DeploymentSpec{
			Name:        "shop",
			Environment: EnvironmentProduction,
			Mode:        ModeSingleContainer,
			Components: []Component{{
				Name: "web",
				Kind: ComponentApp,
			}},
		},
		Components: []ComponentPlan{{
			Mode: Mode("nomad"),
			Component: Component{
				Name: "db",
				Kind: ComponentStateful,
				Env:  []Env{{Name: "PASSWORD", Value: "bad"}},
			},
		}},
	}

	err := plan.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want resolved component errors")
	}
	msg := err.Error()
	for _, want := range []string{
		"unsupported mode",
		"stateful details are required",
		"sensitive env must use value_from",
		"production stateful components require backup",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Validate() error = %q, want %q", msg, want)
		}
	}
}
