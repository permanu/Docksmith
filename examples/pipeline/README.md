# pipeline

Demonstrates each stage of the docksmith pipeline separately.

## Usage

```bash
go run . /path/to/your/project
```

## What it does

Runs detect, plan, and emit as three independent steps, printing diagnostic
information to stderr at each stage. The final Dockerfile goes to stdout.

This is useful for understanding or debugging the generation process, or for
inserting custom logic between stages (e.g., modifying the BuildPlan before
emitting).

## Pipeline stages

1. **Detect** -- Identifies framework, runtime version, package manager
2. **Plan** -- Converts the Framework into abstract build stages and steps
3. **Emit** -- Serializes the plan into a Dockerfile string
