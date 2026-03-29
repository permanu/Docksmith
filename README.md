<!--
docksmith: Go library + CLI for framework detection and Dockerfile generation
module: github.com/permanu/docksmith
language: go
go-version: ">=1.26"
license: Apache-2.0
dependencies: 2
runtimes: 12
detectors: 45
architecture: detect-plan-emit
output: dockerfile
status: pre-release
-->

# Docksmith

Go library and CLI that detects your framework and generates a hardened, multi-stage Dockerfile. Two dependencies. No lock-in.

## At a glance

| | |
|---|---|
| **Type** | Go library + CLI |
| **Module** | `github.com/permanu/docksmith` |
| **Go version** | 1.26+ |
| **Dependencies** | 2 (BurntSushi/toml, yaml.v3) |
| **Runtimes** | 12 (Node, Python, Go, Ruby, Rust, Java, PHP, .NET, Elixir, Deno, Bun, Static) |
| **Detectors** | 45 built-in + static fallback |
| **Architecture** | Detect → Plan → Emit (each independently usable) |
| **Output** | Plain Dockerfile (committable, no lock-in) |
| **API docs** | [pkg.go.dev/github.com/permanu/docksmith](https://pkg.go.dev/github.com/permanu/docksmith) |
| **License** | Apache 2.0 |
| **Status** | Pre-release; internal use at Permanu |

```
$ docksmith detect .
framework:  express
port:       3000
node:       22
pm:         npm

$ docksmith dockerfile . > Dockerfile
```

## What comes out

Point docksmith at an Express app with a `package.json` and it produces this:

```dockerfile
# syntax=docker/dockerfile:1

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    if [ -f package-lock.json ]; then npm ci || npm install; else npm install; fi
RUN apk add --no-cache tini

FROM deps AS build
COPY . .
RUN npm install

FROM node:22-alpine AS runtime
WORKDIR /app
COPY --from=build --link /app /app
CMD ["npm", "start"]
COPY --from=deps /sbin/tini /sbin/tini
ENTRYPOINT ["/sbin/tini", "--"]
USER node
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD node -e "const http=require('http');http.get('http://localhost:3000/', \
    r=>{process.exit(r.statusCode===200?0:1)}).on('error',()=>process.exit(1))"
```

Things to notice: multi-stage build, BuildKit cache mounts, tini as PID 1, non-root user, health check using only Node stdlib (no curl needed). No manual work.

For Go, it uses distroless -- zero shell attack surface:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/app .

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=builder /app/app ./app
USER nonroot
EXPOSE 8080
CMD ["./app"]
```

## Install

**Requirements:** Go 1.26+ (for library/CLI install). Docker only needed for `docksmith build`.

```bash
go install github.com/permanu/docksmith/cmd/docksmith@latest
```

Or grab a binary from the [releases page](https://github.com/permanu/docksmith/releases).

## Quick start

```bash
docksmith detect .                   # what framework is this?
docksmith dockerfile . > Dockerfile  # generate it
docker build -t myapp .             # build normally
```

Other commands: `plan` (inspect the build plan), `eject` (write Dockerfile + .dockerignore to disk), `init` (generate a docksmith.toml pre-filled from detection), `build` (detect + generate + docker build in one shot), `registry search` (find community framework definitions), `registry install` (install a YAML definition from the registry).

## How it works

Three stages, each independently usable:

1. **Detect** -- scans your project for package.json, go.mod, requirements.txt, Cargo.toml, etc. Identifies the framework, runtime version, and package manager. 45 detectors, each a small function.
2. **Plan** -- takes the detection result and builds a `BuildPlan`: which base images, which stages, what commands, plus hardening (non-root user, tini, health checks, distroless where possible, BuildKit cache mounts).
3. **Emit** -- serializes the plan into a Dockerfile string. Standard Dockerfile syntax. No proprietary format, no runtime dependency.

The entire library has two external dependencies (a TOML parser and a YAML parser). No Docker client, no container runtime, no network calls during detection or generation.

The output is a plain Dockerfile. You can commit it, modify it, or throw it away. No lock-in.

## What the generated Dockerfile includes

Every generated Dockerfile gets these by default:

- **Multi-stage builds** -- separate dependency, build, and runtime stages. Only runtime artifacts end up in the final image.
- **BuildKit cache mounts** -- `--mount=type=cache` for package manager caches (npm, pip, Go modules, etc.), so rebuilds reuse downloaded dependencies.
- **Non-root user** -- the final stage always runs as a non-root user (`node`, `nonroot`, etc. depending on the runtime).
- **Tini as PID 1** -- Node.js and Python containers use tini to handle signals and zombie processes correctly.
- **Distroless runtime** -- Go and Rust containers use `gcr.io/distroless/static` for the runtime stage. No shell, no package manager, minimal attack surface.
- **Health checks** -- auto-injected per runtime. Node uses stdlib `http.get`, Python uses `urllib`, Java uses `wget`. Go distroless containers do not get a health check (no shell to run it in).

## How docksmith compares

There are good tools in this space. Here's why I built another one.

**Railpack** powers all Railway deployments and produces smaller images than docksmith does — Railway reports 38% smaller Node and 77% smaller Python images vs Nixpacks. It builds OCI images directly via BuildKit LLB, which means parallel layer execution and graph-level optimization that a flat Dockerfile can't match. If you're on Railway, just use Railpack. If you're building your own platform and want to hand users a Dockerfile they can read, modify, and commit — that's where docksmith fits.

**Nixpacks** is in maintenance mode (Railway recommends Railpack), but it covers 23 language providers including Crystal, Haskell, Dart, and Zig. Docksmith doesn't support those. If you need one of those runtimes, Nixpacks is still the answer.

**Cloud Native Buildpacks** (Paketo) is the most mature option — CNCF project, used by Heroku and Google Cloud. The tradeoff is opacity: you get OCI layers, not a Dockerfile. You can't read, tweak, or commit what it produces. For teams that want full control over their container definition, that's a dealbreaker.

**Docksmith** generates plain Dockerfiles. That's the whole point. You get a readable file with multi-stage builds, non-root users, tini, distroless bases, health checks, and cache mounts — things most teams know they should add but skip because it's tedious. The output has no runtime dependency on docksmith. Commit it, forget docksmith exists, and your Dockerfile still works.

The library angle matters too: docksmith is a Go library first, CLI second. If you're building a PaaS, a deploy tool, or a CI pipeline that needs framework detection or Dockerfile generation, you can `import "github.com/permanu/docksmith"` and call three functions. Railpack and Nixpacks can do this too, but Buildpacks are harder to embed.

## Supported frameworks

| Runtime | Frameworks |
|---|---|
| Node.js | Next.js, Nuxt, SvelteKit, Astro, Remix, Gatsby, Vite, Create React App, Angular, Vue CLI, SolidStart, NestJS, Express, Fastify |
| Python | Django, FastAPI, Flask |
| Go | Gin, Echo, Fiber, net/http |
| Ruby | Rails, Sinatra |
| Rust | Actix, Axum |
| Java | Spring Boot, Quarkus, Micronaut, Maven (generic), Gradle (generic) |
| PHP | Laravel, Symfony, Slim, WordPress, plain PHP |
| .NET | ASP.NET Core, Blazor, Worker |
| Elixir | Phoenix |
| Deno | Fresh, Oak, plain Deno |
| Bun | Elysia, Hono, plain Bun |
| Static | HTML/CSS/JS (served via nginx, detected as fallback) |

Detection is priority-ordered. More specific frameworks (e.g., Next.js) are checked before generic ones (e.g., plain Node). If nothing matches and the directory has no web-servable content, docksmith returns an error -- it does not silently generate a broken Dockerfile.

## Library usage

Docksmith is a Go library first. The CLI is a thin wrapper. Full API documentation is on [pkg.go.dev](https://pkg.go.dev/github.com/permanu/docksmith).

```go
import "github.com/permanu/docksmith"

// Detect framework
fw, err := docksmith.Detect("./my-project")

// Generate build plan
plan, err := docksmith.Plan(fw)

// Emit Dockerfile
dockerfile := docksmith.EmitDockerfile(plan)
```

One-shot, with overrides:

```go
dockerfile, fw, err := docksmith.Build("./my-project",
    docksmith.WithExpose(8080),
    docksmith.WithStartCommand("./server"),
    docksmith.WithExtraEnv(map[string]string{"GIN_MODE": "release"}),
)
```

Available options: `WithUser`, `WithHealthcheck`, `WithHealthcheckDisabled`, `WithRuntimeImage`, `WithBaseImage`, `WithEntrypoint`, `WithExtraEnv`, `WithExpose`, `WithInstallCommand`, `WithBuildCommand`, `WithStartCommand`, `WithSystemDeps`, `WithBuildCacheDisabled`, `WithSecrets`, `WithContextRoot`.

Each subpackage (`detect`, `plan`, `emit`, `config`, `yamldef`) is independently importable:

```go
import (
    "github.com/permanu/docksmith/detect"
    "github.com/permanu/docksmith/plan"
    "github.com/permanu/docksmith/emit"
)

// Use detect alone — CI tools that need framework metadata
fw, _ := detect.Detect("./my-project")
fmt.Println(fw.Name, fw.Port) // "nextjs", 3000

// Use plan alone — tools that need build step info
bp, _ := plan.Plan(fw)
fmt.Println(len(bp.Stages)) // 3 (deps, build, runtime)

// Use emit alone — construct plans programmatically
dockerfile := emit.EmitDockerfile(customPlan)
```

## Configuration

Create `docksmith.toml` in your project root to override detected defaults:

```toml
runtime = "node"
version = "22"

[build]
command = "bun run build"

[start]
command = "bun run start"

[install]
command = "bun install --frozen-lockfile"
system_deps = ["libpq-dev"]

[runtime_config]
image = "node:22-alpine"
expose = 3000

[env]
NODE_ENV = "production"
```

Monorepo support -- separate build context from app directory:

```toml
context_root = "."  # repo root as Docker build context
# run: docksmith dockerfile --root . ./apps/frontend
```

Buildtime secrets for private registries and API keys:

```toml
[secrets]
npm = { target = "/root/.npmrc" }
license_key = { env = "LICENSE_KEY" }
```

When private registry files (`.npmrc`, `pip.conf`, `.netrc`, `settings.xml`) are detected, docksmith auto-generates BuildKit `--mount=type=secret` instructions and excludes those files from `.dockerignore`.

Also reads `docksmith.yaml` and `docksmith.json`. Run `docksmith init` to generate a config pre-filled from detection.

## Custom framework definitions

Add detection for any framework with a YAML file in `~/.docksmith/frameworks/` or `.docksmith/frameworks/` in your project:

```yaml
name: my-framework
runtime: node
priority: 10

detect:
  all:
    - dependency: my-framework
  any:
    - file: my-framework.config.js

defaults:
  build: "npm run build"
  start: "npm start"

plan:
  port: 3000
  stages:
    - name: builder
      base: node
      steps:
        - workdir: /app
        - copy: ["package*.json", "."]
        - run: "{{install_command}}"
          cache: /root/.npm
        - copy: [".", "."]
        - run: "{{build_command}}"
    - name: runtime
      base: node
      steps:
        - workdir: /app
        - copy_from:
            stage: builder
            src: /app
            dst: .
        - cmd: ["node", "server.js"]

tests:
  - name: detects my-framework
    fixture:
      package.json: '{"dependencies": {"my-framework": "^1.0.0"}}'
    expect:
      detected: true
      framework: my-framework
```

YAML definitions include inline tests. Run them with `docksmith test my-framework.yaml`. The community registry (`docksmith registry search`) provides additional definitions.

## Development

```bash
make check    # fmt + vet + test + lint (full CI gate)
make test     # tests with race detector
make lint     # golangci-lint
make build    # build CLI to bin/docksmith
make fuzz     # 30s fuzz testing
```

Requires Go 1.26+ and optionally golangci-lint. The `testdata/fixtures/` directory has sample projects for each framework — adding a new framework means adding a detector, a planner, and a fixture.

See [CONTRIBUTING.md](CONTRIBUTING.md) for full guidelines.

## License

Apache License 2.0. See [LICENSE](LICENSE).
