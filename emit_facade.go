package docksmith

import "github.com/permanu/docksmith/emit"

// EmitDockerfile serializes a BuildPlan into a Dockerfile string.
func EmitDockerfile(plan *BuildPlan) string {
	return emit.EmitDockerfile(plan)
}

// GenerateDockerignore returns .dockerignore file content tailored to the framework.
func GenerateDockerignore(fw *Framework) string {
	return emit.GenerateDockerignore(fw)
}

// Emit helper re-exports for backward compatibility.
var sanitizeDockerfileArg = emit.SanitizeDockerfileArg
var shellSplit = emit.ShellSplit
var jsonArray = emit.JSONArray
var pmCopyLockfiles = emit.PMCopyLockfiles
