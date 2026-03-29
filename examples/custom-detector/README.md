# custom-detector

Register a custom framework detector that extends docksmith's built-in detection.

## Usage

```bash
go run . /path/to/hugo-project
```

## What it does

Uses `docksmith.RegisterDetector` to add a Hugo static site detector. Custom
detectors are checked before built-in ones, so they can override default behavior.

This pattern is useful for:
- Internal frameworks not covered by built-in detectors
- Custom conventions specific to your organization
- Overriding detection for ambiguous project structures

## API

```go
docksmith.RegisterDetector(name string, fn docksmith.DetectorFunc)
docksmith.RegisterDetectorBefore(before, name string, fn docksmith.DetectorFunc)
```
