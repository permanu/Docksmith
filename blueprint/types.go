// Package blueprint defines Docksmith's production deployment contract.
package blueprint

import (
	"errors"
	"fmt"
	"strings"
)

// Mode selects the deployment backend shape.
type Mode string

const (
	ModeSingleContainer Mode = "single-container"
	ModeSwarm           Mode = "swarm"
)

// Environment selects validation strictness for a deployment.
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

// ComponentKind identifies the runtime responsibility of a component.
type ComponentKind string

const (
	ComponentApp      ComponentKind = "app"
	ComponentStateful ComponentKind = "stateful"
)

// ProcessType identifies the lifecycle role of a process.
type ProcessType string

const (
	ProcessTypeWeb     ProcessType = "web"
	ProcessTypeWorker  ProcessType = "worker"
	ProcessTypeCron    ProcessType = "cron"
	ProcessTypeRelease ProcessType = "release"
)

// DependencyCondition identifies when a dependency is ready enough to start.
type DependencyCondition string

const (
	DependencyConditionStarted DependencyCondition = "started"
	DependencyConditionHealthy DependencyCondition = "healthy"
)

// DeploymentSpec is the user/authored production blueprint input.
type DeploymentSpec struct {
	SchemaVersion string      `json:"schema_version"`
	Name          string      `json:"name"`
	Environment   Environment `json:"environment"`
	Mode          Mode        `json:"mode"`
	Components    []Component `json:"components"`
	Release       Release     `json:"release,omitempty"`
	Secrets       []Secret    `json:"secrets,omitempty"`
	Placement     Placement   `json:"placement,omitempty"`
	Upgrade       Upgrade     `json:"upgrade,omitempty"`
	Metadata      []Label     `json:"metadata,omitempty"`
}

// DeploymentPlan is the resolved contract that an emitter/orchestrator can consume.
type DeploymentPlan struct {
	Spec       DeploymentSpec  `json:"spec"`
	Components []ComponentPlan `json:"components"`
}

// ComponentPlan carries resolved placement/runtime data for one component.
type ComponentPlan struct {
	Component Component `json:"component"`
	Mode      Mode      `json:"mode"`
	Placement Placement `json:"placement,omitempty"`
}

// Component describes either the application container or a backing stateful service.
type Component struct {
	Name         string        `json:"name"`
	Kind         ComponentKind `json:"kind"`
	Image        string        `json:"image,omitempty"`
	Process      Process       `json:"process,omitempty"`
	Stateful     *Stateful     `json:"stateful,omitempty"`
	Scaling      Scaling       `json:"scaling,omitempty"`
	Env          []Env         `json:"env,omitempty"`
	Secrets      []SecretMount `json:"secrets,omitempty"`
	Health       Health        `json:"health,omitempty"`
	Backup       Backup        `json:"backup,omitempty"`
	Upgrade      Upgrade       `json:"upgrade,omitempty"`
	Placement    Placement     `json:"placement,omitempty"`
	Capabilities Capabilities  `json:"capabilities,omitempty"`
}

// Process describes a container process.
type Process struct {
	Type            ProcessType `json:"type,omitempty"`
	Command         []string    `json:"command,omitempty"`
	Args            []string    `json:"args,omitempty"`
	WorkingDir      string      `json:"working_dir,omitempty"`
	User            string      `json:"user,omitempty"`
	Ports           []Port      `json:"ports,omitempty"`
	Schedule        string      `json:"schedule,omitempty"`
	InternalOnly    bool        `json:"internal_only,omitempty"`
	DependsOn       []DependsOn `json:"depends_on,omitempty"`
	StopSignal      string      `json:"stop_signal,omitempty"`
	StopGracePeriod string      `json:"stop_grace_period,omitempty"`
}

// Release describes one-shot commands that run during a release, such as migrations.
type Release struct {
	Commands []ReleaseCommand `json:"commands,omitempty"`
}

// ReleaseCommand describes a one-shot release lifecycle command.
type ReleaseCommand struct {
	Name       string      `json:"name"`
	Command    []string    `json:"command,omitempty"`
	Args       []string    `json:"args,omitempty"`
	WorkingDir string      `json:"working_dir,omitempty"`
	User       string      `json:"user,omitempty"`
	DependsOn  []DependsOn `json:"depends_on,omitempty"`
}

// DependsOn describes a dependency and its readiness condition.
type DependsOn struct {
	Component string              `json:"component"`
	Condition DependencyCondition `json:"condition"`
}

// Port describes a component port.
type Port struct {
	Name      string `json:"name,omitempty"`
	Container int    `json:"container"`
	Published int    `json:"published,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

// Stateful describes persistent service data.
type Stateful struct {
	DataPath   string `json:"data_path"`
	VolumeName string `json:"volume_name,omitempty"`
	Storage    string `json:"storage,omitempty"`
	HA         bool   `json:"ha,omitempty"`
}

// Scaling describes manual and automatic replica policy.
type Scaling struct {
	Replicas    int         `json:"replicas,omitempty"`
	Autoscaling Autoscaling `json:"autoscaling,omitempty"`
}

// Autoscaling describes optional autoscaling bounds.
type Autoscaling struct {
	Enabled     bool `json:"enabled,omitempty"`
	MinReplicas int  `json:"min_replicas,omitempty"`
	MaxReplicas int  `json:"max_replicas,omitempty"`
	CPUPercent  int  `json:"cpu_percent,omitempty"`
}

// Capabilities records explicitly supported high-risk behaviors.
type Capabilities struct {
	StatefulHA          bool `json:"stateful_ha,omitempty"`
	StatefulAutoscaling bool `json:"stateful_autoscaling,omitempty"`
}

// Secret declares an external secret known to the deployment.
type Secret struct {
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
	Key      string `json:"key,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// SecretMount wires an external secret into a component.
type SecretMount struct {
	Name     string `json:"name"`
	EnvName  string `json:"env_name,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

// Env describes an environment variable. ValueFrom must be used for secrets.
type Env struct {
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"value_from,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

// Health describes readiness/liveness checks.
type Health struct {
	Path        string   `json:"path,omitempty"`
	Command     []string `json:"command,omitempty"`
	Interval    string   `json:"interval,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	StartPeriod string   `json:"start_period,omitempty"`
	Retries     int      `json:"retries,omitempty"`
}

// Backup describes backup policy for production stateful components.
type Backup struct {
	Enabled        bool   `json:"enabled,omitempty"`
	Schedule       string `json:"schedule,omitempty"`
	Retention      string `json:"retention,omitempty"`
	DestinationRef string `json:"destination_ref,omitempty"`
}

// Upgrade describes rollout behavior.
type Upgrade struct {
	Strategy        string  `json:"strategy,omitempty"`
	Order           string  `json:"order,omitempty"`
	Parallelism     int     `json:"parallelism,omitempty"`
	Delay           string  `json:"delay,omitempty"`
	FailureAction   string  `json:"failure_action,omitempty"`
	Monitor         string  `json:"monitor,omitempty"`
	MaxFailureRatio float64 `json:"max_failure_ratio,omitempty"`
}

// Placement describes scheduling constraints.
type Placement struct {
	Constraints []string `json:"constraints,omitempty"`
	Preferences []string `json:"preferences,omitempty"`
	Networks    []string `json:"networks,omitempty"`
}

// Label is a compact metadata key/value.
type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Validate checks production-safety invariants on the authored spec.
func (s *DeploymentSpec) Validate() error {
	if s == nil {
		return errors.New("blueprint: nil deployment spec")
	}
	var errs []error
	if s.Name == "" {
		errs = append(errs, errors.New("blueprint: deployment name is required"))
	}
	if s.Mode != "" && s.Mode != ModeSingleContainer && s.Mode != ModeSwarm {
		errs = append(errs, fmt.Errorf("blueprint: unsupported mode %q", s.Mode))
	}
	if len(s.Components) == 0 {
		errs = append(errs, errors.New("blueprint: at least one component is required"))
	}
	for i := range s.Components {
		if err := s.Components[i].Validate(s.Environment); err != nil {
			errs = append(errs, fmt.Errorf("blueprint: component %q: %w", s.Components[i].Name, err))
		}
	}
	if err := s.Release.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("blueprint: release: %w", err))
	}
	if err := s.validateGraph(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (s *DeploymentSpec) validateGraph() error {
	componentNames := make(map[string]struct{}, len(s.Components))
	for i := range s.Components {
		if s.Components[i].Name != "" {
			componentNames[s.Components[i].Name] = struct{}{}
		}
	}
	secretNames := make(map[string]struct{}, len(s.Secrets))
	for i := range s.Secrets {
		if s.Secrets[i].Name != "" {
			secretNames[s.Secrets[i].Name] = struct{}{}
		}
	}

	var errs []error
	for i := range s.Components {
		component := &s.Components[i]
		for j := range component.Process.DependsOn {
			dependency := component.Process.DependsOn[j].Component
			if dependency != "" && !hasName(componentNames, dependency) {
				errs = append(errs, fmt.Errorf("blueprint: component %q: depends_on %q references unknown component", component.Name, dependency))
			}
		}
		for j := range component.Env {
			valueFrom := component.Env[j].ValueFrom
			if valueFrom == "" {
				continue
			}
			name, ok := strings.CutPrefix(valueFrom, "secret:")
			if !ok || name == "" {
				errs = append(errs, fmt.Errorf("blueprint: component %q: env %q value_from must use secret:<name>", component.Name, component.Env[j].Name))
				continue
			}
			if !hasName(secretNames, name) {
				errs = append(errs, fmt.Errorf("blueprint: component %q: env %q value_from references unknown secret %q", component.Name, component.Env[j].Name, name))
			}
		}
		for j := range component.Secrets {
			name := component.Secrets[j].Name
			if name != "" && !hasName(secretNames, name) {
				errs = append(errs, fmt.Errorf("blueprint: component %q: secret mount %q references unknown secret", component.Name, name))
			}
		}
	}
	for i := range s.Release.Commands {
		command := &s.Release.Commands[i]
		for j := range command.DependsOn {
			dependency := command.DependsOn[j].Component
			if dependency != "" && !hasName(componentNames, dependency) {
				errs = append(errs, fmt.Errorf("blueprint: release command %q: depends_on %q references unknown component", command.Name, dependency))
			}
		}
	}
	return errors.Join(errs...)
}

func hasName(names map[string]struct{}, name string) bool {
	_, ok := names[name]
	return ok
}

// Validate checks resolved plan safety.
func (p *DeploymentPlan) Validate() error {
	if p == nil {
		return errors.New("blueprint: nil deployment plan")
	}
	errs := []error{p.Spec.Validate()}
	for i := range p.Components {
		if err := p.Components[i].Component.Validate(p.Spec.Environment); err != nil {
			errs = append(errs, fmt.Errorf("blueprint: planned component %q: %w", p.Components[i].Component.Name, err))
		}
		if p.Components[i].Mode != "" && p.Components[i].Mode != ModeSingleContainer && p.Components[i].Mode != ModeSwarm {
			errs = append(errs, fmt.Errorf("blueprint: planned component %q has unsupported mode %q", p.Components[i].Component.Name, p.Components[i].Mode))
		}
	}
	return errors.Join(errs...)
}

// Validate checks component-level safety invariants.
func (c *Component) Validate(env Environment) error {
	var errs []error
	if c.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if c.Kind != ComponentApp && c.Kind != ComponentStateful {
		errs = append(errs, fmt.Errorf("unsupported kind %q", c.Kind))
	}
	if c.Kind == ComponentStateful && c.Stateful == nil {
		errs = append(errs, errors.New("stateful details are required"))
	}
	if err := c.Process.Validate(env); err != nil {
		errs = append(errs, err)
	}
	if c.Kind == ComponentApp && (c.Process.Type == "" || c.Process.Type == ProcessTypeWeb) && env == EnvironmentProduction && !c.Process.InternalOnly && len(c.Process.Ports) == 0 {
		errs = append(errs, errors.New("production web process requires a port unless internal-only"))
	}
	replicas := c.Scaling.Replicas
	if replicas == 0 {
		replicas = 1
	}
	if c.Process.Type == ProcessTypeRelease {
		if c.Scaling.Replicas != 0 || c.Scaling.Autoscaling.Enabled {
			errs = append(errs, errors.New("release commands cannot scale"))
		}
	}
	if c.Kind == ComponentStateful {
		if c.Scaling.Autoscaling.Enabled && !c.Capabilities.StatefulAutoscaling {
			errs = append(errs, errors.New("stateful autoscaling requires explicit capability"))
		}
		if replicas > 1 && !c.Capabilities.StatefulHA && (c.Stateful == nil || !c.Stateful.HA) {
			errs = append(errs, errors.New("stateful replicas > 1 require explicit HA support"))
		}
		if env == EnvironmentProduction && !c.Backup.Enabled {
			errs = append(errs, errors.New("production stateful components require backup"))
		}
	}
	for i := range c.Env {
		if err := c.Env[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("env %q: %w", c.Env[i].Name, err))
		}
	}
	return errors.Join(errs...)
}

// Validate checks process lifecycle invariants.
func (p Process) Validate(env Environment) error {
	var errs []error
	switch p.Type {
	case "", ProcessTypeWeb, ProcessTypeWorker, ProcessTypeCron, ProcessTypeRelease:
	default:
		errs = append(errs, fmt.Errorf("unsupported process type %q", p.Type))
	}
	if p.Type == ProcessTypeCron && p.Schedule == "" {
		errs = append(errs, errors.New("cron process requires schedule"))
	}
	for i := range p.DependsOn {
		if err := p.DependsOn[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("depends_on %q: %w", p.DependsOn[i].Component, err))
		}
	}
	return errors.Join(errs...)
}

// Validate checks release command invariants.
func (r Release) Validate() error {
	var errs []error
	for i := range r.Commands {
		if err := r.Commands[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("command %q: %w", r.Commands[i].Name, err))
		}
	}
	return errors.Join(errs...)
}

// Validate checks one-shot release command invariants.
func (r ReleaseCommand) Validate() error {
	var errs []error
	if r.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	for i := range r.DependsOn {
		if err := r.DependsOn[i].Validate(); err != nil {
			errs = append(errs, fmt.Errorf("depends_on %q: %w", r.DependsOn[i].Component, err))
		}
	}
	return errors.Join(errs...)
}

// Validate checks dependency readiness invariants.
func (d DependsOn) Validate() error {
	var errs []error
	if d.Component == "" {
		errs = append(errs, errors.New("component is required"))
	}
	if d.Condition != DependencyConditionStarted && d.Condition != DependencyConditionHealthy {
		errs = append(errs, fmt.Errorf("condition must be %q or %q", DependencyConditionStarted, DependencyConditionHealthy))
	}
	return errors.Join(errs...)
}

// Validate rejects literal sensitive values. Non-sensitive literals are allowed.
func (e Env) Validate() error {
	if e.Name == "" {
		return errors.New("name is required")
	}
	if e.Value != "" && e.ValueFrom != "" {
		return errors.New("value and value_from are mutually exclusive")
	}
	if e.Value != "" && looksSensitiveName(e.Name) {
		return errors.New("sensitive env must use value_from")
	}
	return nil
}

func looksSensitiveName(name string) bool {
	n := strings.ToUpper(name)
	return strings.Contains(n, "SECRET") ||
		strings.Contains(n, "TOKEN") ||
		strings.Contains(n, "PASSWORD") ||
		strings.Contains(n, "PASSWD") ||
		strings.Contains(n, "PRIVATE_KEY") ||
		strings.Contains(n, "API_KEY") ||
		strings.Contains(n, "ACCESS_KEY")
}
