// Package docksmith detects application frameworks and generates
// production-ready Dockerfiles.
//
// The pipeline has three stages:
//
//	fw, err := docksmith.Detect("/path/to/project")
//	plan := docksmith.Plan(fw)
//	dockerfile := docksmith.EmitDockerfile(plan)
//
// Each stage is independently useful. Detect analyzes project files
// to identify the framework and runtime. Plan converts a Framework
// into abstract build steps. EmitDockerfile serializes a plan into
// a multi-stage Dockerfile with cache mounts and non-root users.
//
// Detection supports 45 frameworks across 12 runtimes. Custom
// frameworks can be added via YAML definitions or RegisterDetector.
package docksmith
