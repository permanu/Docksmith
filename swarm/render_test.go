package swarm

import (
	"errors"
	"strings"
	"testing"

	"github.com/permanu/docksmith/blueprint"
)

func TestRenderStatelessReplicatedService(t *testing.T) {
	got, err := RenderStack(Stack{
		Services: []Service{{
			Name:  "web",
			Image: "registry.example.com/app:web",
			Environment: map[string]string{
				"B": "two",
				"A": "one",
			},
			Ports:    []Port{{Target: 8080, Published: 80, Protocol: "tcp", Mode: "ingress"}},
			Networks: []string{"public"},
			Deploy: Deploy{
				Replicas: 3,
				Resources: Resources{
					Limits:       ResourceSpec{CPUs: "1.0", Memory: "512M"},
					Reservations: ResourceSpec{Memory: "128M"},
				},
				UpdateConfig:   RolloutConfig{Parallelism: 1, Delay: "10s", FailureAction: "rollback", Order: "start-first"},
				RollbackConfig: RolloutConfig{Parallelism: 1, Delay: "5s", FailureAction: "pause"},
				Placement:      Placement{Constraints: []string{"node.platform.os == linux"}},
			},
			Healthcheck: &Healthcheck{
				Test:     []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
				Interval: "30s",
				Timeout:  "5s",
				Retries:  3,
			},
		}},
		Networks: []Network{{Name: "public"}},
	})
	if err != nil {
		t.Fatalf("RenderStack returned error: %v", err)
	}

	want := `version: "3.9"
services:
  web:
    image: "registry.example.com/app:web"
    environment:
      A: "one"
      B: "two"
    ports:
      - target: 8080
        published: 80
        protocol: "tcp"
        mode: "ingress"
    networks:
      - "public"
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: "1.0"
          memory: "512M"
        reservations:
          memory: "128M"
      update_config:
        parallelism: 1
        delay: "10s"
        failure_action: "rollback"
        order: "start-first"
      rollback_config:
        parallelism: 1
        delay: "5s"
        failure_action: "pause"
      placement:
        constraints:
          - "node.platform.os == linux"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: "30s"
      timeout: "5s"
      retries: 3
networks:
  public:
    driver: "overlay"
`
	if got != want {
		t.Fatalf("unexpected yaml\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderStatefulSinglePrimaryService(t *testing.T) {
	got, err := RenderStack(Stack{
		Services: []Service{{
			Name:     "postgres",
			Image:    "postgres:16",
			Stateful: true,
			Mounts: []Mount{{
				Type:   "volume",
				Source: "pgdata",
				Target: "/var/lib/postgresql/data",
			}},
			Deploy: Deploy{
				Replicas: 1,
				Placement: Placement{Constraints: []string{
					"node.labels.disk == ssd",
					"node.role == worker",
				}},
			},
		}},
		Volumes: []Volume{{Name: "pgdata"}},
	})
	if err != nil {
		t.Fatalf("RenderStack returned error: %v", err)
	}

	want := `version: "3.9"
services:
  postgres:
    image: "postgres:16"
    volumes:
      - type: "volume"
        source: "pgdata"
        target: "/var/lib/postgresql/data"
    deploy:
      replicas: 1
      placement:
        constraints:
          - "node.labels.disk == ssd"
          - "node.role == worker"
volumes:
  pgdata:
    driver: "local"
`
	if got != want {
		t.Fatalf("unexpected yaml\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderSecretReferences(t *testing.T) {
	got, err := RenderStack(Stack{
		Services: []Service{{
			Name:  "api",
			Image: "api:latest",
			Secrets: []ServiceSecret{
				{Source: "db_password", Target: "db_password", Mode: "0400"},
				{Source: "api_token"},
			},
		}},
		Secrets: []Secret{
			{Name: "db_password", File: "./secrets/db_password.txt"},
			{Name: "api_token", External: true},
		},
	})
	if err != nil {
		t.Fatalf("RenderStack returned error: %v", err)
	}

	want := `version: "3.9"
services:
  api:
    image: "api:latest"
    secrets:
      - source: "api_token"
      - source: "db_password"
        target: "db_password"
        mode: "0400"
secrets:
  api_token:
    external: true
  db_password:
    file: "./secrets/db_password.txt"
`
	if got != want {
		t.Fatalf("unexpected yaml\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestValidateRejectsInvalidComposeNames(t *testing.T) {
	validService := Service{Name: "api", Image: "api:latest"}
	tests := []struct {
		name string
		spec Stack
	}{
		{
			name: "service",
			spec: Stack{Services: []Service{{Name: "bad name", Image: "api:latest"}}},
		},
		{
			name: "secret",
			spec: Stack{
				Services: []Service{validService},
				Secrets:  []Secret{{Name: "bad/secret"}},
			},
		},
		{
			name: "config",
			spec: Stack{
				Services: []Service{validService},
				Configs:  []Config{{Name: "-bad-config"}},
			},
		},
		{
			name: "network",
			spec: Stack{
				Services: []Service{validService},
				Networks: []Network{{Name: "bad network"}},
			},
		},
		{
			name: "volume",
			spec: Stack{
				Services: []Service{validService},
				Volumes:  []Volume{{Name: strings.Repeat("a", 64)}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderStack(tt.spec)
			if !errors.Is(err, ErrInvalidSpec) {
				t.Fatalf("expected ErrInvalidSpec, got %v", err)
			}
		})
	}
}

func TestValidateRejectsUnsafeEnvironmentAndLabelKeys(t *testing.T) {
	tests := []struct {
		name string
		svc  Service
	}{
		{
			name: "environment yaml injection",
			svc: Service{
				Name:  "api",
				Image: "api:latest",
				Environment: map[string]string{
					"GOOD:\n    privileged": "true",
				},
			},
		},
		{
			name: "environment not posix",
			svc: Service{
				Name:  "api",
				Image: "api:latest",
				Environment: map[string]string{
					"1BAD": "value",
				},
			},
		},
		{
			name: "label yaml injection",
			svc: Service{
				Name:  "api",
				Image: "api:latest",
				Labels: map[string]string{
					"com.example.bad:\n    privileged": "true",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderStack(Stack{Services: []Service{tt.svc}})
			if !errors.Is(err, ErrInvalidSpec) {
				t.Fatalf("expected ErrInvalidSpec, got %v", err)
			}
		})
	}
}

func TestValidateRejectsMissingServiceSecretDeclaration(t *testing.T) {
	_, err := RenderStack(Stack{
		Services: []Service{{
			Name:    "api",
			Image:   "api:latest",
			Secrets: []ServiceSecret{{Source: "api_token"}},
		}},
	})
	if !errors.Is(err, ErrInvalidSpec) {
		t.Fatalf("expected ErrInvalidSpec, got %v", err)
	}
}

func TestValidateRejectsUnsafeStatefulReplicas(t *testing.T) {
	_, err := RenderStack(Stack{
		Services: []Service{{
			Name:     "mysql",
			Image:    "mysql:8",
			Stateful: true,
			Deploy:   Deploy{Replicas: 2},
		}},
	})
	if !errors.Is(err, ErrUnsupportedStatefulScale) {
		t.Fatalf("expected ErrUnsupportedStatefulScale, got %v", err)
	}
}

func TestRenderBlueprintValueFromSecret(t *testing.T) {
	got, err := RenderBlueprintStack(blueprint.DeploymentPlan{
		Spec: blueprint.DeploymentSpec{
			Name:        "app",
			Environment: blueprint.EnvironmentStaging,
			Mode:        blueprint.ModeSwarm,
			Components: []blueprint.Component{{
				Name:  "api",
				Kind:  blueprint.ComponentApp,
				Image: "api:v1",
				Env: []blueprint.Env{
					{Name: "LOG_LEVEL", Value: "info"},
					{Name: "API_TOKEN", ValueFrom: "secret:api_token"},
				},
			}},
			Secrets: []blueprint.Secret{{Name: "api_token"}},
		},
	})
	if err != nil {
		t.Fatalf("RenderBlueprintStack returned error: %v", err)
	}

	want := `version: "3.9"
services:
  api:
    image: "api:v1"
    environment:
      API_TOKEN_FILE: "/run/secrets/api_token"
      LOG_LEVEL: "info"
    secrets:
      - source: "api_token"
    deploy:
      replicas: 1
      rollback_config:
        failure_action: "pause"
secrets:
  api_token:
    external: true
`
	if got != want {
		t.Fatalf("unexpected yaml\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if strings.Contains(got, "API_TOKEN:") {
		t.Fatalf("secret env name was injected into environment:\n%s", got)
	}
}

func TestRenderBlueprintDistrolessHealthPathDoesNotGenerateWget(t *testing.T) {
	got, err := RenderBlueprintStack(blueprint.DeploymentPlan{
		Spec: blueprint.DeploymentSpec{
			Name:        "app",
			Environment: blueprint.EnvironmentStaging,
			Mode:        blueprint.ModeSwarm,
			Components: []blueprint.Component{{
				Name:  "api",
				Kind:  blueprint.ComponentApp,
				Image: "gcr.io/distroless/static-debian12:nonroot",
				Process: blueprint.Process{
					Ports: []blueprint.Port{{Container: 8080}},
				},
				Health: blueprint.Health{Path: "/health"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("RenderBlueprintStack returned error: %v", err)
	}
	if strings.Contains(got, "healthcheck:") || strings.Contains(got, "wget") {
		t.Fatalf("distroless image should not get generated wget healthcheck:\n%s", got)
	}
}

func TestRenderBlueprintStack(t *testing.T) {
	got, err := RenderBlueprintStack(blueprint.DeploymentPlan{
		Spec: blueprint.DeploymentSpec{
			Name:        "app",
			Environment: blueprint.EnvironmentStaging,
			Mode:        blueprint.ModeSwarm,
			Placement: blueprint.Placement{
				Networks: []string{"edge"},
			},
			Components: []blueprint.Component{{
				Name:  "api",
				Kind:  blueprint.ComponentApp,
				Image: "api:v1",
				Process: blueprint.Process{
					Ports: []blueprint.Port{{Container: 8080, Published: 8080, Protocol: "tcp"}},
				},
				Scaling: blueprint.Scaling{Replicas: 2},
				Health:  blueprint.Health{Path: "/health", Interval: "10s"},
			}},
			Secrets: []blueprint.Secret{{Name: "api_token"}},
			Upgrade: blueprint.Upgrade{
				Parallelism:   1,
				Delay:         "5s",
				FailureAction: "rollback",
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderBlueprintStack returned error: %v", err)
	}

	want := `version: "3.9"
services:
  api:
    image: "api:v1"
    ports:
      - target: 8080
        published: 8080
        protocol: "tcp"
    networks:
      - "edge"
    deploy:
      replicas: 2
      update_config:
        parallelism: 1
        delay: "5s"
        failure_action: "rollback"
      rollback_config:
        parallelism: 1
        delay: "5s"
        failure_action: "pause"
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8080/health >/dev/null || exit 1"]
      interval: "10s"
secrets:
  api_token:
    external: true
networks:
  edge:
    driver: "overlay"
`
	if got != want {
		t.Fatalf("unexpected yaml\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
