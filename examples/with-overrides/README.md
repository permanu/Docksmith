# with-overrides

Customize Dockerfile generation using PlanOption overrides.

## Usage

```bash
go run . /path/to/your/project > Dockerfile
```

## What it does

Detects the framework normally, then applies overrides before generating the
Dockerfile. Overrides take precedence over auto-detected values.

## Available overrides

| Option                  | Description                              |
|-------------------------|------------------------------------------|
| `WithExpose(port)`      | Override the exposed port                |
| `WithBuildCommand(cmd)` | Custom build command                     |
| `WithStartCommand(cmd)` | Custom start/entrypoint command          |
| `WithInstallCommand(cmd)` | Custom dependency install command      |
| `WithHealthcheck(cmd)`  | Add a HEALTHCHECK instruction            |
| `WithUser(name)`        | Set the USER instruction                 |
| `WithExtraEnv(map)`     | Add environment variables                |
| `WithBaseImage(img)`    | Override the build stage base image      |
| `WithRuntimeImage(img)` | Override the runtime stage base image    |
| `WithSystemDeps(deps)`  | Install additional system packages       |
| `WithEntrypoint(args)`  | Set a custom ENTRYPOINT                  |
| `WithHealthcheckDisabled()` | Remove the HEALTHCHECK instruction   |
| `WithBuildCacheDisabled()` | Disable BuildKit cache mounts          |
