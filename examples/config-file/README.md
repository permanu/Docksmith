# config-file

Use a docksmith.toml config file to customize Dockerfile generation.

## Usage

```bash
go run . /path/to/project-with-config
```

## What it does

Loads the project's docksmith config file (TOML, YAML, or JSON), converts it to
plan options, and generates a Dockerfile. Falls back to auto-detection if no
config file is present.

## Example docksmith.toml

```toml
runtime = "node"

[build]
command = "npm run build"

[start]
command = "node dist/server.js"

[install]
system_deps = ["curl"]

[runtime_config]
expose = 3000
healthcheck = "curl -f http://localhost:3000/health || exit 1"

[env]
NODE_ENV = "production"
```

## Supported config files

- `docksmith.toml`
- `docksmith.yaml` / `docksmith.yml`
- `docksmith.json`
- `.docksmith.yaml`
