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
status: internal-testing
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
| **Status** | Internal use at Permanu; not tested at external production scale |

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

Other commands: `plan` (inspect the build plan), `eject` (write Dockerfile + .dockerignore to disk), `init` (generate a docksmith.toml pre-filled from detection), `build` (detect + generate + docker build in one shot).

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

## Comparison

| | Docksmith | Railpack | Nixpacks | Cloud Native Buildpacks |
|---|---|---|---|---|
| **Approach** | Generates Dockerfiles | Builds OCI images via BuildKit LLB | Generates Dockerfiles | Builds OCI images |
| **Written in** | Go | Go | Rust | Go/Java |
| **Output** | Readable, committable Dockerfile | OCI image (no intermediate Dockerfile) | Dockerfile | Opaque OCI layers |
| **Use as library** | Yes (Go API) | Go API + CLI | CLI only | CLI + pack API |
| **Languages** | 12 runtimes, 45 framework detectors | 11 language providers | 23 language providers | 9 language families (Paketo) |
| **Multi-stage builds** | Always | BuildKit layers (implicit) | Partial | No |
| **Non-root user** | Always | Depends on provider | Sometimes | Sometimes |
| **Health checks** | Auto-injected per runtime | No | No | No |
| **Tini init** | Node, Python | No | No | No |
| **Distroless** | Go, Rust | No | No | No |
| **Monorepo support** | Manual (point at app dir) | Yes (workspace detection) | Limited | Varies |
| **Buildtime secrets** | No | Yes (BuildKit secrets) | No | Varies |
| **Runtime mgmt** | Alpine apk | Mise | Nix | Buildpack-provided |
| **Custom frameworks** | YAML definitions + Go API | Provider plugins (Go) | Nix expressions | Buildpacks (complex) |
| **Status** | Internal testing at Permanu | Powers all Railway deployments | Maintenance mode | Mature, wide adoption |
| **License** | Apache 2.0 | MIT | MIT | Apache 2.0 |

### Comparison notes

**Railpack** — Railway's successor to Nixpacks (January 2025). Builds OCI images directly via BuildKit LLB using Mise for runtime management. Supports monorepos, BuildKit secrets, and SPA frameworks. Railway reports 38% smaller Node and 77% smaller Python images vs Nixpacks.
- *Stronger than Docksmith*: monorepo support, proven at Railway's scale, smaller images (graph-based parallel BuildKit execution), SPA-specific optimizations (asset hashing, CDN headers, fallback routing)
- *Docksmith differs*: readable/committable Dockerfile output, embeddable Go library, hardening defaults (tini, distroless, health checks)

**Nixpacks** — Maintenance mode. Railway recommends Railpack. Widest language coverage (23 providers including Crystal, Haskell, Dart, Zig, and others docksmith does not support).

**Cloud Native Buildpacks / Paketo** — Most mature option. CNCF project. Used by Heroku, Google Cloud, and Spring Boot. Tradeoff: opaque OCI layers — you cannot inspect or modify the generated layers the way you can with a Dockerfile.

**Docksmith** — Choose when you want readable Dockerfiles, a Go library to embed in your own platform, or hardening defaults without manual work. Does not build images itself — generates Dockerfiles and hands off to Docker/BuildKit. Not yet tested at production scale outside Permanu.

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

Available options: `WithUser`, `WithHealthcheck`, `WithHealthcheckDisabled`, `WithRuntimeImage`, `WithBaseImage`, `WithEntrypoint`, `WithExtraEnv`, `WithExpose`, `WithInstallCommand`, `WithBuildCommand`, `WithStartCommand`, `WithSystemDeps`, `WithBuildCacheDisabled`.

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

## Limitations

Things docksmith does not do, or does not do well yet:

- **Monorepos.** Docksmith operates on a single directory. You can point it at a subdirectory (`docksmith detect ./apps/frontend`) and it will detect and generate a Dockerfile for that app. However, the generated `COPY . .` uses that directory as the Docker build context — shared packages or configs above the app directory are not included. There is no separate root-dir vs app-dir concept, no workspace root detection, and no multi-app orchestration.
- **Private registries.** No built-in support for authenticating to private npm/PyPI/Go module registries during builds. You can work around this with BuildKit secrets, but docksmith does not generate those steps.
- **Buildtime secrets.** No secret injection. If your build needs API keys or tokens, you need to add those steps manually.
- **Runtime configuration.** Docksmith generates a Dockerfile. It does not handle runtime concerns like environment variables, volumes, networking, or orchestration.
- **Image building.** The `build` command shells out to `docker build`. There is no native BuildKit client. If Docker is not installed, `build` fails.
- **Windows containers.** Linux only. No Windows container support.
- **Not tested at scale.** Docksmith is used internally at Permanu and has been tested via E2E on VPS deployments. It has not served production traffic at scale.

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
