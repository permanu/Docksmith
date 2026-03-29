package docksmith

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/emit"
	"github.com/permanu/docksmith/plan"
	"github.com/permanu/docksmith/registry"
	"github.com/permanu/docksmith/yamldef"
)

// ---------------------------------------------------------------------------
// Type aliases — re-export core types for backward compatibility
// ---------------------------------------------------------------------------

type Framework = core.Framework
type DetectorFunc = core.DetectorFunc

// Plan types
type StepType = core.StepType

const (
	StepWorkdir     = core.StepWorkdir
	StepCopy        = core.StepCopy
	StepCopyFrom    = core.StepCopyFrom
	StepRun         = core.StepRun
	StepEnv         = core.StepEnv
	StepArg         = core.StepArg
	StepExpose      = core.StepExpose
	StepCmd         = core.StepCmd
	StepEntrypoint  = core.StepEntrypoint
	StepUser        = core.StepUser
	StepHealthcheck = core.StepHealthcheck
)

type BuildPlan = core.BuildPlan
type Stage = core.Stage
type Step = core.Step
type CacheMount = core.CacheMount
type SecretMount = core.SecretMount
type CopyFrom = core.CopyFrom

// Config types
type Config = config.Config
type BuildConfig = config.BuildConfig
type StartConfig = config.StartConfig
type InstallConfig = config.InstallConfig
type RuntimeCfg = config.RuntimeCfg

// Detect types
type NamedDetector = detect.NamedDetector
type DetectOptions = detect.DetectOptions

// Plan option type
type PlanOption = plan.PlanOption

// YAML definition types
type FrameworkDef = yamldef.FrameworkDef
type DetectRules = yamldef.DetectRules
type DetectRule = yamldef.DetectRule
type VersionConfig = yamldef.VersionConfig
type VersionSource = yamldef.VersionSource
type PMConfig = yamldef.PMConfig
type PMSource = yamldef.PMSource
type PlanDef = yamldef.PlanDef
type StageDef = yamldef.StageDef
type StepDef = yamldef.StepDef
type CopyFromDef = yamldef.CopyFromDef
type DefaultsDef = yamldef.DefaultsDef
type TestCase = yamldef.TestCase
type TestExpect = yamldef.TestExpect
type TestResult = yamldef.TestResult

// Registry types
const DefaultRegistryURL = registry.DefaultRegistryURL

type RegistryIndex = registry.Index
type RegistryEntry = registry.Entry

// ---------------------------------------------------------------------------
// Sentinel errors — re-export from core
// ---------------------------------------------------------------------------

var (
	ErrNotDetected  = core.ErrNotDetected
	ErrInvalidConfig = core.ErrInvalidConfig
	ErrInvalidPlan  = core.ErrInvalidPlan
)

// ---------------------------------------------------------------------------
// Function aliases — re-export from subpackages
// ---------------------------------------------------------------------------

var FrameworkFromJSON = core.FrameworkFromJSON

// Plan option constructors
var (
	WithUser               = plan.WithUser
	WithHealthcheck        = plan.WithHealthcheck
	WithHealthcheckDisabled = plan.WithHealthcheckDisabled
	WithRuntimeImage       = plan.WithRuntimeImage
	WithBaseImage          = plan.WithBaseImage
	WithEntrypoint         = plan.WithEntrypoint
	WithExtraEnv           = plan.WithExtraEnv
	WithExpose             = plan.WithExpose
	WithInstallCommand     = plan.WithInstallCommand
	WithBuildCommand       = plan.WithBuildCommand
	WithStartCommand       = plan.WithStartCommand
	WithSystemDeps         = plan.WithSystemDeps
	WithBuildCacheDisabled = plan.WithBuildCacheDisabled
)

var ResolveDockerTag = plan.ResolveDockerTag
var FrameworkDefaults = plan.FrameworkDefaults
var BuildkitCacheArgs = plan.BuildkitCacheArgs
var CacheDir = plan.CacheDir

// Registry function aliases
var FetchRegistryIndex = registry.FetchIndex
var SearchRegistry = registry.Search
var InstallFramework = registry.InstallFramework

// Package-manager helpers re-exported for plan code and backward compatibility.
var (
	pmInstallCommand   = detect.PMInstallCommand
	pmRunBuild         = detect.PMRunBuild
	pmRunStart         = detect.PMRunStart
	pmRunInstall       = detect.PMRunInstall
	nodeVersionAtLeast = detect.NodeVersionAtLeast
)

// Emit helper re-exports for backward compatibility.
var sanitizeDockerfileArg = emit.SanitizeDockerfileArg
var shellSplit = emit.ShellSplit
var jsonArray = emit.JSONArray
var pmCopyLockfiles = emit.PMCopyLockfiles

// ---------------------------------------------------------------------------
// Facade functions — main pipeline
// ---------------------------------------------------------------------------

// GenerateDockerfile runs Plan + EmitDockerfile for the given framework.
// Returns ("", nil) when fw.Name == "dockerfile" (user already has one).
func GenerateDockerfile(fw *Framework, opts ...PlanOption) (string, error) {
	if fw == nil || fw.Name == "dockerfile" {
		return "", nil
	}
	p, err := Plan(fw, opts...)
	if err != nil {
		return "", fmt.Errorf("generate dockerfile: %w", err)
	}
	return EmitDockerfile(p), nil
}

// Build runs the full pipeline for dir: detect -> plan -> emit.
func Build(dir string, opts ...PlanOption) (string, *Framework, error) {
	return BuildWithOptions(dir, DetectOptions{}, opts...)
}

// BuildWithOptions runs the pipeline with custom detection options.
func BuildWithOptions(dir string, detectOpts DetectOptions, planOpts ...PlanOption) (string, *Framework, error) {
	fw, err := DetectWithOptions(dir, detectOpts)
	if err != nil {
		return "", nil, fmt.Errorf("build: %w", err)
	}
	if fw.Name == "dockerfile" {
		return "", fw, nil
	}
	p, err := Plan(fw, planOpts...)
	if err != nil {
		return "", fw, err
	}
	return EmitDockerfile(p), fw, nil
}

// ---------------------------------------------------------------------------
// Detect facade
// ---------------------------------------------------------------------------

// Detect analyzes dir and returns the detected framework.
func Detect(dir string) (*Framework, error) {
	return detect.Detect(dir)
}

// DetectWithOptions runs detection with custom options.
func DetectWithOptions(dir string, opts DetectOptions) (*Framework, error) {
	return detect.DetectWithOptions(dir, opts)
}

// RegisterDetector prepends d to the registry.
func RegisterDetector(name string, d DetectorFunc) {
	detect.RegisterDetector(name, d)
}

// RegisterDetectorBefore inserts d immediately before the named detector.
func RegisterDetectorBefore(before, name string, d DetectorFunc) {
	detect.RegisterDetectorBefore(before, name, d)
}

// ConfigToFramework converts a Config to a Framework for Dockerfile generation.
func ConfigToFramework(c *config.Config) *Framework {
	return detect.ConfigToFramework(c)
}

// ---------------------------------------------------------------------------
// Plan facade
// ---------------------------------------------------------------------------

// Plan converts a detected Framework into a BuildPlan.
func Plan(fw *Framework, opts ...PlanOption) (*BuildPlan, error) {
	return plan.Plan(fw, opts...)
}

// ---------------------------------------------------------------------------
// Emit facade
// ---------------------------------------------------------------------------

// EmitDockerfile serializes a BuildPlan into a Dockerfile string.
func EmitDockerfile(p *BuildPlan) string {
	return emit.EmitDockerfile(p)
}

// GenerateDockerignore returns .dockerignore file content tailored to the framework.
func GenerateDockerignore(fw *Framework) string {
	return emit.GenerateDockerignore(fw)
}

// ---------------------------------------------------------------------------
// Config facade
// ---------------------------------------------------------------------------

// LoadConfig reads the first matching config file from dir.
// Returns (nil, nil) if no config file exists.
func LoadConfig(dir string) (*Config, error) {
	return config.Load(dir)
}

// LoadPlanOptions reads the config from dir and converts it to a PlanOption slice.
// Returns nil (not an error) when no config file exists.
func LoadPlanOptions(dir string) ([]PlanOption, error) {
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return ConfigToPlanOptions(cfg)
}

// ConfigToPlanOptions converts Config fields into a PlanOption slice.
func ConfigToPlanOptions(c *Config) ([]PlanOption, error) {
	var opts []PlanOption

	if c.Build.Command != "" {
		opts = append(opts, WithBuildCommand(c.Build.Command))
	}
	if c.Build.NoCache {
		opts = append(opts, WithBuildCacheDisabled())
	}
	if c.Start.Command != "" {
		opts = append(opts, WithStartCommand(c.Start.Command))
	}
	if len(c.Start.Entrypoint) > 0 {
		opts = append(opts, WithEntrypoint(c.Start.Entrypoint...))
	}
	if c.Install.Command != "" {
		opts = append(opts, WithInstallCommand(c.Install.Command))
	}
	if len(c.Install.SystemDeps) > 0 {
		opts = append(opts, WithSystemDeps(c.Install.SystemDeps...))
	}
	if len(c.Env) > 0 {
		opts = append(opts, WithExtraEnv(c.Env))
	}
	if c.RuntimeConfig.Image != "" {
		opts = append(opts, WithRuntimeImage(c.RuntimeConfig.Image))
	}
	if c.RuntimeConfig.Expose > 0 {
		opts = append(opts, WithExpose(c.RuntimeConfig.Expose))
	}
	if c.RuntimeConfig.UserSet {
		opts = append(opts, WithUser(c.RuntimeConfig.User))
	}
	if c.RuntimeConfig.HCSet {
		if c.RuntimeConfig.Healthcheck == "" {
			opts = append(opts, WithHealthcheckDisabled())
		} else {
			opts = append(opts, WithHealthcheck(c.RuntimeConfig.Healthcheck))
		}
	}

	return opts, nil
}

// ---------------------------------------------------------------------------
// YAML framework detect wrappers
// ---------------------------------------------------------------------------

// evalDetectRules delegates to yamldef.EvalDetectRules.
func evalDetectRules(dir string, rules DetectRules) bool {
	return yamldef.EvalDetectRules(dir, rules)
}

// evalRule delegates to yamldef.EvalRule.
func evalRule(dir string, rule DetectRule) bool {
	return yamldef.EvalRule(dir, rule)
}

// isYAMLFile delegates to yamldef.IsYAMLFile.
func isYAMLFile(name string) bool {
	return yamldef.IsYAMLFile(name)
}

// extractDotPath delegates to yamldef.ExtractDotPath.
func extractDotPath(root any, dotPath string) any {
	return yamldef.ExtractDotPath(root, dotPath)
}

// fileMatchesRegex delegates to yamldef for regex matching against file content.
func fileMatchesRegex(path, pattern string) bool {
	if len(pattern) > 1024 {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	data, err := yamldef.ReadFileLimited(path)
	if err != nil {
		return false
	}
	return re.Match(data)
}

// ---------------------------------------------------------------------------
// YAML framework loader wrappers
// ---------------------------------------------------------------------------

// LoadFrameworkDefs loads YAML framework definitions from dir.
func LoadFrameworkDefs(dir string) ([]*FrameworkDef, error) {
	return yamldef.LoadFrameworkDefs(dir)
}

// LoadAndRegisterFrameworks loads YAML defs from each dir (in order) and
// registers them as detectors. Resolution order after registration:
//  1. .docksmith/frameworks/ in project dir (call with project path first)
//  2. ~/.docksmith/frameworks/ (call with home path second)
//  3. Built-in Go detectors (already registered at init time)
//
// YAML detectors are prepended, so the last dir passed has lowest YAML priority.
// Dirs that don't exist are silently skipped.
func LoadAndRegisterFrameworks(dirs ...string) error {
	var errs []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		defs, err := yamldef.LoadFrameworkDefs(dir)
		if err != nil {
			errs = append(errs, err.Error())
		}
		for _, def := range defs {
			d := def // capture
			RegisterDetector("yaml:"+d.Name, func(dir string) *Framework {
				return evalDefAgainstDir(d, dir)
			})
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// evalDefAgainstDir runs the detect rules for def against dir.
func evalDefAgainstDir(def *FrameworkDef, dir string) *Framework {
	name, port, matched := yamldef.EvalDefAgainstDir(def, dir)
	if !matched {
		return nil
	}
	return &Framework{
		Name: name,
		Port: port,
	}
}

// BuildPlanFromDef converts a FrameworkDef into a BuildPlan.
// It resolves Base fields via ResolveDockerTag and converts StepDefs to Steps.
func BuildPlanFromDef(def *FrameworkDef, fw *Framework) (*BuildPlan, error) {
	if len(def.Plan.Stages) == 0 {
		return nil, fmt.Errorf("%w: yaml def %q has no stages", ErrInvalidPlan, def.Name)
	}

	port := def.Plan.Port
	if fw != nil && fw.Port != 0 {
		port = fw.Port
	}

	stages := make([]Stage, 0, len(def.Plan.Stages))
	for _, sd := range def.Plan.Stages {
		s, err := buildStageFromDef(sd, fw)
		if err != nil {
			return nil, fmt.Errorf("stage %q: %w", sd.Name, err)
		}
		stages = append(stages, s)
	}

	return &BuildPlan{
		Framework: def.Name,
		Stages:    stages,
		Expose:    port,
	}, nil
}

func buildStageFromDef(sd StageDef, fw *Framework) (Stage, error) {
	from := sd.From
	if from == "" && sd.Base != "" {
		version := versionForBase(sd.Base, fw)
		from = ResolveDockerTag(sd.Base, version)
	}
	if from == "" {
		return Stage{}, fmt.Errorf("stage %q has no from or base", sd.Name)
	}

	steps := make([]Step, 0, len(sd.Steps))
	for _, stepDef := range sd.Steps {
		step, err := buildStepFromDef(stepDef)
		if err != nil {
			return Stage{}, err
		}
		steps = append(steps, step)
	}

	return Stage{Name: sd.Name, From: from, Steps: steps}, nil
}

func buildStepFromDef(sd StepDef) (Step, error) {
	switch {
	case sd.Workdir != "":
		return Step{Type: StepWorkdir, Args: []string{sd.Workdir}}, nil
	case len(sd.Copy) > 0:
		return Step{Type: StepCopy, Args: sd.Copy}, nil
	case sd.CopyFrom != nil:
		return Step{
			Type: StepCopyFrom,
			CopyFrom: &CopyFrom{
				Stage: sd.CopyFrom.Stage,
				Src:   sd.CopyFrom.Src,
				Dst:   sd.CopyFrom.Dst,
			},
		}, nil
	case sd.Run != "":
		s := Step{Type: StepRun, Args: []string{sd.Run}}
		if sd.Cache != "" {
			s.CacheMount = &CacheMount{Target: sd.Cache}
		}
		return s, nil
	case len(sd.Env) > 0:
		keys := yamldef.SortedKeys(sd.Env)
		for _, k := range keys {
			return Step{Type: StepEnv, Args: []string{k, sd.Env[k]}}, nil
		}
	case sd.Expose != "":
		return Step{Type: StepExpose, Args: []string{sd.Expose}}, nil
	case len(sd.Cmd) > 0:
		return Step{Type: StepCmd, Args: sd.Cmd}, nil
	}
	return Step{}, fmt.Errorf("empty step definition")
}

func versionForBase(base string, fw *Framework) string {
	if fw == nil {
		return ""
	}
	switch base {
	case "node":
		return fw.NodeVersion
	case "python":
		return fw.PythonVersion
	case "go":
		return fw.GoVersion
	}
	return ""
}

// ---------------------------------------------------------------------------
// YAML framework plan wrappers
// ---------------------------------------------------------------------------

// buildPlanFromDef converts a FrameworkDef into a BuildPlan using context
// detected from dir. Template variables are substituted with strings.ReplaceAll;
// unknown tokens are left in-place (never an error). Supported variables:
// {{runtime}}, {{version}}, {{pm}}, {{lockfile}}, {{install_command}},
// {{build_command}}, {{start_command}}, {{port}}.
// BuildPlanFromDefDir builds a BuildPlan from a FrameworkDef by resolving
// version and package manager from the given project directory.
func BuildPlanFromDefDir(def *FrameworkDef, dir string) (*BuildPlan, error) {
	return buildPlanFromDefDir(def, dir)
}

func buildPlanFromDefDir(def *FrameworkDef, dir string) (*BuildPlan, error) {
	if def == nil {
		return nil, fmt.Errorf("buildPlanFromDef: def must not be nil")
	}

	version := yamldef.ResolveVersion(def, dir)
	pm := yamldef.ResolvePM(def, dir)

	vars := map[string]string{
		"{{runtime}}":         def.Runtime,
		"{{version}}":         version,
		"{{pm}}":              pm,
		"{{lockfile}}":        yamldef.PMLockfileName(pm),
		"{{install_command}}": yamldef.ResolveInstallCommand(def, pm),
		"{{build_command}}":   def.Defaults.Build,
		"{{start_command}}":   def.Defaults.Start,
		"{{port}}":            fmt.Sprintf("%d", def.Plan.Port),
	}

	stages := make([]Stage, 0, len(def.Plan.Stages))
	for _, sd := range def.Plan.Stages {
		s, err := resolveStage(sd, def, vars, version)
		if err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}

	return &BuildPlan{
		Framework: def.Name,
		Expose:    def.Plan.Port,
		Stages:    stages,
	}, nil
}

func resolveStage(sd StageDef, def *FrameworkDef, vars map[string]string, version string) (Stage, error) {
	from, err := resolveFrom(sd, def, version)
	if err != nil {
		return Stage{}, err
	}

	steps := make([]Step, 0, len(sd.Steps))
	for _, stepDef := range sd.Steps {
		step, err := resolveStep(stepDef, vars)
		if err != nil {
			return Stage{}, err
		}
		steps = append(steps, step)
	}

	return Stage{
		Name:  sd.Name,
		From:  from,
		Steps: steps,
	}, nil
}

func resolveFrom(sd StageDef, def *FrameworkDef, version string) (string, error) {
	if sd.Base != "" {
		return ResolveDockerTag(sd.Base, version), nil
	}
	if sd.From != "" {
		return sd.From, nil
	}
	return "", fmt.Errorf("stage %q: either base or from must be set", sd.Name)
}

func resolveStep(sd StepDef, vars map[string]string) (Step, error) {
	if sd.Workdir != "" {
		return Step{
			Type: StepWorkdir,
			Args: []string{yamldef.Sub(sd.Workdir, vars)},
		}, nil
	}
	if sd.CopyFrom != nil {
		return Step{
			Type: StepCopyFrom,
			CopyFrom: &CopyFrom{
				Stage: yamldef.Sub(sd.CopyFrom.Stage, vars),
				Src:   yamldef.Sub(sd.CopyFrom.Src, vars),
				Dst:   yamldef.Sub(sd.CopyFrom.Dst, vars),
			},
			Link: true,
		}, nil
	}
	if len(sd.Copy) > 0 {
		args := make([]string, len(sd.Copy))
		for i, s := range sd.Copy {
			args[i] = yamldef.Sub(s, vars)
		}
		return Step{Type: StepCopy, Args: args}, nil
	}
	if sd.Run != "" {
		step := Step{
			Type: StepRun,
			Args: []string{yamldef.Sub(sd.Run, vars)},
		}
		if sd.Cache != "" {
			step.CacheMount = &CacheMount{Target: yamldef.Sub(sd.Cache, vars)}
		}
		return step, nil
	}
	if len(sd.Env) > 0 {
		keys := yamldef.SortedKeys(sd.Env)
		args := make([]string, 0, len(keys)*2)
		for _, k := range keys {
			args = append(args, yamldef.Sub(k, vars), yamldef.Sub(sd.Env[k], vars))
		}
		return Step{Type: StepEnv, Args: args}, nil
	}
	if len(sd.Cmd) > 0 {
		args := make([]string, len(sd.Cmd))
		for i, s := range sd.Cmd {
			args[i] = yamldef.Sub(s, vars)
		}
		return Step{Type: StepCmd, Args: args}, nil
	}
	if sd.Expose != "" {
		return Step{
			Type: StepExpose,
			Args: []string{yamldef.Sub(sd.Expose, vars)},
		}, nil
	}
	return Step{Type: StepRun, Args: []string{""}}, nil
}

func sortedKeys(m map[string]string) []string {
	return yamldef.SortedKeys(m)
}

func pmLockfileName(pm string) string {
	return yamldef.PMLockfileName(pm)
}

// ---------------------------------------------------------------------------
// YAML framework test runner wrappers
// ---------------------------------------------------------------------------

// RunFrameworkTests delegates to yamldef.RunFrameworkTests.
func RunFrameworkTests(yamlPath string) ([]TestResult, error) {
	return yamldef.RunFrameworkTests(yamlPath)
}

// RunFrameworkDefTests delegates to yamldef.RunFrameworkDefTests.
func RunFrameworkDefTests(def *FrameworkDef) error {
	return yamldef.RunFrameworkDefTests(def)
}

func runTestCaseForDef(def *FrameworkDef, tc TestCase) TestResult {
	return yamldef.RunTestCaseForDef(def, tc)
}

func frameworkName(fw *Framework) string {
	if fw == nil {
		return ""
	}
	return fw.Name
}

// runTestCase runs a single test case using the global detector registry.
func runTestCase(tc TestCase) TestResult {
	dir, err := os.MkdirTemp("", "docksmith-test-*")
	if err != nil {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mktemp: %v", err)}
	}
	defer os.RemoveAll(dir)

	for relPath, content := range tc.Fixture {
		full, pathErr := yamldef.ContainedPath(dir, relPath)
		if pathErr != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("unsafe fixture path %q: %v", relPath, pathErr)}
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("mkdir %s: %v", relPath, err)}
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("write %s: %v", relPath, err)}
		}
	}

	fw, detectErr := Detect(dir)
	if detectErr != nil && !errors.Is(detectErr, ErrNotDetected) {
		return TestResult{Name: tc.Name, Passed: false, Reason: fmt.Sprintf("detect: %v", detectErr)}
	}
	detected := fw != nil && fw.Name != "static"

	if detected != tc.Expect.Detected {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("detected=%v, want %v (framework=%q)", detected, tc.Expect.Detected, fw.Name),
		}
	}

	if tc.Expect.Framework != "" && fw.Name != tc.Expect.Framework {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("framework=%q, want %q", fw.Name, tc.Expect.Framework),
		}
	}

	if tc.Expect.Port != 0 && fw.Port != tc.Expect.Port {
		return TestResult{
			Name:   tc.Name,
			Passed: false,
			Reason: fmt.Sprintf("port=%d, want %d", fw.Port, tc.Expect.Port),
		}
	}

	return TestResult{Name: tc.Name, Passed: true}
}

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

const maxFileReadBytes = 10 << 20 // 10 MB

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasFile(dir, name string) bool {
	if name == "" {
		return false
	}
	return fileExists(filepath.Join(dir, name))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func fileContains(path, substr string) bool {
	data, err := readFileLimited(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// parseVersionString cleans a raw version string from .nvmrc, .node-version,
// or semver constraint fields. Handles ranges like ">=3.9,<4" by taking the
// first constraint. Returns "" for aliases (lts/*, stable, node).
func parseVersionString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)
	if s == "" || s == "lts/*" || s == "stable" || s == "node" {
		return ""
	}
	hasComma := strings.Contains(s, ",")
	hasOperator := len(s) > 0 && (s[0] == '>' || s[0] == '<' || s[0] == '=' || s[0] == '~' || s[0] == '^')
	if !hasComma && !hasOperator {
		return s
	}
	if hasComma {
		s = strings.TrimSpace(s[:strings.Index(s, ",")])
	}
	s = strings.TrimLeft(s, "><=~^")
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ".", 3)
	major := parts[0]
	if major == "" || !isDigits(major) {
		return ""
	}
	if len(parts) < 2 {
		return major
	}
	minor := strings.TrimSuffix(parts[1], "x")
	if minor == "" || !isDigits(minor) || minor == "0" {
		return major
	}
	return major + "." + minor
}

// extractMajorVersion pulls a usable version from semver constraints.
func extractMajorVersion(constraint string) string {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return ""
	}
	constraint = strings.TrimLeft(constraint, "><=~^")
	constraint = strings.TrimSpace(constraint)
	parts := strings.SplitN(constraint, ".", 3)
	major := strings.TrimSpace(parts[0])
	if major == "" || !isDigits(major) {
		return ""
	}
	if len(parts) < 2 {
		return major
	}
	minor := strings.TrimSuffix(strings.TrimSpace(parts[1]), "x")
	if minor == "" || !isDigits(minor) || minor == "0" {
		return major
	}
	return major + "." + minor
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// readFileLimited reads up to maxFileReadBytes from path.
func readFileLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lr := io.LimitReader(f, maxFileReadBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxFileReadBytes {
		return nil, fmt.Errorf("file %s exceeds %d byte limit", filepath.Base(path), maxFileReadBytes)
	}
	return data, nil
}

// containedPath joins base and rel, then verifies the result is under base.
// Prevents path traversal via "../" or absolute paths in rel.
func containedPath(base, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	rel = strings.ReplaceAll(rel, "\x00", "")
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute path %q not allowed", rel)
	}
	joined := filepath.Join(base, rel)
	cleaned := filepath.Clean(joined)
	baseClean := filepath.Clean(base) + string(filepath.Separator)
	if !strings.HasPrefix(cleaned+string(filepath.Separator), baseClean) && cleaned != filepath.Clean(base) {
		return "", fmt.Errorf("path %q escapes base directory", rel)
	}
	return cleaned, nil
}
