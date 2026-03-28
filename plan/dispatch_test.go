package plan

import (
	"github.com/permanu/docksmith/core"
	"errors"
	"testing"
)

func TestPlan_AllFrameworks(t *testing.T) {
	cases := []struct {
		name string
		fw   core.Framework
	}{
		{name: "nextjs", fw: core.Framework{Name: "nextjs", Port: 3000, BuildCommand: "npm run build", StartCommand: "npm start"}},
		{name: "nuxt", fw: core.Framework{Name: "nuxt", Port: 3000, BuildCommand: "npm run build", StartCommand: "npm start"}},
		{name: "sveltekit", fw: core.Framework{Name: "sveltekit", Port: 3000, BuildCommand: "npm run build", StartCommand: "node build"}},
		{name: "astro", fw: core.Framework{Name: "astro", Port: 3000, BuildCommand: "npm run build", StartCommand: "node ./dist/server/entry.mjs"}},
		{name: "remix", fw: core.Framework{Name: "remix", Port: 3000, BuildCommand: "npm run build", StartCommand: "npm start"}},
		{name: "gatsby", fw: core.Framework{Name: "gatsby", Port: 9000, BuildCommand: "npm run build", StartCommand: "npx gatsby serve"}},
		{name: "vite", fw: core.Framework{Name: "vite", Port: 3000, BuildCommand: "npm run build", StartCommand: "npx serve -s dist"}},
		{name: "create-react-app", fw: core.Framework{Name: "create-react-app", Port: 3000, BuildCommand: "npm run build", StartCommand: "npx serve -s build"}},
		{name: "angular", fw: core.Framework{Name: "angular", Port: 4200, BuildCommand: "npm run build", StartCommand: "npx serve -s dist"}},
		{name: "vue-cli", fw: core.Framework{Name: "vue-cli", Port: 8080, BuildCommand: "npm run build", StartCommand: "npx serve -s dist"}},
		{name: "solidstart", fw: core.Framework{Name: "solidstart", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/server.js"}},
		{name: "nestjs", fw: core.Framework{Name: "nestjs", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/main.js"}},
		{name: "express", fw: core.Framework{Name: "express", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/main.js"}},
		{name: "fastify", fw: core.Framework{Name: "fastify", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/main.js"}},
		{name: "bun", fw: core.Framework{Name: "bun", Port: 3000, StartCommand: "bun run src/index.ts"}},
		{name: "bun-elysia", fw: core.Framework{Name: "bun-elysia", Port: 3000, StartCommand: "bun run src/index.ts"}},
		{name: "bun-hono", fw: core.Framework{Name: "bun-hono", Port: 3000, StartCommand: "bun run src/index.ts"}},
		{name: "deno", fw: core.Framework{Name: "deno", Port: 8000, StartCommand: "deno run --allow-net main.ts"}},
		{name: "deno-fresh", fw: core.Framework{Name: "deno-fresh", Port: 8000, StartCommand: "deno run --allow-net main.ts"}},
		{name: "deno-oak", fw: core.Framework{Name: "deno-oak", Port: 8000, StartCommand: "deno run --allow-net main.ts"}},
		{name: "django", fw: core.Framework{Name: "django", Port: 8000, StartCommand: "gunicorn config.wsgi:application"}},
		{name: "fastapi", fw: core.Framework{Name: "fastapi", Port: 8000, StartCommand: "uvicorn main:app --host 0.0.0.0 --port 8000"}},
		{name: "flask", fw: core.Framework{Name: "flask", Port: 5000, StartCommand: "gunicorn app:app"}},
		{name: "go", fw: core.Framework{Name: "go", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-gin", fw: core.Framework{Name: "go-gin", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-echo", fw: core.Framework{Name: "go-echo", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-fiber", fw: core.Framework{Name: "go-fiber", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-std", fw: core.Framework{Name: "go-std", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-chi", fw: core.Framework{Name: "go-chi", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "rails", fw: core.Framework{Name: "rails", Port: 3000, BuildCommand: "bundle install", StartCommand: "bundle exec rails server -b 0.0.0.0"}},
		{name: "sinatra", fw: core.Framework{Name: "sinatra", Port: 4567, BuildCommand: "bundle install", StartCommand: "ruby app.rb"}},
		{name: "laravel", fw: core.Framework{Name: "laravel", Port: 8000, BuildCommand: "composer install --no-dev", StartCommand: "php artisan serve --host=0.0.0.0 --port=8000"}},
		{name: "wordpress", fw: core.Framework{Name: "wordpress", Port: 80, StartCommand: "apache2-foreground"}},
		{name: "symfony", fw: core.Framework{Name: "symfony", Port: 8000, StartCommand: "apache2-foreground"}},
		{name: "slim", fw: core.Framework{Name: "slim", Port: 8000, StartCommand: "apache2-foreground"}},
		{name: "php", fw: core.Framework{Name: "php", Port: 8000, StartCommand: "php -S 0.0.0.0:8000"}},
		{name: "spring-boot", fw: core.Framework{Name: "spring-boot", Port: 8080, BuildCommand: "mvn -B package -DskipTests", StartCommand: "java -jar app.jar"}},
		{name: "quarkus", fw: core.Framework{Name: "quarkus", Port: 8080, BuildCommand: "./gradlew build -x test", StartCommand: "java -jar app.jar"}},
		{name: "micronaut", fw: core.Framework{Name: "micronaut", Port: 8080, BuildCommand: "./gradlew build -x test", StartCommand: "java -jar app.jar"}},
		{name: "maven", fw: core.Framework{Name: "maven", Port: 8080, BuildCommand: "mvn -B package -DskipTests", StartCommand: "java -jar app.jar"}},
		{name: "gradle", fw: core.Framework{Name: "gradle", Port: 8080, BuildCommand: "./gradlew build -x test", StartCommand: "java -jar app.jar"}},
		{name: "aspnet-core", fw: core.Framework{Name: "aspnet-core", Port: 8080, BuildCommand: "dotnet publish -c Release -o /app/publish", StartCommand: "dotnet /app/publish/app.dll"}},
		{name: "blazor", fw: core.Framework{Name: "blazor", Port: 8080, BuildCommand: "dotnet publish -c Release -o /app/publish", StartCommand: "dotnet /app/publish/app.dll"}},
		{name: "dotnet-worker", fw: core.Framework{Name: "dotnet-worker", Port: 8080, BuildCommand: "dotnet publish -c Release -o /app/publish", StartCommand: "dotnet /app/publish/app.dll"}},
		{name: "rust", fw: core.Framework{Name: "rust", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "rust-generic", fw: core.Framework{Name: "rust-generic", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "rust-actix", fw: core.Framework{Name: "rust-actix", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "rust-axum", fw: core.Framework{Name: "rust-axum", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "elixir-phoenix", fw: core.Framework{Name: "elixir-phoenix", Port: 4000, BuildCommand: "mix release", StartCommand: "./bin/server"}},
		{name: "static", fw: core.Framework{Name: "static", Port: 0}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Plan(&tc.fw)
			if err != nil {
				t.Fatalf("Plan(%q) returned error: %v", tc.name, err)
			}
			if plan.Framework != tc.name {
				t.Errorf("plan.Framework = %q, want %q", plan.Framework, tc.name)
			}
			if err := plan.Validate(); err != nil {
				t.Errorf("plan.Validate() failed: %v", err)
			}
		})
	}
}

func TestPlan_UnknownFramework(t *testing.T) {
	fw := &core.Framework{Name: "unknown-thing", Port: 8080}
	_, err := Plan(fw)
	if err == nil {
		t.Fatal("expected error for unknown framework, got nil")
	}
	if !errors.Is(err, core.ErrNotDetected) {
		t.Errorf("expected core.ErrNotDetected, got %v", err)
	}
}

func TestPlan_NilFramework(t *testing.T) {
	_, err := Plan(nil)
	if err == nil {
		t.Fatal("expected error for nil framework, got nil")
	}
}
