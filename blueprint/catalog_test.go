package blueprint

import (
	"strings"
	"testing"
)

func TestBuiltinRegistryLookup(t *testing.T) {
	r := BuiltinRegistry()

	bp, ok := r.Lookup("postgres-single")
	if !ok {
		t.Fatal("Lookup(postgres-single) ok = false")
	}
	if bp.Name != "postgres-single" {
		t.Fatalf("Lookup(postgres-single).Name = %q", bp.Name)
	}

	if _, ok := r.Lookup("missing"); ok {
		t.Fatal("Lookup(missing) ok = true, want false")
	}
}

func TestRegistryRejectsDuplicateBlueprint(t *testing.T) {
	_, err := NewRegistry(PostgresSingleBlueprint(), PostgresSingleBlueprint())
	if err == nil {
		t.Fatal("NewRegistry() error = nil, want duplicate rejection")
	}
	if !strings.Contains(err.Error(), "duplicate blueprint") {
		t.Fatalf("NewRegistry() error = %q, want duplicate message", err)
	}
}

func TestBlueprintProfileValidation(t *testing.T) {
	profile := BlueprintProfile{
		Name:        "production",
		Environment: EnvironmentProduction,
		Mode:        ModeSwarm,
		Component: Component{
			Name:    "web",
			Kind:    ComponentApp,
			Process: Process{Ports: []Port{{Container: 8080}}},
		},
	}

	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	profile.Mode = Mode("nomad")
	err := profile.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want unsupported mode")
	}
	if !strings.Contains(err.Error(), "unsupported mode") {
		t.Fatalf("Validate() error = %q, want unsupported mode", err)
	}
}

func TestBlueprintValidationRequiresProductionStatefulBackup(t *testing.T) {
	bp := RedisSingleBlueprint()
	bp.Profiles[0].Component.Backup = Backup{}

	err := bp.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want backup requirement")
	}
	if !strings.Contains(err.Error(), "production stateful components require backup") {
		t.Fatalf("Validate() error = %q, want backup message", err)
	}
}

func TestPostgresBlueprintExposesDatabaseURL(t *testing.T) {
	bp := PostgresSingleBlueprint()

	var found bool
	for i := range bp.Outputs {
		if bp.Outputs[i].Name == "DATABASE_URL" {
			found = true
			if !bp.Outputs[i].Secret {
				t.Fatal("DATABASE_URL Secret = false, want true")
			}
		}
	}
	if !found {
		t.Fatal("postgres-single outputs missing DATABASE_URL")
	}
}

func TestBlueprintProfileConvertsToDeploymentSpec(t *testing.T) {
	bp := AppWebBlueprint()
	profile, ok := bp.Profile("production")
	if !ok {
		t.Fatal("Profile(production) ok = false")
	}

	spec, err := profile.DeploymentSpec("shop")
	if err != nil {
		t.Fatalf("DeploymentSpec() error = %v", err)
	}
	if spec.Name != "shop" {
		t.Fatalf("DeploymentSpec().Name = %q, want shop", spec.Name)
	}
	if len(spec.Components) != 1 || spec.Components[0].Name != "web" {
		t.Fatalf("DeploymentSpec().Components = %#v, want web component", spec.Components)
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("converted spec Validate() error = %v", err)
	}
}
