// Package core defines the shared types used across all docksmith layers:
// Framework (detection result), BuildPlan (abstract build steps), Stage,
// Step, CacheMount, and SecretMount.
package core

import (
	"encoding/json"
	"fmt"
)

// Framework holds the detection result for a project directory.
// Detect populates it; Plan consumes it to build a multi-stage Dockerfile.
// Zero-value fields are ignored during planning.
type Framework struct {
	Name           string   `json:"name"`
	BuildCommand   string   `json:"build_command"`
	StartCommand   string   `json:"start_command"`
	Port           int      `json:"port"`
	OutputDir      string   `json:"output_dir,omitempty"` // static asset dir (e.g. "dist", ".next"); empty for server frameworks
	NodeVersion    string   `json:"node_version,omitempty"`
	PackageManager string   `json:"package_manager,omitempty"` // npm, pnpm, yarn, bun — drives install commands and lockfile selection
	PythonVersion  string   `json:"python_version,omitempty"`
	PythonPM       string   `json:"python_pm,omitempty"` // pip, poetry, uv, pdm, pipenv — distinct from PackageManager (JS-only)
	GoVersion      string   `json:"go_version,omitempty"`
	SystemDeps     []string `json:"system_deps,omitempty"` // OS packages needed at build time (e.g. libpq-dev for psycopg2)
	PHPVersion     string   `json:"php_version,omitempty"`
	DotnetVersion  string   `json:"dotnet_version,omitempty"`
	JavaVersion    string   `json:"java_version,omitempty"`
	DenoVersion    string   `json:"deno_version,omitempty"`
	BunVersion     string   `json:"bun_version,omitempty"`
}

// DetectorFunc checks a directory and returns a Framework if detected, nil otherwise.
type DetectorFunc func(dir string) *Framework

// ToJSON serializes a Framework to JSON for transport or caching.
func (f *Framework) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}

// FrameworkFromJSON deserializes a Framework from JSON.
func FrameworkFromJSON(data []byte) (*Framework, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty framework data")
	}
	var fw Framework
	if err := json.Unmarshal(data, &fw); err != nil {
		return nil, fmt.Errorf("parse framework: %w", err)
	}
	return &fw, nil
}
