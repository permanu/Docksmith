# docksmith examples

Runnable examples showing how to use docksmith as a Go library.

## Examples

| Directory | Description |
|-----------|-------------|
| [detect/](detect/) | Detect a framework from a project directory |
| [generate/](generate/) | Generate a Dockerfile in one call (detect + plan + emit) |
| [pipeline/](pipeline/) | Full pipeline with each stage shown separately |
| [custom-detector/](custom-detector/) | Register a custom framework detector |
| [with-overrides/](with-overrides/) | Use PlanOption overrides (port, build, start, healthcheck) |
| [config-file/](config-file/) | Load a docksmith.toml config to drive generation |

## Running an example

```bash
cd examples/detect
go run . /path/to/your/project
```

## Building all examples

```bash
go build ./examples/...
```
