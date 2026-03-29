package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Config represents a user-provided docksmith.toml/yaml/json configuration.
type Config struct {
	Runtime        string            `toml:"runtime"          yaml:"runtime"          json:"runtime"`
	Version        string            `toml:"version"          yaml:"version"          json:"version,omitempty"`
	PackageManager string            `toml:"package_manager"  yaml:"package_manager"  json:"package_manager,omitempty"`
	Dockerfile     string            `toml:"dockerfile"       yaml:"dockerfile"       json:"dockerfile,omitempty"`
	Env            map[string]string `toml:"env"              yaml:"env"              json:"env,omitempty"`
	Build          BuildConfig       `toml:"build"            yaml:"build"            json:"build,omitempty"`
	Start          StartConfig       `toml:"start"            yaml:"start"            json:"start,omitempty"`
	Install        InstallConfig            `toml:"install"          yaml:"install"          json:"install,omitempty"`
	RuntimeConfig  RuntimeCfg               `toml:"runtime_config"   yaml:"runtime_config"   json:"runtime_config,omitempty"`
	Secrets        map[string]SecretConfig   `toml:"secrets"          yaml:"secrets"          json:"secrets,omitempty"`
}

// BuildConfig groups build-time overrides.
type BuildConfig struct {
	Command string `toml:"command"  yaml:"command"  json:"command,omitempty"`
	NoCache bool   `toml:"no_cache" yaml:"no_cache" json:"no_cache,omitempty"`
}

// StartConfig groups start-time overrides.
type StartConfig struct {
	Command    string   `toml:"command"    yaml:"command"    json:"command,omitempty"`
	Entrypoint []string `toml:"entrypoint" yaml:"entrypoint" json:"entrypoint,omitempty"`
}

// InstallConfig groups install-time overrides.
type InstallConfig struct {
	Command    string   `toml:"command"     yaml:"command"     json:"command,omitempty"`
	SystemDeps []string `toml:"system_deps" yaml:"system_deps" json:"system_deps,omitempty"`
}

// SecretConfig defines a build-time secret from docksmith.toml.
// At least one of Target (file mount path) or Env (env var name) must be set.
type SecretConfig struct {
	Target string `toml:"target" yaml:"target" json:"target,omitempty"`
	Env    string `toml:"env"    yaml:"env"    json:"env,omitempty"`
}

// RuntimeCfg groups runtime-stage overrides.
// User and Healthcheck use sentinel booleans because false disables the feature.
type RuntimeCfg struct {
	Image       string `toml:"image"  yaml:"image"  json:"image,omitempty"`
	Expose      int    `toml:"expose" yaml:"expose" json:"expose,omitempty"`
	User        string `toml:"-"      yaml:"-"      json:"-"`
	UserSet     bool   `toml:"-"      yaml:"-"      json:"-"`
	Healthcheck string `toml:"-"      yaml:"-"      json:"-"`
	HCSet       bool   `toml:"-"      yaml:"-"      json:"-"`
}

// rawRuntimeCfg accepts bool or string for user/healthcheck during decode.
type rawRuntimeCfg struct {
	Image       string `toml:"image"       yaml:"image"       json:"image,omitempty"`
	Expose      int    `toml:"expose"      yaml:"expose"      json:"expose,omitempty"`
	User        any    `toml:"user"        yaml:"user"        json:"user,omitempty"`
	Healthcheck any    `toml:"healthcheck" yaml:"healthcheck" json:"healthcheck,omitempty"`
}

func (r rawRuntimeCfg) normalize() (RuntimeCfg, error) {
	cfg := RuntimeCfg{Image: r.Image, Expose: r.Expose}
	if r.User != nil {
		switch v := r.User.(type) {
		case bool:
			if v {
				return cfg, fmt.Errorf("runtime_config.user: true is invalid; use a username string or false to disable")
			}
			cfg.UserSet = true
		case string:
			cfg.User = v
			cfg.UserSet = true
		default:
			return cfg, fmt.Errorf("runtime_config.user: must be string or false, got %T", r.User)
		}
	}
	if r.Healthcheck != nil {
		switch v := r.Healthcheck.(type) {
		case bool:
			if v {
				return cfg, fmt.Errorf("runtime_config.healthcheck: true is invalid; use a command string or false to disable")
			}
			cfg.HCSet = true
		case string:
			cfg.Healthcheck = v
			cfg.HCSet = true
		default:
			return cfg, fmt.Errorf("runtime_config.healthcheck: must be string or false, got %T", r.Healthcheck)
		}
	}
	return cfg, nil
}

// DefaultPorts maps runtime names to their default port.
var DefaultPorts = map[string]int{
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

// ValidRuntimes is the set of supported runtime names.
var ValidRuntimes = map[string]bool{
	"node": true, "python": true, "go": true, "php": true,
	"java": true, "dotnet": true, "rust": true, "ruby": true,
	"elixir": true, "deno": true, "bun": true, "static": true,
}

// DefaultFileNames is the ordered list of config filenames to search for.
var DefaultFileNames = []string{
	"docksmith.toml",
	"docksmith.yaml",
	"docksmith.yml",
	"docksmith.json",
	".docksmith.yaml",
}

// Load reads the first matching config file from dir using DefaultFileNames.
// Returns (nil, nil) if no config file exists.
func Load(dir string) (*Config, error) {
	return LoadWithNames(dir, DefaultFileNames)
}

// LoadWithNames reads the first matching config file from dir using the given names.
// Returns (nil, nil) if no config file exists.
func LoadWithNames(dir string, names []string) (*Config, error) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		cfg, parseErr := ParseConfig(name, data)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid %s: %w", name, parseErr)
		}
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", name, err)
		}
		cfg.applyDefaults()
		return cfg, nil
	}
	return nil, nil
}

type rawConfig struct {
	Runtime        string            `toml:"runtime"         yaml:"runtime"         json:"runtime"`
	Version        string            `toml:"version"         yaml:"version"         json:"version,omitempty"`
	PackageManager string            `toml:"package_manager" yaml:"package_manager" json:"package_manager,omitempty"`
	Dockerfile     string            `toml:"dockerfile"      yaml:"dockerfile"      json:"dockerfile,omitempty"`
	Env            map[string]string `toml:"env"             yaml:"env"             json:"env,omitempty"`
	Build          BuildConfig       `toml:"build"           yaml:"build"           json:"build,omitempty"`
	Start          StartConfig       `toml:"start"           yaml:"start"           json:"start,omitempty"`
	Install        InstallConfig            `toml:"install"         yaml:"install"         json:"install,omitempty"`
	RuntimeConfig  rawRuntimeCfg            `toml:"runtime_config"  yaml:"runtime_config"  json:"runtime_config,omitempty"`
	Secrets        map[string]SecretConfig   `toml:"secrets"         yaml:"secrets"         json:"secrets,omitempty"`
}

// ParseConfig parses raw config data based on the file extension in name.
func ParseConfig(name string, data []byte) (*Config, error) {
	var raw rawConfig
	switch {
	case strings.HasSuffix(name, ".json"):
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
	case strings.HasSuffix(name, ".toml"):
		md, err := toml.Decode(string(data), &raw)
		if err != nil {
			return nil, err
		}
		if undecoded := md.Undecoded(); len(undecoded) > 0 {
			keys := make([]string, len(undecoded))
			for i, k := range undecoded {
				keys[i] = k.String()
			}
			return nil, fmt.Errorf("unknown keys: %s", strings.Join(keys, ", "))
		}
	default:
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
	}

	rc, err := raw.RuntimeConfig.normalize()
	if err != nil {
		return nil, err
	}

	return &Config{
		Runtime:        raw.Runtime,
		Version:        raw.Version,
		PackageManager: raw.PackageManager,
		Dockerfile:     raw.Dockerfile,
		Env:            raw.Env,
		Build:          raw.Build,
		Start:          raw.Start,
		Install:        raw.Install,
		RuntimeConfig:  rc,
		Secrets:        raw.Secrets,
	}, nil
}

// Validate checks that the config has required fields and valid values.
func (c *Config) Validate() error {
	if c.Dockerfile != "" {
		return nil
	}
	if c.Runtime == "" {
		return fmt.Errorf("runtime is required (or specify dockerfile)")
	}
	if !ValidRuntimes[c.Runtime] {
		return fmt.Errorf("unsupported runtime %q; valid: node, python, go, php, java, dotnet, rust, ruby, elixir, deno, bun, static", c.Runtime)
	}
	if c.Start.Command == "" && c.Runtime != "static" {
		return fmt.Errorf("start.command is required for runtime %q", c.Runtime)
	}
	return c.validateSecrets()
}

func (c *Config) validateSecrets() error {
	for id, sec := range c.Secrets {
		if id == "" {
			return fmt.Errorf("secrets: empty secret ID")
		}
		if sec.Target == "" && sec.Env == "" {
			return fmt.Errorf("secrets.%s: at least one of target or env must be set", id)
		}
		if sec.Target != "" && strings.Contains(sec.Target, "..") {
			return fmt.Errorf("secrets.%s: target path must not contain '..'", id)
		}
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.RuntimeConfig.Expose == 0 {
		if p, ok := DefaultPorts[c.Runtime]; ok {
			c.RuntimeConfig.Expose = p
		}
	}
}
