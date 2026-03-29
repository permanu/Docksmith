# generate

Detect a project and generate a production-ready Dockerfile in one call.

## Usage

```bash
go run . /path/to/your/project > Dockerfile
```

## What it does

Uses `docksmith.Build` to run the full pipeline (detect, plan, emit) and prints
the resulting multi-stage Dockerfile to stdout. If the project already has a
Dockerfile, it exits cleanly without overwriting.

## Example

```bash
# Generate and write to file
go run . ~/my-express-app > Dockerfile

# Preview without writing
go run . ~/my-express-app
```
