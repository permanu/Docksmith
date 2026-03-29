# Contributing to Docksmith

## Adding Framework Support

The easiest way to contribute is adding detection for a new framework.

### Option 1: YAML Definition (preferred)

Drop a YAML file in `frameworks/`. No Go required.

```yaml
name: my-framework
runtime: node
priority: 10

detect:
  all:
    - file: package.json
  any:
    - dependency: my-framework

plan:
  port: 3000
  stages:
    - name: deps
      base: "node:{{version}}-alpine"
      steps:
        - copy: ["package.json"]
        - run: "npm ci"
    - name: runtime
      from: deps
      steps:
        - copy: ["."]
        - cmd: ["npm", "start"]

tests:
  - name: "basic detection"
    fixture:
      package.json: '{"dependencies": {"my-framework": "1.0.0"}}'
    expect:
      detected: true
```

Validate with `docksmith test frameworks/my-framework.yaml`.

### Option 2: Go Implementation

For frameworks needing complex detection logic:

1. Add detector in `detect_<runtime>.go`
2. Register in the `detectors` slice in `detect.go`
3. Add plan builder in `plan_<runtime>.go`
4. Add testdata fixtures in `testdata/fixtures/<framework>/`

## Development

```bash
make check    # build + vet + test + lint
make test     # tests with race detector
make lint     # golangci-lint
```

## Pull Request Process

1. Fork the repo and create a branch from `main`
2. Write tests first (TDD)
3. Run `make check` — all gates must pass
4. Keep changes focused — one feature or fix per PR
5. Verify all tests pass with `make test`

## Code Style

- Go 1.26, idiomatic
- `gofmt` and `goimports` enforced
- See `.golangci.yml` for lint rules
- No file over 300 lines, no function over 50 lines

## Commit Messages

Imperative mood, max 72 chars. Body explains why, not what.

```
add gleam framework detection

Detect Gleam projects by looking for gleam.toml in the project root.
```

## License

By contributing, you agree that your contributions will be licensed
under the Apache License 2.0.
