package docksmith

// FrameworkDef is the top-level structure parsed from a YAML framework
// definition file. These files are the contract between docksmith core and
// community-contributed framework definitions.
//
// Example file layout:
//
//	frameworks/
//	  nextjs.yaml
//	  fastapi.yaml
//	  laravel.yaml
type FrameworkDef struct {
	// Name is the canonical framework identifier (e.g. "nextjs", "fastapi").
	Name string `yaml:"name"`
	// Runtime is the base language runtime (e.g. "node", "python", "go").
	Runtime string `yaml:"runtime"`
	// Priority controls detection order; higher values win ties. Default 0.
	Priority int `yaml:"priority"`
	// Detect describes how to identify this framework in a project directory.
	Detect DetectRules `yaml:"detect"`
	// Version describes how to extract the runtime version from project files.
	Version VersionConfig `yaml:"version"`
	// PackageManager describes how to detect the package manager in use.
	PackageManager PMConfig `yaml:"package_manager"`
	// Plan describes the multi-stage Docker build for this framework.
	Plan PlanDef `yaml:"plan"`
	// Defaults holds fallback command strings used in template variable expansion.
	Defaults DefaultsDef `yaml:"defaults"`
	// Tests contains fixture-based integration test cases for this definition.
	Tests []TestCase `yaml:"tests"`
}

// DetectRules combines multiple rule lists with boolean semantics:
//   - All: every rule must match (AND)
//   - Any: at least one rule must match (OR)
//   - None: no rule may match (NOT)
//
// An empty DetectRules always returns true (matches everything).
type DetectRules struct {
	All  []DetectRule `yaml:"all,omitempty"`
	Any  []DetectRule `yaml:"any,omitempty"`
	None []DetectRule `yaml:"none,omitempty"`
}

// DetectRule is a single predicate evaluated against a project directory.
// At most one kind field should be set per rule; if multiple are set, they
// are evaluated in the order: file, dir, contains, regex, dependency, json, toml.
type DetectRule struct {
	// File checks that a file (or glob) exists in the project directory.
	File string `yaml:"file,omitempty"`
	// Dir checks that a subdirectory exists in the project directory.
	Dir string `yaml:"dir,omitempty"`
	// Contains requires File to be set; true when the file contains the substring.
	Contains string `yaml:"contains,omitempty"`
	// Regex requires File to be set; true when the file matches the regular expression.
	Regex string `yaml:"regex,omitempty"`
	// Dependency checks whether the named package appears in the project's
	// dependency manifest (package.json, requirements.txt, go.mod, etc.).
	Dependency string `yaml:"dependency,omitempty"`
	// JSON is the path to a JSON file; used together with Path for value extraction.
	JSON string `yaml:"json,omitempty"`
	// TOML is the path to a TOML file; used together with Path for value extraction.
	TOML string `yaml:"toml,omitempty"`
	// Path is a dot-separated key path for JSON/TOML extraction (e.g. "scripts.build").
	Path string `yaml:"path,omitempty"`
}

// VersionConfig describes one or more strategies for detecting the runtime
// version from project files.  Sources are tried in order; the first non-empty
// result wins.
type VersionConfig struct {
	Sources []VersionSource `yaml:"sources"`
	// Default is used when no source yields a version.
	Default string `yaml:"default"`
}

// VersionSource describes one method for extracting a version string.
type VersionSource struct {
	// File is a plain text file whose trimmed content is the version (e.g. ".node-version").
	File string `yaml:"file,omitempty"`
	// JSON is the path to a JSON file; used together with Path.
	JSON string `yaml:"json,omitempty"`
	// TOML is the path to a TOML file; used together with Path.
	TOML string `yaml:"toml,omitempty"`
	// Path is a dot-separated key path for JSON/TOML extraction.
	Path string `yaml:"path,omitempty"`
}

// PMConfig describes strategies for detecting which package manager is in use.
type PMConfig struct {
	Sources []PMSource `yaml:"sources"`
	// Default is used when no source yields a package manager name.
	Default string `yaml:"default"`
}

// PMSource describes one method for detecting the package manager.
type PMSource struct {
	// JSON is the path to a JSON file; used together with Path.
	JSON string `yaml:"json,omitempty"`
	// Path is a dot-separated key path for JSON extraction.
	Path string `yaml:"path,omitempty"`
	// File is a lockfile whose mere presence identifies the package manager.
	// When set, Value must also be set.
	File string `yaml:"file,omitempty"`
	// Value is the package manager name returned when File is found.
	Value string `yaml:"value,omitempty"`
}

// PlanDef describes the complete multi-stage Docker build plan for a framework.
type PlanDef struct {
	// Port is the default container port exposed by this framework.
	Port int `yaml:"port"`
	// Stages lists the build stages in order (e.g. deps, build, runtime).
	Stages []StageDef `yaml:"stages"`
}

// StageDef is one Docker build stage.
type StageDef struct {
	// Name is the stage identifier referenced by later stages.
	Name string `yaml:"name"`
	// Base is a runtime key passed to ResolveDockerTag (e.g. "node", "python").
	// Mutually exclusive with From.
	Base string `yaml:"base,omitempty"`
	// From is a literal Docker image reference or a prior stage name.
	// Mutually exclusive with Base.
	From string `yaml:"from,omitempty"`
	// Steps are the build instructions for this stage.
	Steps []StepDef `yaml:"steps"`
}

// StepDef is one Docker build instruction within a stage.
// Exactly one action field should be set per step.
type StepDef struct {
	// Workdir sets the working directory (WORKDIR).
	Workdir string `yaml:"workdir,omitempty"`
	// Copy is a list of src... dst arguments (COPY).
	Copy []string `yaml:"copy,omitempty"`
	// CopyFrom copies a path from a prior build stage (COPY --from).
	CopyFrom *CopyFromDef `yaml:"copy_from,omitempty"`
	// Run is a shell command (RUN).
	Run string `yaml:"run,omitempty"`
	// Cache is a BuildKit cache-mount target path for the Run step.
	Cache string `yaml:"cache,omitempty"`
	// Env is a map of environment variable key-value pairs (ENV).
	Env map[string]string `yaml:"env,omitempty"`
	// Cmd is the default container command as a string slice (CMD).
	Cmd []string `yaml:"cmd,omitempty"`
	// Expose is the port to expose as a string (EXPOSE).
	Expose string `yaml:"expose,omitempty"`
}

// CopyFromDef describes a cross-stage COPY --from instruction.
type CopyFromDef struct {
	// Stage is the name of the prior build stage to copy from.
	Stage string `yaml:"stage"`
	// Src is the source path inside the named stage.
	Src string `yaml:"src"`
	// Dst is the destination path in the current stage.
	Dst string `yaml:"dst"`
}

// DefaultsDef holds fallback command strings used during template expansion.
// Install is keyed by package manager name (e.g. "npm", "pip", "cargo").
type DefaultsDef struct {
	// Install maps package manager name to its install command.
	Install map[string]string `yaml:"install,omitempty"`
	// Build is the default build command (used when no override is provided).
	Build string `yaml:"build,omitempty"`
	// Start is the default start/run command for the container.
	Start string `yaml:"start,omitempty"`
}

// TestCase is a fixture-based test embedded in the framework definition.
// It describes a minimal directory layout and the expected detection outcome.
type TestCase struct {
	// Name is a human-readable description of the test scenario.
	Name string `yaml:"name"`
	// Fixture maps relative file paths to their contents.
	// An empty string value means the file should exist but be empty.
	Fixture map[string]string `yaml:"fixture"`
	// Expect describes the expected detection result.
	Expect TestExpect `yaml:"expect"`
}

// TestExpect describes what the detector should produce for a given fixture.
type TestExpect struct {
	// Detected is true when the framework should be identified.
	Detected bool `yaml:"detected"`
	// Framework is the expected Framework.Name value (optional assertion).
	Framework string `yaml:"framework,omitempty"`
	// Port is the expected exposed port (optional assertion).
	Port int `yaml:"port,omitempty"`
}
