package docksmith

import (
	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/detect"
)

// Type aliases re-export detect types for backward compatibility.
type NamedDetector = detect.NamedDetector
type DetectOptions = detect.DetectOptions

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

// Package-manager helpers re-exported for plan code and backward compatibility.
var (
	pmInstallCommand = detect.PMInstallCommand
	pmRunBuild       = detect.PMRunBuild
	pmRunStart       = detect.PMRunStart
	pmRunInstall     = detect.PMRunInstall
	nodeVersionAtLeast = detect.NodeVersionAtLeast
)
