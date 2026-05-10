// Package swarm renders neutral deployment specs into Docker Swarm stack YAML.
package swarm

import "errors"

var (
	ErrInvalidSpec              = errors.New("invalid swarm spec")
	ErrUnsupportedStatefulScale = errors.New("unsupported unsafe stateful replicas")
)

// Stack is a neutral deployment description consumed by the Swarm renderer.
//
// Integration point: when the blueprint package lands, either alias its
// deployment spec to these shapes or add a narrow adapter that fills Stack
// without changing RenderStack.
type Stack struct {
	Version  string
	Services []Service
	Secrets  []Secret
	Configs  []Config
	Networks []Network
	Volumes  []Volume
}

type Service struct {
	Name        string
	Image       string
	Command     []string
	Entrypoint  []string
	Environment map[string]string
	Labels      map[string]string
	Ports       []Port
	Mounts      []Mount
	Secrets     []ServiceSecret
	Configs     []ServiceConfig
	Networks    []string
	Deploy      Deploy
	Healthcheck *Healthcheck
	Stateful    bool
	// StatefulReplicasSafe allows replicas > 1 only when an upstream blueprint
	// has explicitly declared HA/stateful scaling support.
	StatefulReplicasSafe bool
}

type Port struct {
	Target    int
	Published int
	Protocol  string
	Mode      string
}

type Mount struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool
}

type ServiceSecret struct {
	Source string
	Target string
	UID    string
	GID    string
	Mode   string
}

type ServiceConfig struct {
	Source string
	Target string
	UID    string
	GID    string
	Mode   string
}

type Deploy struct {
	Replicas       int
	Resources      Resources
	UpdateConfig   RolloutConfig
	RollbackConfig RolloutConfig
	Placement      Placement
}

type Resources struct {
	Limits       ResourceSpec
	Reservations ResourceSpec
}

type ResourceSpec struct {
	CPUs   string
	Memory string
}

type RolloutConfig struct {
	Parallelism     int
	Delay           string
	FailureAction   string
	Monitor         string
	MaxFailureRatio string
	Order           string
}

type Placement struct {
	Constraints []string
}

type Healthcheck struct {
	Test        []string
	Interval    string
	Timeout     string
	StartPeriod string
	Retries     int
}

type Secret struct {
	Name     string
	File     string
	External bool
}

type Config struct {
	Name     string
	File     string
	External bool
}

type Network struct {
	Name     string
	Driver   string
	External bool
}

type Volume struct {
	Name     string
	Driver   string
	External bool
}
