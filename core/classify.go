package core

// Runtime classifiers used by Plan to dispatch to the correct planner
// (planNode, planPython, planGo, etc.). Each returns true for framework
// names belonging to that runtime.

func IsBunFramework(name string) bool {
	return name == "bun" || name == "bun-elysia" || name == "bun-hono"
}

func IsDenoFramework(name string) bool {
	return name == "deno" || name == "deno-fresh" || name == "deno-oak"
}

func IsNodeFramework(name string) bool {
	switch name {
	case "nextjs", "nuxt", "sveltekit", "astro", "remix", "gatsby",
		"vite", "create-react-app", "angular", "vue-cli", "solidstart",
		"nestjs", "express", "fastify":
		return true
	}
	return false
}

func IsPythonFramework(name string) bool {
	return name == "django" || name == "fastapi" || name == "flask"
}

func IsGoFramework(name string) bool {
	switch name {
	case "go", "go-gin", "go-echo", "go-fiber", "go-std", "go-chi":
		return true
	}
	return false
}

func IsRubyFramework(name string) bool {
	return name == "rails" || name == "sinatra"
}

func IsPHPFramework(name string) bool {
	return name == "laravel" || name == "wordpress" || name == "symfony" || name == "slim" || name == "php"
}

func IsJavaFramework(name string) bool {
	return name == "spring-boot" || name == "quarkus" || name == "micronaut" ||
		name == "maven" || name == "gradle"
}

func IsDotnetFramework(name string) bool {
	return name == "aspnet-core" || name == "blazor" || name == "dotnet-worker"
}

func IsRustFramework(name string) bool {
	return name == "rust" || name == "rust-generic" || name == "rust-actix" || name == "rust-axum"
}
