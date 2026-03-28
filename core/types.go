package core

import (
	"encoding/json"
	"fmt"
)

// Framework represents a detected application framework and its build/runtime requirements.
type Framework struct {
	Name           string   `json:"name"`
	BuildCommand   string   `json:"build_command"`
	StartCommand   string   `json:"start_command"`
	Port           int      `json:"port"`
	OutputDir      string   `json:"output_dir,omitempty"`
	NodeVersion    string   `json:"node_version,omitempty"`
	PackageManager string   `json:"package_manager,omitempty"`
	PythonVersion  string   `json:"python_version,omitempty"`
	PythonPM       string   `json:"python_pm,omitempty"`
	GoVersion      string   `json:"go_version,omitempty"`
	SystemDeps     []string `json:"system_deps,omitempty"`
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
