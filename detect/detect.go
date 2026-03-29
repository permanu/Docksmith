// Package detect identifies application frameworks from project files.
// It scans for package.json, go.mod, requirements.txt, Cargo.toml, and
// similar markers to determine the framework, runtime version, and
// package manager. 45 detectors are registered at init time; custom
// detectors can be added via RegisterDetector.
package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
)

// staticFileExtensions are file extensions that indicate actual web-servable content.
var staticFileExtensions = map[string]bool{
	".html": true, ".htm": true, ".css": true, ".js": true,
	".svg": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true, ".ico": true, ".json": true,
	".xml": true, ".woff": true, ".woff2": true, ".ttf": true,
}

// hasServableContent checks whether a directory contains files that look like
// a static website (HTML, CSS, JS, images). Returns false for empty dirs or
// dirs with only non-web files — these should not silently deploy as static.
func hasServableContent(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if staticFileExtensions[ext] {
			return true
		}
	}
	return false
}

// NamedDetector pairs a name with a detection function for registry ordering.
type NamedDetector struct {
	Name   string
	Detect core.DetectorFunc
}

// DetectOptions configures detection behavior.
type DetectOptions struct {
	// ConfigFileNames lists filenames to treat as user config (e.g. "docksmith.toml").
	ConfigFileNames []string
}

// detectors is the ordered registry. Individual runtime detectors
// append to this at init time. Dockerfile and static fallback are handled inline.
var detectors []NamedDetector

// Detect analyzes dir and returns the detected framework.
// Returns a static-site framework as fallback if nothing matches.
func Detect(dir string) (*core.Framework, error) {
	return DetectWithOptions(dir, DetectOptions{})
}

// DetectWithOptions runs detection with custom options.
// Returns an error when a config file exists but is invalid — this prevents
// silent fallthrough to auto-detection when the user intended a specific config.
func DetectWithOptions(dir string, opts DetectOptions) (*core.Framework, error) {
	// User config — highest priority of all.
	fw, cfgErr := loadConfigFramework(dir, opts)
	if cfgErr != nil {
		return nil, cfgErr
	}
	if fw != nil {
		return fw, nil
	}

	// User Dockerfile — second priority before auto-detection.
	if hasFile(dir, "Dockerfile") {
		return &core.Framework{Name: "dockerfile", Port: 8080}, nil
	}

	for _, nd := range detectors {
		if fw := nd.Detect(dir); fw != nil {
			return fw, nil
		}
	}

	// Only fall back to static if the directory has actual web-servable content.
	// An empty dir or dir with only non-web files should error, not silently
	// deploy nothing behind nginx.
	if hasServableContent(dir) {
		return &core.Framework{Name: "static", Port: 80, OutputDir: "."}, nil
	}

	return nil, fmt.Errorf("%w: no supported framework found in %s — add a Dockerfile or docksmith.toml to configure manually", core.ErrNotDetected, dir)
}

// loadConfigFramework loads the user config and converts it to a Framework.
// Returns (nil, nil) when no config file exists.
// Returns (nil, err) when a config file exists but is invalid.
func loadConfigFramework(dir string, opts DetectOptions) (*core.Framework, error) {
	names := opts.ConfigFileNames
	if len(names) == 0 {
		names = config.DefaultFileNames
	}
	cfg, err := config.LoadWithNames(dir, names)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return ConfigToFramework(cfg), nil
}

// RegisterDetector prepends d to the registry, giving it the highest priority
// among registered detectors.
func RegisterDetector(name string, d core.DetectorFunc) {
	detectors = append([]NamedDetector{{name, d}}, detectors...)
}

// RegisterDetectorBefore inserts d immediately before the named detector.
// If before is not found, d is prepended.
func RegisterDetectorBefore(before, name string, d core.DetectorFunc) {
	for i, nd := range detectors {
		if nd.Name == before {
			updated := make([]NamedDetector, 0, len(detectors)+1)
			updated = append(updated, detectors[:i]...)
			updated = append(updated, NamedDetector{name, d})
			updated = append(updated, detectors[i:]...)
			detectors = updated
			return
		}
	}
	RegisterDetector(name, d)
}

// ConfigToFramework converts a Config to a Framework for Dockerfile generation.
func ConfigToFramework(c *config.Config) *core.Framework {
	if c.Dockerfile != "" {
		return &core.Framework{
			Name:      "dockerfile",
			OutputDir: c.Dockerfile,
		}
	}

	fw := &core.Framework{
		Name:         runtimeToFrameworkName(c.Runtime),
		BuildCommand: c.Build.Command,
		StartCommand: c.Start.Command,
		Port:         c.RuntimeConfig.Expose,
		SystemDeps:   c.Install.SystemDeps,
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
	}

	return fw
}

func runtimeToFrameworkName(runtime string) string {
	switch runtime {
	case "node":
		return "express"
	case "python":
		return "flask"
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
		return runtime
	}
}
