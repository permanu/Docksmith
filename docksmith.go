package docksmith

import "fmt"

// GenerateDockerfile runs Plan + EmitDockerfile for the given framework.
// Returns ("", nil) when fw.Name == "dockerfile" (user already has one).
func GenerateDockerfile(fw *Framework) (string, error) {
	if fw == nil || fw.Name == "dockerfile" {
		return "", nil
	}
	plan, err := Plan(fw)
	if err != nil {
		return "", fmt.Errorf("generate dockerfile: %w", err)
	}
	return EmitDockerfile(plan), nil
}

// Build runs the full pipeline for dir: detect → plan → emit.
func Build(dir string) (string, *Framework, error) {
	return BuildWithOptions(dir, DetectOptions{})
}

// BuildWithOptions runs the pipeline with custom detection options.
func BuildWithOptions(dir string, opts DetectOptions) (string, *Framework, error) {
	fw, err := DetectWithOptions(dir, opts)
	if err != nil {
		return "", nil, fmt.Errorf("build: %w", err)
	}
	if fw.Name == "dockerfile" {
		return "", fw, nil
	}
	plan, err := Plan(fw)
	if err != nil {
		return "", fw, err
	}
	return EmitDockerfile(plan), fw, nil
}
