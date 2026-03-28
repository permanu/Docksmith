package emit

import (
	"strings"

	"github.com/permanu/docksmith/core"
)

// baseIgnorePatterns are included for every runtime.
var baseIgnorePatterns = []string{
	".git",
	".gitignore",
	"README.md",
	"LICENSE",
	".env",
	".env.*",
	"Dockerfile",
	".dockerignore",
	"docker-compose*.yml",
}

// GenerateDockerignore returns .dockerignore file content tailored to the framework.
func GenerateDockerignore(fw *core.Framework) string {
	patterns := make([]string, len(baseIgnorePatterns))
	copy(patterns, baseIgnorePatterns)
	patterns = append(patterns, runtimeIgnorePatterns(fw.Name)...)

	var b strings.Builder
	for _, p := range patterns {
		b.WriteString(p)
		b.WriteByte('\n')
	}
	return b.String()
}

func runtimeIgnorePatterns(name string) []string {
	switch {
	case core.IsNodeFramework(name) || core.IsDenoFramework(name):
		return []string{"node_modules", ".next", ".nuxt", "dist", "build", ".cache", "coverage"}

	case core.IsBunFramework(name):
		return []string{"node_modules", ".next", "dist", "build", ".cache", "coverage"}

	case core.IsPythonFramework(name):
		return []string{"__pycache__", "*.pyc", ".venv", "venv", ".pytest_cache", ".mypy_cache", "*.egg-info"}

	case core.IsGoFramework(name):
		return []string{"vendor", "*.test", "*.out"}

	case core.IsRubyFramework(name):
		return []string{".bundle", "vendor/bundle", "log", "tmp"}

	case core.IsPHPFramework(name):
		return []string{"vendor", "storage/logs", "bootstrap/cache"}

	case core.IsJavaFramework(name):
		return []string{"target", "build", ".gradle", "*.class", "*.jar"}

	case core.IsRustFramework(name):
		return []string{"target"}

	case core.IsDotnetFramework(name):
		return []string{"bin", "obj", "*.user", "*.suo"}

	default:
		return nil
	}
}
