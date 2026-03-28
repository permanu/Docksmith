package core

// IsBunFramework reports whether name is a Bun-based framework.
func IsBunFramework(name string) bool {
	return name == "bun" || name == "bun-elysia" || name == "bun-hono"
}

// IsDenoFramework reports whether name is a Deno-based framework.
func IsDenoFramework(name string) bool {
	return name == "deno" || name == "deno-fresh" || name == "deno-oak"
}

// IsNodeFramework reports whether name is a Node.js-based framework.
func IsNodeFramework(name string) bool {
	switch name {
	case "nextjs", "nuxt", "sveltekit", "astro", "remix", "gatsby",
		"vite", "create-react-app", "angular", "vue-cli", "solidstart",
		"nestjs", "express", "fastify":
		return true
	}
	return false
}

// IsPythonFramework reports whether name is a Python-based framework.
func IsPythonFramework(name string) bool {
	return name == "django" || name == "fastapi" || name == "flask"
}

// IsGoFramework reports whether name is a Go-based framework.
func IsGoFramework(name string) bool {
	switch name {
	case "go", "go-gin", "go-echo", "go-fiber", "go-std", "go-chi":
		return true
	}
	return false
}

// IsRubyFramework reports whether name is a Ruby-based framework.
func IsRubyFramework(name string) bool {
	return name == "rails" || name == "sinatra"
}

// IsPHPFramework reports whether name is a PHP-based framework.
func IsPHPFramework(name string) bool {
	return name == "laravel" || name == "wordpress" || name == "symfony" || name == "slim" || name == "php"
}

// IsJavaFramework reports whether name is a Java-based framework.
func IsJavaFramework(name string) bool {
	return name == "spring-boot" || name == "quarkus" || name == "micronaut" ||
		name == "maven" || name == "gradle"
}

// IsDotnetFramework reports whether name is a .NET-based framework.
func IsDotnetFramework(name string) bool {
	return name == "aspnet-core" || name == "blazor" || name == "dotnet-worker"
}

// IsRustFramework reports whether name is a Rust-based framework.
func IsRustFramework(name string) bool {
	return name == "rust" || name == "rust-generic" || name == "rust-actix" || name == "rust-axum"
}
