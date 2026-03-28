package docksmith

import "fmt"

// Plan converts a detected Framework into a BuildPlan.
func Plan(fw *Framework) (*BuildPlan, error) {
	if fw == nil {
		return nil, fmt.Errorf("%w: nil framework", ErrNotDetected)
	}
	switch {
	// Bun must precede Node — bun projects also have package.json.
	case isBunFramework(fw.Name):
		return planBun(fw)
	case isDenoFramework(fw.Name):
		return planDeno(fw)
	case isNodeFramework(fw.Name):
		return planNode(fw)
	case isPythonFramework(fw.Name):
		return planPython(fw)
	case isGoFramework(fw.Name):
		return planGo(fw)
	case isRubyFramework(fw.Name):
		return planRuby(fw)
	case isPHPFramework(fw.Name):
		return planPHP(fw)
	case isJavaFramework(fw.Name):
		return planJava(fw)
	case isDotnetFramework(fw.Name):
		return planDotnet(fw)
	case isRustFramework(fw.Name):
		return planRust(fw)
	case fw.Name == "elixir-phoenix":
		return planElixir(fw)
	case fw.Name == "static":
		return planStatic(fw)
	default:
		return nil, fmt.Errorf("%w: %q", ErrNotDetected, fw.Name)
	}
}

// Bun detectors must run before Node — bun projects also have package.json.
func isBunFramework(name string) bool {
	return name == "bun" || name == "bun-elysia" || name == "bun-hono"
}

func isDenoFramework(name string) bool {
	return name == "deno" || name == "deno-fresh" || name == "deno-oak"
}

func isNodeFramework(name string) bool {
	switch name {
	case "nextjs", "nuxt", "sveltekit", "astro", "remix", "gatsby",
		"vite", "create-react-app", "angular", "vue-cli", "solidstart",
		"nestjs", "express", "fastify":
		return true
	}
	return false
}

func isPythonFramework(name string) bool {
	return name == "django" || name == "fastapi" || name == "flask"
}

func isGoFramework(name string) bool {
	switch name {
	case "go", "go-gin", "go-echo", "go-fiber", "go-std", "go-chi":
		return true
	}
	return false
}

func isRubyFramework(name string) bool {
	return name == "rails" || name == "sinatra"
}

func isPHPFramework(name string) bool {
	return name == "laravel" || name == "wordpress" || name == "symfony" || name == "slim" || name == "php"
}

func isJavaFramework(name string) bool {
	return name == "spring-boot" || name == "quarkus" || name == "micronaut" ||
		name == "maven" || name == "gradle"
}

func isDotnetFramework(name string) bool {
	return name == "aspnet-core" || name == "blazor" || name == "dotnet-worker"
}

func isRustFramework(name string) bool {
	return name == "rust" || name == "rust-generic" || name == "rust-actix" || name == "rust-axum"
}
