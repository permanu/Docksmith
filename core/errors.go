package core

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotDetected   = errors.New("no framework detected")
	ErrInvalidConfig = errors.New("invalid config")
	ErrInvalidPlan   = errors.New("invalid build plan")
)

// NearMiss records a partial detection match — e.g., go.mod found but no main package.
type NearMiss struct {
	Runtime  string // e.g. "go", "python", "node"
	Found    string // what was found, e.g. "go.mod"
	Missing  string // what was missing, e.g. "main package (main.go or cmd/*/main.go)"
	Hint     string // actionable suggestion
}

// String formats a near-miss for display.
func (nm NearMiss) String() string {
	s := fmt.Sprintf("found %s but missing %s", nm.Found, nm.Missing)
	if nm.Hint != "" {
		s += " — " + nm.Hint
	}
	return s
}

// DetectionError provides rich context when framework detection fails.
// It wraps ErrNotDetected so errors.Is(err, ErrNotDetected) remains true.
type DetectionError struct {
	Dir          string     // directory that was scanned
	FilesChecked []string   // marker files/patterns that were looked for
	NearMisses   []NearMiss // partial matches
}

// Error formats the full diagnostic message.
func (e *DetectionError) Error() string {
	var b strings.Builder
	b.WriteString("no framework detected in ")
	b.WriteString(e.Dir)

	if len(e.NearMisses) > 0 {
		b.WriteString("\n\nnear matches:")
		for _, nm := range e.NearMisses {
			b.WriteString("\n  - ")
			b.WriteString(nm.String())
		}
	}

	if len(e.FilesChecked) > 0 {
		b.WriteString("\n\nscanned for: ")
		b.WriteString(strings.Join(e.FilesChecked, ", "))
	}

	b.WriteString("\n\nto fix, try one of:")
	b.WriteString("\n  1. add a docksmith.toml config file:")
	b.WriteString("\n")
	b.WriteString(exampleConfig(e.NearMisses))
	b.WriteString("\n  2. add a Dockerfile to your project root")
	b.WriteString("\n  3. specify the framework manually: docksmith build --framework <name>")
	b.WriteString("\n  4. search the community registry: docksmith registry search <runtime>")

	return b.String()
}

// Unwrap returns ErrNotDetected so errors.Is works.
func (e *DetectionError) Unwrap() error {
	return ErrNotDetected
}

// exampleConfig generates a minimal docksmith.toml example, tailored to the
// best near-miss if available.
func exampleConfig(nearMisses []NearMiss) string {
	runtime := "go"
	startCmd := "./app"
	if len(nearMisses) > 0 {
		switch nearMisses[0].Runtime {
		case "node":
			runtime = "node"
			startCmd = "npm start"
		case "python":
			runtime = "python"
			startCmd = "gunicorn app:app --bind 0.0.0.0:8000"
		case "ruby":
			runtime = "ruby"
			startCmd = "bundle exec rails server"
		case "rust":
			runtime = "rust"
			startCmd = "./target/release/app"
		case "java":
			runtime = "java"
			startCmd = "java -jar target/*.jar"
		case "php":
			runtime = "php"
			startCmd = "php -S 0.0.0.0:8000 -t public"
		case "elixir":
			runtime = "elixir"
			startCmd = "mix phx.server"
		case "dotnet":
			runtime = "dotnet"
			startCmd = "dotnet /app/publish/MyApp.dll"
		case "go":
			// default above
		}
	}

	return fmt.Sprintf(`     runtime = %q
     [start]
     command = %q
`, runtime, startCmd)
}
