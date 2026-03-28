package docksmith

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Config represents a user-provided docksmith.toml/yaml/json configuration.
// When present in the project root, it overrides all auto-detection.
type Config struct {
	Runtime        string            `toml:"runtime"         yaml:"runtime"         json:"runtime"`
	Version        string            `toml:"version"         yaml:"version"         json:"version,omitempty"`
	PackageManager string            `toml:"package_manager" yaml:"package_manager" json:"package_manager,omitempty"`
	Build          string            `toml:"build"           yaml:"build"           json:"build,omitempty"`
	Start          string            `toml:"start"           yaml:"start"           json:"start,omitempty"`
	Port           int               `toml:"port"            yaml:"port"            json:"port,omitempty"`
	Dockerfile     string            `toml:"dockerfile"      yaml:"dockerfile"      json:"dockerfile,omitempty"`
	Env            map[string]string `toml:"env"             yaml:"env"             json:"env,omitempty"`
	SystemDeps     []string          `toml:"system_deps"     yaml:"system_deps"     json:"system_deps,omitempty"`
}

// defaultPorts maps runtime names to their conventional default ports.
var defaultPorts = map[string]int{
	"node":   3000,
	"python": 8000,
	"go":     8080,
	"php":    80,
	"java":   8080,
	"dotnet": 8080,
	"rust":   8080,
	"ruby":   3000,
	"elixir": 4000,
	"deno":   8000,
	"bun":    3000,
	"static": 80,
}

// validRuntimes is the set of supported runtime identifiers.
var validRuntimes = map[string]bool{
	"node": true, "python": true, "go": true, "php": true,
	"java": true, "dotnet": true, "rust": true, "ruby": true,
	"elixir": true, "deno": true, "bun": true, "static": true,
}

// defaultConfigFileNames lists filenames checked in priority order.
var defaultConfigFileNames = []string{
	"docksmith.toml",
	"docksmith.yaml",
	"docksmith.yml",
	"docksmith.json",
	".docksmith.yaml",
}

// LoadConfig reads the first matching config file from dir using the default
// filename priority list. Returns (nil, nil) if no config file exists.
// Returns (nil, err) if a config file exists but is invalid.
func LoadConfig(dir string) (*Config, error) {
	return loadConfigWithNames(dir, defaultConfigFileNames)
}

// loadConfigWithNames is the internal loader used by both LoadConfig and
// DetectWithOptions (when ConfigFileNames is provided).
func loadConfigWithNames(dir string, names []string) (*Config, error) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		cfg, parseErr := parseConfig(name, data)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid %s: %w", name, parseErr)
		}
		if err := cfg.validate(); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", name, err)
		}
		cfg.applyDefaults()
		return cfg, nil
	}
	return nil, nil
}

// parseConfig parses raw bytes as TOML, YAML, or JSON based on the filename.
func parseConfig(name string, data []byte) (*Config, error) {
	var cfg Config
	switch {
	case strings.HasSuffix(name, ".json"):
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	case strings.HasSuffix(name, ".toml"):
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	default:
		// .yaml / .yml
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

// validate checks that required fields are present.
// Either (runtime + start) or dockerfile must be set.
func (c *Config) validate() error {
	if c.Dockerfile != "" {
		return nil // custom Dockerfile mode — nothing else required
	}
	if c.Runtime == "" {
		return fmt.Errorf("runtime is required (or specify dockerfile)")
	}
	if !validRuntimes[c.Runtime] {
		return fmt.Errorf("unsupported runtime %q; valid: node, python, go, php, java, dotnet, rust, ruby, elixir, deno, bun, static", c.Runtime)
	}
	if c.Start == "" && c.Runtime != "static" {
		return fmt.Errorf("start command is required for runtime %q", c.Runtime)
	}
	return nil
}

// applyDefaults fills in port if not specified by the user.
func (c *Config) applyDefaults() {
	if c.Port == 0 {
		if p, ok := defaultPorts[c.Runtime]; ok {
			c.Port = p
		}
	}
}

// ToFramework converts a Config to a Framework for Dockerfile generation.
// When Dockerfile is set, the returned Framework has Name "dockerfile" and
// the path stored in OutputDir (repurposed as the Dockerfile path indicator).
func (c *Config) ToFramework() *Framework {
	if c.Dockerfile != "" {
		return &Framework{
			Name:      "dockerfile",
			OutputDir: c.Dockerfile,
		}
	}

	fw := &Framework{
		Name:         c.runtimeToFrameworkName(),
		BuildCommand: c.Build,
		StartCommand: c.Start,
		Port:         c.Port,
		SystemDeps:   c.SystemDeps,
	}

	switch c.Runtime {
	case "node":
		fw.NodeVersion = c.Version
		fw.PackageManager = c.PackageManager
	case "python":
		fw.PythonVersion = c.Version
		fw.PythonPM = c.PackageManager
	case "go":
		fw.GoVersion = c.Version
		if fw.BuildCommand == "" {
			fw.BuildCommand = "go build -o app ."
		}
	case "php":
		fw.PHPVersion = c.Version
	case "java":
		fw.JavaVersion = c.Version
	case "dotnet":
		fw.DotnetVersion = c.Version
	case "deno":
		fw.DenoVersion = c.Version
	case "bun":
		fw.BunVersion = c.Version
		fw.PackageManager = "bun"
	case "rust":
		// no dedicated version field on Framework
	case "ruby":
		// no dedicated version field on Framework
	case "elixir":
		// no dedicated version field on Framework
	}

	return fw
}

// runtimeToFrameworkName maps the config runtime to the Framework.Name that
// the Dockerfile generator understands.
func (c *Config) runtimeToFrameworkName() string {
	switch c.Runtime {
	case "node":
		return "express" // generic Node.js
	case "python":
		return "flask" // generic Python
	case "go":
		return "go-std"
	case "php":
		return "php"
	case "java":
		return "maven"
	case "dotnet":
		return "aspnet-core"
	case "rust":
		return "rust-generic"
	case "ruby":
		return "rails"
	case "elixir":
		return "elixir-phoenix"
	case "deno":
		return "deno"
	case "bun":
		return "bun"
	case "static":
		return "static"
	default:
		return c.Runtime
	}
}
