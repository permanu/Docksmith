# Docksmith

Detect any framework. Generate production Dockerfiles.

Docksmith analyzes your source code, detects the framework and runtime,
and generates optimized multi-stage Dockerfiles with cache mounts,
non-root users, and health checks built in.

## Quick Start

```bash
# detect what framework your project uses
docksmith detect .

# generate a Dockerfile
docksmith dockerfile .

# or just build it
docksmith build .
```

Zero config required. Docksmith auto-detects everything.

## Supported Runtimes

| Runtime | Frameworks |
|---------|-----------|
| Node.js | Next.js, Nuxt, SvelteKit, Astro, Remix, Vite, Angular, NestJS, Express, Fastify, and more |
| Python | Django, FastAPI, Flask |
| Go | Gin, Echo, Fiber, stdlib |
| Bun | Elysia, Hono, plain Bun |
| Deno | Fresh, Oak, plain Deno |
| Ruby | Rails, Sinatra |
| PHP | Laravel, WordPress, Symfony, Slim |
| Java | Spring Boot, Quarkus, Micronaut, Maven, Gradle |
| .NET | ASP.NET Core, Blazor |
| Rust | Actix, Axum |
| Elixir | Phoenix |
| Static | nginx |

## As a Library

```go
import "github.com/permanu/docksmith"

fw, err := docksmith.Detect("/path/to/project")
plan := docksmith.Plan(fw)
dockerfile := docksmith.EmitDockerfile(plan)
```

## Custom Framework Support

Add detection for any framework with a YAML file:

```yaml
# ~/.docksmith/frameworks/my-framework.yaml
name: my-framework
runtime: node
detect:
  any:
    - dependency: my-framework
plan:
  port: 3000
  stages:
    - name: runtime
      base: "node:{{version}}-alpine"
      steps:
        - copy: ["."]
        - cmd: ["npm", "start"]
```

## Override Detection

When auto-detection gets something wrong, create `docksmith.toml`:

```toml
build = "bun run build"
start = "bun run start"
port = 3000
```

## Install

```bash
go install github.com/permanu/docksmith/cmd/docksmith@latest
```

Or with Homebrew (coming soon):

```bash
brew install permanu/tap/docksmith
```

## License

Apache License 2.0. See [LICENSE](LICENSE).
