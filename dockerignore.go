package docksmith

import "strings"

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
func GenerateDockerignore(fw *Framework) string {
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
	case isNodeFramework(name) || isDenoFramework(name):
		return []string{"node_modules", ".next", ".nuxt", "dist", "build", ".cache", "coverage"}

	case isBunFramework(name):
		return []string{"node_modules", ".next", "dist", "build", ".cache", "coverage"}

	case isPythonFramework(name):
		return []string{"__pycache__", "*.pyc", ".venv", "venv", ".pytest_cache", ".mypy_cache", "*.egg-info"}

	case isGoFramework(name):
		return []string{"vendor", "*.test", "*.out"}

	case isRubyFramework(name):
		return []string{".bundle", "vendor/bundle", "log", "tmp"}

	case isPHPFramework(name):
		return []string{"vendor", "storage/logs", "bootstrap/cache"}

	case isJavaFramework(name):
		return []string{"target", "build", ".gradle", "*.class", "*.jar"}

	case isRustFramework(name):
		return []string{"target"}

	case isDotnetFramework(name):
		return []string{"bin", "obj", "*.user", "*.suo"}

	default:
		return nil
	}
}
