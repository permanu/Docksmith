package docksmith

import "fmt"

// GenerateDockerfile runs Plan + EmitDockerfile for the given framework.
// Returns ("", nil) when fw.Name == "dockerfile" (user already has one).
func GenerateDockerfile(fw *Framework, opts ...PlanOption) (string, error) {
	if fw == nil || fw.Name == "dockerfile" {
		return "", nil
	}
	plan, err := Plan(fw, opts...)
	if err != nil {
		return "", fmt.Errorf("generate dockerfile: %w", err)
	}
	return EmitDockerfile(plan), nil
}

// Build runs the full pipeline for dir: detect → plan → emit.
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
	plan, err := Plan(fw, planOpts...)
	if err != nil {
		return "", fw, err
	}
	return EmitDockerfile(plan), fw, nil
}
