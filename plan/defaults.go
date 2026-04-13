package plan

// FrameworkDefaults returns fallback build and start commands for a framework name.
// Used by the CLI for display hints and by config merging when the user omits commands.
// Returns ("", "") for unrecognized names — callers should check both values.
func FrameworkDefaults(name string) (buildCmd, startCmd string) {
	switch name {
	case "nextjs", "next":
		return "npm run build", "npm start"
	case "nuxt":
		return "npm run build", "npm start"
	case "vite", "create-react-app", "vue-cli", "static":
		return "npm run build", "npx serve -s dist"
	case "sveltekit":
		return "npm run build", "node build"
	case "astro":
		return "npm run build", "node ./dist/server/entry.mjs"
	case "remix":
		return "npm run build", "npm start"
	case "express", "fastify", "nestjs":
		return "npm run build", "node dist/main.js"
	case "django":
		return "pip install -r requirements.txt", "gunicorn config.wsgi:application --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2"
	case "fastapi":
		return "pip install -r requirements.txt", "gunicorn main:app --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} -k uvicorn.workers.UvicornWorker"
	case "flask":
		return "pip install -r requirements.txt", "gunicorn app:app --bind 0.0.0.0:${PORT:-8000} --workers ${WEB_CONCURRENCY:-2} --threads 2"
	case "go", "go-gin", "go-echo", "go-fiber", "go-chi", "go-std":
		return "go build -o server .", "./server"
	case "spring-boot", "maven":
		return "mvn -B package -DskipTests", "java -jar app.jar"
	case "gradle", "quarkus", "micronaut":
		return "./gradlew build -x test", "java -jar app.jar"
	case "rails":
		return "bundle install", "bundle exec rails server -b 0.0.0.0"
	case "sinatra":
		return "bundle install", "ruby app.rb"
	case "laravel":
		return "composer install --no-dev", "php artisan serve --host=0.0.0.0 --port=8000"
	case "php", "symfony", "slim":
		return "", "php -S 0.0.0.0:8000"
	case "rust", "rust-actix", "rust-axum", "rust-rocket", "rust-generic":
		return "cargo build --release", "./target/release/app"
	case "elixir-phoenix":
		return "mix release", "./bin/server"
	case "deno", "deno-fresh", "deno-oak":
		return "", "deno run --allow-net --allow-env main.ts"
	case "bun", "bun-elysia", "bun-hono":
		return "bun install", "bun run src/index.ts"
	case "aspnet-core", "blazor":
		return "dotnet publish -c Release -o /app/publish", "dotnet /app/publish/app.dll"
	case "dotnet-worker":
		return "dotnet publish -c Release -o /app/publish", "dotnet /app/publish/app.dll"
	default:
		return "", ""
	}
}
