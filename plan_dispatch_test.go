package docksmith

import (
	"errors"
	"testing"
)

func TestPlan_AllFrameworks(t *testing.T) {
	cases := []struct {
		name string
		fw   Framework
	}{
		{name: "nextjs", fw: Framework{Name: "nextjs", Port: 3000, BuildCommand: "npm run build", StartCommand: "npm start"}},
		{name: "nuxt", fw: Framework{Name: "nuxt", Port: 3000, BuildCommand: "npm run build", StartCommand: "npm start"}},
		{name: "sveltekit", fw: Framework{Name: "sveltekit", Port: 3000, BuildCommand: "npm run build", StartCommand: "node build"}},
		{name: "astro", fw: Framework{Name: "astro", Port: 3000, BuildCommand: "npm run build", StartCommand: "node ./dist/server/entry.mjs"}},
		{name: "remix", fw: Framework{Name: "remix", Port: 3000, BuildCommand: "npm run build", StartCommand: "npm start"}},
		{name: "gatsby", fw: Framework{Name: "gatsby", Port: 9000, BuildCommand: "npm run build", StartCommand: "npx gatsby serve"}},
		{name: "vite", fw: Framework{Name: "vite", Port: 3000, BuildCommand: "npm run build", StartCommand: "npx serve -s dist"}},
		{name: "create-react-app", fw: Framework{Name: "create-react-app", Port: 3000, BuildCommand: "npm run build", StartCommand: "npx serve -s build"}},
		{name: "angular", fw: Framework{Name: "angular", Port: 4200, BuildCommand: "npm run build", StartCommand: "npx serve -s dist"}},
		{name: "vue-cli", fw: Framework{Name: "vue-cli", Port: 8080, BuildCommand: "npm run build", StartCommand: "npx serve -s dist"}},
		{name: "solidstart", fw: Framework{Name: "solidstart", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/server.js"}},
		{name: "nestjs", fw: Framework{Name: "nestjs", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/main.js"}},
		{name: "express", fw: Framework{Name: "express", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/main.js"}},
		{name: "fastify", fw: Framework{Name: "fastify", Port: 3000, BuildCommand: "npm run build", StartCommand: "node dist/main.js"}},
		{name: "bun", fw: Framework{Name: "bun", Port: 3000, StartCommand: "bun run src/index.ts"}},
		{name: "bun-elysia", fw: Framework{Name: "bun-elysia", Port: 3000, StartCommand: "bun run src/index.ts"}},
		{name: "bun-hono", fw: Framework{Name: "bun-hono", Port: 3000, StartCommand: "bun run src/index.ts"}},
		{name: "deno", fw: Framework{Name: "deno", Port: 8000, StartCommand: "deno run --allow-net main.ts"}},
		{name: "deno-fresh", fw: Framework{Name: "deno-fresh", Port: 8000, StartCommand: "deno run --allow-net main.ts"}},
		{name: "deno-oak", fw: Framework{Name: "deno-oak", Port: 8000, StartCommand: "deno run --allow-net main.ts"}},
		{name: "django", fw: Framework{Name: "django", Port: 8000, StartCommand: "gunicorn config.wsgi:application"}},
		{name: "fastapi", fw: Framework{Name: "fastapi", Port: 8000, StartCommand: "uvicorn main:app --host 0.0.0.0 --port 8000"}},
		{name: "flask", fw: Framework{Name: "flask", Port: 5000, StartCommand: "gunicorn app:app"}},
		{name: "go", fw: Framework{Name: "go", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-gin", fw: Framework{Name: "go-gin", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-echo", fw: Framework{Name: "go-echo", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-fiber", fw: Framework{Name: "go-fiber", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-std", fw: Framework{Name: "go-std", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "go-chi", fw: Framework{Name: "go-chi", Port: 8080, BuildCommand: "go build -o server .", StartCommand: "./server"}},
		{name: "rails", fw: Framework{Name: "rails", Port: 3000, BuildCommand: "bundle install", StartCommand: "bundle exec rails server -b 0.0.0.0"}},
		{name: "sinatra", fw: Framework{Name: "sinatra", Port: 4567, BuildCommand: "bundle install", StartCommand: "ruby app.rb"}},
		{name: "laravel", fw: Framework{Name: "laravel", Port: 8000, BuildCommand: "composer install --no-dev", StartCommand: "php artisan serve --host=0.0.0.0 --port=8000"}},
		{name: "wordpress", fw: Framework{Name: "wordpress", Port: 80, StartCommand: "apache2-foreground"}},
		{name: "symfony", fw: Framework{Name: "symfony", Port: 8000, StartCommand: "apache2-foreground"}},
		{name: "slim", fw: Framework{Name: "slim", Port: 8000, StartCommand: "apache2-foreground"}},
		{name: "php", fw: Framework{Name: "php", Port: 8000, StartCommand: "php -S 0.0.0.0:8000"}},
		{name: "spring-boot", fw: Framework{Name: "spring-boot", Port: 8080, BuildCommand: "mvn -B package -DskipTests", StartCommand: "java -jar app.jar"}},
		{name: "quarkus", fw: Framework{Name: "quarkus", Port: 8080, BuildCommand: "./gradlew build -x test", StartCommand: "java -jar app.jar"}},
		{name: "micronaut", fw: Framework{Name: "micronaut", Port: 8080, BuildCommand: "./gradlew build -x test", StartCommand: "java -jar app.jar"}},
		{name: "maven", fw: Framework{Name: "maven", Port: 8080, BuildCommand: "mvn -B package -DskipTests", StartCommand: "java -jar app.jar"}},
		{name: "gradle", fw: Framework{Name: "gradle", Port: 8080, BuildCommand: "./gradlew build -x test", StartCommand: "java -jar app.jar"}},
		{name: "aspnet-core", fw: Framework{Name: "aspnet-core", Port: 8080, BuildCommand: "dotnet publish -c Release -o /app/publish", StartCommand: "dotnet /app/publish/app.dll"}},
		{name: "blazor", fw: Framework{Name: "blazor", Port: 8080, BuildCommand: "dotnet publish -c Release -o /app/publish", StartCommand: "dotnet /app/publish/app.dll"}},
		{name: "dotnet-worker", fw: Framework{Name: "dotnet-worker", Port: 8080, BuildCommand: "dotnet publish -c Release -o /app/publish", StartCommand: "dotnet /app/publish/app.dll"}},
		{name: "rust", fw: Framework{Name: "rust", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "rust-generic", fw: Framework{Name: "rust-generic", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "rust-actix", fw: Framework{Name: "rust-actix", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "rust-axum", fw: Framework{Name: "rust-axum", Port: 8080, BuildCommand: "cargo build --release", StartCommand: "./target/release/app"}},
		{name: "elixir-phoenix", fw: Framework{Name: "elixir-phoenix", Port: 4000, BuildCommand: "mix release", StartCommand: "./bin/server"}},
		{name: "static", fw: Framework{Name: "static", Port: 0}},
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
	fw := &Framework{Name: "unknown-thing", Port: 8080}
	_, err := Plan(fw)
	if err == nil {
		t.Fatal("expected error for unknown framework, got nil")
	}
	if !errors.Is(err, ErrNotDetected) {
		t.Errorf("expected ErrNotDetected, got %v", err)
	}
}

func TestPlan_NilFramework(t *testing.T) {
	_, err := Plan(nil)
	if err == nil {
		t.Fatal("expected error for nil framework, got nil")
	}
}
