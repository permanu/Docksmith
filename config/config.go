package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// AssetCopy describes a file or directory to copy into the runtime stage.
type AssetCopy struct {
	Src   string `toml:"src"   yaml:"src"   json:"src"`
	Dst   string `toml:"dst"   yaml:"dst"   json:"dst"`
	Chown string `toml:"chown" yaml:"chown" json:"chown,omitempty"` // optional, e.g. "permanu:permanu" or "1000:1000"
}

// ExternalTool describes a versioned binary to fetch and verify during the build.
type ExternalTool struct {
	Name        string `toml:"name"         yaml:"name"         json:"name"`
	URL         string `toml:"url"          yaml:"url"          json:"url"`    // may contain ${TARGETARCH}
	SHA256      string `toml:"sha256"       yaml:"sha256"       json:"sha256"` // 64 hex chars
	InstallPath string `toml:"install_path" yaml:"install_path" json:"install_path"`
	Stage       string `toml:"stage"        yaml:"stage"        json:"stage"` // "builder" or "runtime", default "runtime"
}

// Config represents a user-provided docksmith.toml/yaml/json configuration.
type Config struct {
	Runtime        string                  `toml:"runtime"          yaml:"runtime"          json:"runtime"`
	Version        string                  `toml:"version"          yaml:"version"          json:"version,omitempty"`
	PackageManager string                  `toml:"package_manager"  yaml:"package_manager"  json:"package_manager,omitempty"`
	Dockerfile     string                  `toml:"dockerfile"       yaml:"dockerfile"       json:"dockerfile,omitempty"`
	ContextRoot    string                  `toml:"context_root"     yaml:"context_root"     json:"context_root,omitempty"`
	Env            map[string]string       `toml:"env"              yaml:"env"              json:"env,omitempty"`
	Build          BuildConfig             `toml:"build"            yaml:"build"            json:"build,omitempty"`
	Start          StartConfig             `toml:"start"            yaml:"start"            json:"start,omitempty"`
	Install        InstallConfig           `toml:"install"          yaml:"install"          json:"install,omitempty"`
	RuntimeConfig  RuntimeCfg              `toml:"runtime_config"   yaml:"runtime_config"   json:"runtime_config,omitempty"`
	Secrets        map[string]SecretConfig `toml:"secrets"          yaml:"secrets"          json:"secrets,omitempty"`
	RuntimeAssets  []AssetCopy             `toml:"runtime_assets"   yaml:"runtime_assets"   json:"runtime_assets,omitempty"`
	ExternalTools  []ExternalTool          `toml:"external_tools"   yaml:"external_tools"   json:"external_tools,omitempty"`
}

// Binary describes a single Go binary to build in a multi-binary project.
type Binary struct {
	Name          string   `toml:"name"           yaml:"name"           json:"name"`
	Path          string   `toml:"path"           yaml:"path"           json:"path"`
	OutputName    string   `toml:"output_name"    yaml:"output_name"    json:"output_name,omitempty"`
	BuildFlags    []string `toml:"build_flags"    yaml:"build_flags"    json:"build_flags,omitempty"`
	Architectures []string `toml:"architectures"  yaml:"architectures"  json:"architectures,omitempty"`
}

// validCrossArchitectures is the set of GOARCH values allowed in Binary.Architectures.
var validCrossArchitectures = map[string]bool{
	"amd64":   true,
	"arm64":   true,
	"arm":     true,
	"386":     true,
	"riscv64": true,
}

// BuildConfig groups build-time overrides.
type BuildConfig struct {
	Command  string            `toml:"command"    yaml:"command"    json:"command,omitempty"`
	NoCache  bool              `toml:"no_cache"   yaml:"no_cache"   json:"no_cache,omitempty"`
	LdFlags  map[string]string `toml:"ldflags"    yaml:"ldflags"    json:"ldflags,omitempty"`
	Binaries []Binary          `toml:"binaries"   yaml:"binaries"   json:"binaries,omitempty"`
}

// StartConfig groups start-time overrides.
type StartConfig struct {
	Command    string   `toml:"command"           yaml:"command"           json:"command,omitempty"`
	Entrypoint []string `toml:"entrypoint"        yaml:"entrypoint"        json:"entrypoint,omitempty"`
	// EntrypointScript is a path to a shell script relative to the build context.
	// When set, the script is copied into the image as /entrypoint.sh with execute
	// permission and wired as ENTRYPOINT. Start.Command becomes CMD (passed as
	// arguments to the script). Because the user's script takes over PID 1, tini
	// wrapping is bypassed — the script is responsible for signal forwarding and
	// zombie reaping.
	EntrypointScript string `toml:"entrypoint_script" yaml:"entrypoint_script" json:"entrypoint_script,omitempty"`
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

// HealthcheckOpts carries optional HEALTHCHECK timing parameters for docksmith.toml/yaml/json.
// Zero values mean "use the hardcoded default". Interval/Timeout/StartPeriod must
// match ^\d+(ms|s|m|h)$ when set. Retries must be 0..10 (0 means omit the flag).
type HealthcheckOpts struct {
	Interval    string `toml:"interval"     yaml:"interval"     json:"interval,omitempty"`
	Timeout     string `toml:"timeout"      yaml:"timeout"      json:"timeout,omitempty"`
	StartPeriod string `toml:"start_period" yaml:"start_period" json:"start_period,omitempty"`
	Retries     int    `toml:"retries"      yaml:"retries"      json:"retries,omitempty"`
}

// RuntimeCfg groups runtime-stage overrides.
// User and Healthcheck use sentinel booleans because false disables the feature.
type RuntimeCfg struct {
	Image           string          `toml:"image"             yaml:"image"             json:"image,omitempty"`
	Expose          int             `toml:"expose"            yaml:"expose"            json:"expose,omitempty"`
	ImageFamily     string          `toml:"image_family"      yaml:"image_family"      json:"image_family,omitempty"`
	HealthcheckOpts HealthcheckOpts `toml:"healthcheck_opts"  yaml:"healthcheck_opts"  json:"healthcheck_opts,omitempty"`
	User            string          `toml:"-"                 yaml:"-"                 json:"-"`
	UserSet         bool            `toml:"-"                 yaml:"-"                 json:"-"`
	Healthcheck     string          `toml:"-"                 yaml:"-"                 json:"-"`
	HCSet           bool            `toml:"-"                 yaml:"-"                 json:"-"`
}

// rawRuntimeCfg accepts bool or string for user/healthcheck during decode.
type rawRuntimeCfg struct {
	Image           string          `toml:"image"             yaml:"image"             json:"image,omitempty"`
	Expose          int             `toml:"expose"            yaml:"expose"            json:"expose,omitempty"`
	ImageFamily     string          `toml:"image_family"      yaml:"image_family"      json:"image_family,omitempty"`
	HealthcheckOpts HealthcheckOpts `toml:"healthcheck_opts"  yaml:"healthcheck_opts"  json:"healthcheck_opts,omitempty"`
	User            any             `toml:"user"              yaml:"user"              json:"user,omitempty"`
	Healthcheck     any             `toml:"healthcheck"       yaml:"healthcheck"       json:"healthcheck,omitempty"`
}

// validImageFamilies mirrors plan.validImageFamilies to avoid a circular import.
var validImageFamilies = map[string]bool{
	"alpine": true, "slim": true, "distroless": true,
}

var reDuration = regexp.MustCompile(`^\d+(ms|s|m|h)$`)

// reChown matches valid --chown values: name[:name] or uid[:gid].
var reChown = regexp.MustCompile(`^[a-zA-Z0-9_-]+(:[a-zA-Z0-9_-]+)?$|^\d+(:\d+)?$`)

// validate checks HealthcheckOpts field constraints.
func (h HealthcheckOpts) validate() error {
	for field, val := range map[string]string{
		"runtime_config.healthcheck_opts.interval":     h.Interval,
		"runtime_config.healthcheck_opts.timeout":      h.Timeout,
		"runtime_config.healthcheck_opts.start_period": h.StartPeriod,
	} {
		if val != "" && !reDuration.MatchString(val) {
			return fmt.Errorf("%s: %q must match ^\\d+(ms|s|m|h)$", field, val)
		}
	}
	if h.Retries < 0 || h.Retries > 10 {
		return fmt.Errorf("runtime_config.healthcheck_opts.retries: %d must be 0..10", h.Retries)
	}
	return nil
}

func (r rawRuntimeCfg) normalize() (RuntimeCfg, error) {
	if r.ImageFamily != "" && !validImageFamilies[r.ImageFamily] {
		return RuntimeCfg{}, fmt.Errorf("runtime_config.image_family %q is not valid; must be one of: alpine, slim, distroless", r.ImageFamily)
	}
	if err := r.HealthcheckOpts.validate(); err != nil {
		return RuntimeCfg{}, err
	}
	cfg := RuntimeCfg{Image: r.Image, Expose: r.Expose, ImageFamily: r.ImageFamily, HealthcheckOpts: r.HealthcheckOpts}
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
	Runtime        string                  `toml:"runtime"         yaml:"runtime"         json:"runtime"`
	Version        string                  `toml:"version"         yaml:"version"         json:"version,omitempty"`
	PackageManager string                  `toml:"package_manager" yaml:"package_manager" json:"package_manager,omitempty"`
	Dockerfile     string                  `toml:"dockerfile"      yaml:"dockerfile"      json:"dockerfile,omitempty"`
	ContextRoot    string                  `toml:"context_root"    yaml:"context_root"    json:"context_root,omitempty"`
	Env            map[string]string       `toml:"env"             yaml:"env"             json:"env,omitempty"`
	Build          BuildConfig             `toml:"build"           yaml:"build"           json:"build,omitempty"`
	Start          StartConfig             `toml:"start"           yaml:"start"           json:"start,omitempty"`
	Install        InstallConfig           `toml:"install"         yaml:"install"         json:"install,omitempty"`
	RuntimeConfig  rawRuntimeCfg           `toml:"runtime_config"  yaml:"runtime_config"  json:"runtime_config,omitempty"`
	Secrets        map[string]SecretConfig `toml:"secrets"         yaml:"secrets"         json:"secrets,omitempty"`
	RuntimeAssets  []AssetCopy             `toml:"runtime_assets"  yaml:"runtime_assets"  json:"runtime_assets,omitempty"`
	ExternalTools  []ExternalTool          `toml:"external_tools"  yaml:"external_tools"  json:"external_tools,omitempty"`
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
		ContextRoot:    raw.ContextRoot,
		Env:            raw.Env,
		Build:          raw.Build,
		Start:          raw.Start,
		Install:        raw.Install,
		RuntimeConfig:  rc,
		Secrets:        raw.Secrets,
		RuntimeAssets:  raw.RuntimeAssets,
		ExternalTools:  raw.ExternalTools,
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
	if err := c.validateLdFlags(); err != nil {
		return err
	}
	if err := c.validateBinaries(); err != nil {
		return err
	}
	if err := c.validateEntrypointScript(); err != nil {
		return err
	}
	if err := c.validateRuntimeAssets(); err != nil {
		return err
	}
	if err := c.validateSecrets(); err != nil {
		return err
	}
	return c.validateExternalTools()
}

var ldflagsKeyRE = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_./]*$`)

func (c *Config) validateLdFlags() error {
	for k, v := range c.Build.LdFlags {
		if !ldflagsKeyRE.MatchString(k) {
			return fmt.Errorf("build.ldflags: key %q must match ^[a-zA-Z_][a-zA-Z0-9_./]*$", k)
		}
		if strings.ContainsAny(v, "\"\n") {
			return fmt.Errorf("build.ldflags: value for key %q must not contain '\"' or newline", k)
		}
	}
	return nil
}

func (c *Config) validateBinaries() error {
	seen := make(map[string]bool, len(c.Build.Binaries))
	for i, b := range c.Build.Binaries {
		if !reBinaryName.MatchString(b.Name) {
			return fmt.Errorf("build.binaries[%d].name %q: must match ^[a-z][a-z0-9_-]*$", i, b.Name)
		}
		if b.Path == "" {
			return fmt.Errorf("build.binaries[%d] %q: path must not be empty", i, b.Name)
		}
		if containsDotDot(b.Path) {
			return fmt.Errorf("build.binaries[%d] %q: path must not contain '..'", i, b.Name)
		}
		if !strings.HasPrefix(b.Path, "./") && !strings.HasPrefix(b.Path, "/") {
			return fmt.Errorf("build.binaries[%d] %q: path must start with './' or '/'", i, b.Name)
		}
		if b.OutputName != "" && !reBinaryName.MatchString(b.OutputName) {
			return fmt.Errorf("build.binaries[%d] %q: output_name %q must match ^[a-z][a-z0-9_-]*$", i, b.Name, b.OutputName)
		}
		if err := validateArchitectures(i, b.Name, b.Architectures); err != nil {
			return err
		}
		if seen[b.Name] {
			return fmt.Errorf("build.binaries: duplicate name %q", b.Name)
		}
		seen[b.Name] = true
	}
	return nil
}

func validateArchitectures(idx int, name string, archs []string) error {
	seen := make(map[string]bool, len(archs))
	for _, arch := range archs {
		if !validCrossArchitectures[arch] {
			return fmt.Errorf("build.binaries[%d] %q: architectures: %q is not valid; must be one of: amd64, arm64, arm, 386, riscv64", idx, name, arch)
		}
		if seen[arch] {
			return fmt.Errorf("build.binaries[%d] %q: architectures: duplicate entry %q", idx, name, arch)
		}
		seen[arch] = true
	}
	return nil
}

func (c *Config) validateEntrypointScript() error {
	s := c.Start.EntrypointScript
	if s == "" {
		return nil
	}
	if filepath.IsAbs(s) {
		return fmt.Errorf("start.entrypoint_script: must be a relative path, got absolute %q", s)
	}
	if containsDotDot(s) {
		return fmt.Errorf("start.entrypoint_script: path traversal not allowed in %q", s)
	}
	return nil
}

func (c *Config) validateRuntimeAssets() error {
	for i, a := range c.RuntimeAssets {
		if a.Src == "" {
			return fmt.Errorf("runtime_assets[%d]: src must not be empty", i)
		}
		if a.Dst == "" {
			return fmt.Errorf("runtime_assets[%d]: dst must not be empty", i)
		}
		if containsDotDot(a.Src) || filepath.IsAbs(a.Src) {
			return fmt.Errorf("runtime_assets[%d]: src must be a relative path with no '..' components", i)
		}
		if !filepath.IsAbs(a.Dst) {
			return fmt.Errorf("runtime_assets[%d]: dst must be an absolute path", i)
		}
		if a.Chown != "" && !reChown.MatchString(a.Chown) {
			return fmt.Errorf("runtime_assets[%d]: chown %q must match ^[a-zA-Z0-9_-]+(:[a-zA-Z0-9_-]+)?$ or ^\\d+(:\\d+)?$", i, a.Chown)
		}
	}
	return nil
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

var (
	reToolName   = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	reSHA256     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reBinaryName = reToolName // same rule: ^[a-z][a-z0-9_-]*$
)

func (c *Config) validateExternalTools() error {
	for i, t := range c.ExternalTools {
		if !reToolName.MatchString(t.Name) {
			return fmt.Errorf("external_tools[%d].name %q: must match ^[a-z][a-z0-9_-]*$", i, t.Name)
		}
		if !strings.HasPrefix(t.URL, "https://") {
			return fmt.Errorf("external_tools[%d] %q: url must start with https://", i, t.Name)
		}
		if !reSHA256.MatchString(t.SHA256) {
			return fmt.Errorf("external_tools[%d] %q: sha256 must be 64 lowercase hex chars", i, t.Name)
		}
		if !filepath.IsAbs(t.InstallPath) {
			return fmt.Errorf("external_tools[%d] %q: install_path must be absolute", i, t.Name)
		}
		if t.Stage != "" && t.Stage != "builder" && t.Stage != "runtime" {
			return fmt.Errorf("external_tools[%d] %q: stage must be \"builder\" or \"runtime\"", i, t.Name)
		}
	}
	return nil
}

// ValidateContextRoot checks that contextRoot is an ancestor of appDir and
// contains no path traversal. Both paths must be absolute. Returns the
// app-relative subdirectory within the context root (e.g. "apps/frontend").
func ValidateContextRoot(contextRoot, appDir string) (string, error) {
	if contextRoot == "" {
		return "", nil
	}

	absRoot, err := filepath.Abs(contextRoot)
	if err != nil {
		return "", fmt.Errorf("resolve context root: %w", err)
	}
	absApp, err := filepath.Abs(appDir)
	if err != nil {
		return "", fmt.Errorf("resolve app dir: %w", err)
	}

	// Reject .. components in the raw paths before resolution.
	if containsDotDot(contextRoot) || containsDotDot(appDir) {
		return "", fmt.Errorf("path traversal in context root or app dir")
	}

	rel, err := filepath.Rel(absRoot, absApp)
	if err != nil {
		return "", fmt.Errorf("compute relative path: %w", err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("context root %q is not an ancestor of app dir %q", contextRoot, appDir)
	}
	// "." means they're the same directory — no monorepo offset needed.
	if rel == "." {
		return "", nil
	}
	return filepath.ToSlash(rel), nil
}

func containsDotDot(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func (c *Config) applyDefaults() {
	if c.RuntimeConfig.Expose == 0 {
		if p, ok := DefaultPorts[c.Runtime]; ok {
			c.RuntimeConfig.Expose = p
		}
	}
}
