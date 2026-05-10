package blueprint

import (
	"errors"
	"fmt"
)

// BlueprintKind describes the primary workload shape a blueprint produces.
type BlueprintKind string

const (
	BlueprintKindApp      BlueprintKind = "app"
	BlueprintKindStateful BlueprintKind = "stateful"
)

// ConfigFieldType describes the value expected for a blueprint config field.
type ConfigFieldType string

const (
	ConfigFieldString ConfigFieldType = "string"
	ConfigFieldInt    ConfigFieldType = "int"
	ConfigFieldBool   ConfigFieldType = "bool"
	ConfigFieldSecret ConfigFieldType = "secret"
)

// Blueprint is catalog metadata for a reusable deployment template.
type Blueprint struct {
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Kind         BlueprintKind      `json:"kind"`
	Capabilities []Capability       `json:"capabilities,omitempty"`
	Config       []ConfigField      `json:"config,omitempty"`
	Outputs      []Output           `json:"outputs,omitempty"`
	Modules      []Module           `json:"modules,omitempty"`
	Dependencies []Dependency       `json:"dependencies,omitempty"`
	Profiles     []BlueprintProfile `json:"profiles,omitempty"`
}

// BlueprintProfile is one validated deployment shape exposed by a blueprint.
type BlueprintProfile struct {
	Name        string      `json:"name"`
	Environment Environment `json:"environment"`
	Mode        Mode        `json:"mode"`
	Component   Component   `json:"component"`
}

// Capability describes an advertised feature or safety characteristic.
type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ConfigField declares template input metadata.
type ConfigField struct {
	Name        string          `json:"name"`
	Type        ConfigFieldType `json:"type"`
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Default     string          `json:"default,omitempty"`
}

// Output declares generated values a renderer may expose.
type Output struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"`
}

// Module groups implementation concerns without prescribing a renderer.
type Module struct {
	Name string `json:"name"`
}

// Dependency records a service dependency advertised by app blueprints.
type Dependency struct {
	Name     string `json:"name"`
	Required bool   `json:"required,omitempty"`
}

// Registry is a deterministic in-memory blueprint catalog.
type Registry struct {
	blueprints []Blueprint
}

// NewRegistry validates and stores blueprints in caller-provided order.
func NewRegistry(blueprints ...Blueprint) (*Registry, error) {
	r := &Registry{blueprints: make([]Blueprint, 0, len(blueprints))}
	for i := range blueprints {
		if err := r.Register(blueprints[i]); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// BuiltinRegistry returns the initial Docksmith blueprint catalog.
func BuiltinRegistry() *Registry {
	r, err := NewRegistry(PostgresSingleBlueprint(), RedisSingleBlueprint(), AppWebBlueprint())
	if err != nil {
		panic(err)
	}
	return r
}

// Register validates and appends a blueprint.
func (r *Registry) Register(b Blueprint) error {
	if r == nil {
		return errors.New("blueprint: nil registry")
	}
	if err := b.Validate(); err != nil {
		return err
	}
	for i := range r.blueprints {
		if r.blueprints[i].Name == b.Name {
			return fmt.Errorf("blueprint: duplicate blueprint %q", b.Name)
		}
	}
	r.blueprints = append(r.blueprints, b)
	return nil
}

// Lookup returns a blueprint by name.
func (r *Registry) Lookup(name string) (Blueprint, bool) {
	if r == nil {
		return Blueprint{}, false
	}
	for i := range r.blueprints {
		if r.blueprints[i].Name == name {
			return r.blueprints[i], true
		}
	}
	return Blueprint{}, false
}

// Blueprints returns a copy of catalog entries in registration order.
func (r *Registry) Blueprints() []Blueprint {
	if r == nil || len(r.blueprints) == 0 {
		return nil
	}
	out := make([]Blueprint, len(r.blueprints))
	copy(out, r.blueprints)
	return out
}

// Validate checks blueprint metadata and profile safety.
func (b *Blueprint) Validate() error {
	if b == nil {
		return errors.New("blueprint: nil blueprint")
	}
	var errs []error
	if b.Name == "" {
		errs = append(errs, errors.New("blueprint: name is required"))
	}
	if b.Kind != BlueprintKindApp && b.Kind != BlueprintKindStateful {
		errs = append(errs, fmt.Errorf("blueprint: unsupported kind %q", b.Kind))
	}
	for i := range b.Config {
		if err := b.Config[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("blueprint: config %q: %w", b.Config[i].Name, err))
		}
	}
	for i := range b.Outputs {
		if err := b.Outputs[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("blueprint: output %q: %w", b.Outputs[i].Name, err))
		}
	}
	for i := range b.Profiles {
		if err := b.Profiles[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("blueprint: profile %q: %w", b.Profiles[i].Name, err))
		}
		if b.Kind == BlueprintKindStateful && b.Profiles[i].Component.Kind != ComponentStateful {
			errs = append(errs, fmt.Errorf("blueprint: profile %q must use stateful component", b.Profiles[i].Name))
		}
		if b.Kind == BlueprintKindApp && b.Profiles[i].Component.Kind != ComponentApp {
			errs = append(errs, fmt.Errorf("blueprint: profile %q must use app component", b.Profiles[i].Name))
		}
	}
	return errors.Join(errs...)
}

// Profile returns a profile by name.
func (b Blueprint) Profile(name string) (BlueprintProfile, bool) {
	for i := range b.Profiles {
		if b.Profiles[i].Name == name {
			return b.Profiles[i], true
		}
	}
	return BlueprintProfile{}, false
}

// Validate checks a profile can become a one-component deployment spec.
func (p *BlueprintProfile) Validate() error {
	if p == nil {
		return errors.New("blueprint: nil profile")
	}
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.Environment != EnvironmentDevelopment && p.Environment != EnvironmentStaging && p.Environment != EnvironmentProduction {
		return fmt.Errorf("unsupported environment %q", p.Environment)
	}
	if p.Mode != ModeSingleContainer && p.Mode != ModeSwarm {
		return fmt.Errorf("unsupported mode %q", p.Mode)
	}
	return p.Component.Validate(p.Environment)
}

// DeploymentSpec converts validated profile metadata into a deployment input.
func (p BlueprintProfile) DeploymentSpec(name string) (DeploymentSpec, error) {
	if err := p.Validate(); err != nil {
		return DeploymentSpec{}, err
	}
	if name == "" {
		name = p.Component.Name
	}
	return DeploymentSpec{
		SchemaVersion: "blueprint.docksmith.dev/v1",
		Name:          name,
		Environment:   p.Environment,
		Mode:          p.Mode,
		Components:    []Component{p.Component},
	}, nil
}

// Validate checks config field metadata.
func (f ConfigField) Validate() error {
	if f.Name == "" {
		return errors.New("name is required")
	}
	switch f.Type {
	case ConfigFieldString, ConfigFieldInt, ConfigFieldBool, ConfigFieldSecret:
		return nil
	default:
		return fmt.Errorf("unsupported type %q", f.Type)
	}
}

// Validate checks output metadata.
func (o Output) Validate() error {
	if o.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// PostgresSingleBlueprint returns the built-in single-node Postgres template.
func PostgresSingleBlueprint() Blueprint {
	return Blueprint{
		Name:        "postgres-single",
		Description: "Single-node PostgreSQL with persistent storage.",
		Kind:        BlueprintKindStateful,
		Config: []ConfigField{
			{Name: "database", Type: ConfigFieldString, Required: true},
			{Name: "username", Type: ConfigFieldString, Required: true},
			{Name: "password", Type: ConfigFieldSecret, Required: true},
		},
		Outputs: []Output{
			{Name: "DATABASE_URL", Description: "PostgreSQL connection URL.", Secret: true},
		},
		Profiles: []BlueprintProfile{{
			Name:        "production",
			Environment: EnvironmentProduction,
			Mode:        ModeSingleContainer,
			Component: Component{
				Name:     "postgres",
				Kind:     ComponentStateful,
				Image:    "postgres:16",
				Stateful: &Stateful{DataPath: "/var/lib/postgresql/data", VolumeName: "postgres-data"},
				Backup:   Backup{Enabled: true},
			},
		}},
	}
}

// RedisSingleBlueprint returns the built-in single-node Redis template.
func RedisSingleBlueprint() Blueprint {
	return Blueprint{
		Name:        "redis-single",
		Description: "Single-node Redis with persistent storage.",
		Kind:        BlueprintKindStateful,
		Config: []ConfigField{
			{Name: "password", Type: ConfigFieldSecret},
		},
		Outputs: []Output{
			{Name: "REDIS_URL", Description: "Redis connection URL.", Secret: true},
		},
		Profiles: []BlueprintProfile{{
			Name:        "production",
			Environment: EnvironmentProduction,
			Mode:        ModeSingleContainer,
			Component: Component{
				Name:     "redis",
				Kind:     ComponentStateful,
				Image:    "redis:7",
				Stateful: &Stateful{DataPath: "/data", VolumeName: "redis-data"},
				Backup:   Backup{Enabled: true},
			},
		}},
	}
}

// AppWebBlueprint returns the built-in HTTP app template.
func AppWebBlueprint() Blueprint {
	return Blueprint{
		Name:        "app-web",
		Description: "HTTP web application container.",
		Kind:        BlueprintKindApp,
		Config: []ConfigField{
			{Name: "image", Type: ConfigFieldString, Required: true},
			{Name: "port", Type: ConfigFieldInt, Default: "8080"},
		},
		Outputs: []Output{
			{Name: "URL", Description: "Public HTTP endpoint."},
		},
		Dependencies: []Dependency{
			{Name: "database", Required: false},
		},
		Profiles: []BlueprintProfile{{
			Name:        "production",
			Environment: EnvironmentProduction,
			Mode:        ModeSingleContainer,
			Component: Component{
				Name: "web",
				Kind: ComponentApp,
				Process: Process{
					Type:  ProcessTypeWeb,
					Ports: []Port{{Name: "http", Container: 8080}},
				},
				Scaling: Scaling{Replicas: 1},
				Health:  Health{Path: "/healthz"},
			},
		}},
	}
}
