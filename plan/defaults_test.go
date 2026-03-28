package plan

import "testing"

func TestFrameworkDefaults(t *testing.T) {
	known := []struct {
		name  string
		build string
		start string
	}{
		{"nextjs", "npm run build", "npm start"},
		{"next", "npm run build", "npm start"},
		{"nuxt", "npm run build", "npm start"},
		{"sveltekit", "npm run build", "node build"},
		{"astro", "npm run build", "node ./dist/server/entry.mjs"},
		{"remix", "npm run build", "npm start"},
		{"vite", "npm run build", "npx serve -s dist"},
		{"express", "npm run build", "node dist/main.js"},
		{"fastify", "npm run build", "node dist/main.js"},
		{"nestjs", "npm run build", "node dist/main.js"},
		{"django", "pip install -r requirements.txt", "gunicorn config.wsgi:application --bind 0.0.0.0:8000"},
		{"fastapi", "pip install -r requirements.txt", "uvicorn main:app --host 0.0.0.0 --port 8000"},
		{"flask", "pip install -r requirements.txt", "gunicorn app:app --bind 0.0.0.0:5000"},
		{"go", "go build -o server .", "./server"},
		{"go-gin", "go build -o server .", "./server"},
		{"go-echo", "go build -o server .", "./server"},
		{"go-fiber", "go build -o server .", "./server"},
		{"go-std", "go build -o server .", "./server"},
		{"rails", "bundle install", "bundle exec rails server -b 0.0.0.0"},
		{"laravel", "composer install --no-dev", "php artisan serve --host=0.0.0.0 --port=8000"},
		{"spring-boot", "mvn -B package -DskipTests", "java -jar app.jar"},
		{"aspnet-core", "dotnet publish -c Release -o /app/publish", "dotnet /app/publish/app.dll"},
		{"rust", "cargo build --release", "./target/release/app"},
		{"elixir-phoenix", "mix release", "./bin/server"},
		{"bun", "bun install", "bun run src/index.ts"},
		{"deno", "", "deno run --allow-net --allow-env main.ts"},
		{"static", "npm run build", "npx serve -s dist"},
	}

	for _, tc := range known {
		t.Run(tc.name, func(t *testing.T) {
			build, start := FrameworkDefaults(tc.name)
			if build != tc.build {
				t.Errorf("build: got %q, want %q", build, tc.build)
			}
			if start != tc.start {
				t.Errorf("start: got %q, want %q", start, tc.start)
			}
			if start == "" && tc.start != "" {
				t.Errorf("start command must not be empty for %q", tc.name)
			}
		})
	}
}

func TestFrameworkDefaultsUnknown(t *testing.T) {
	for _, name := range []string{"unknown-framework", "totally-fake", ""} {
		build, start := FrameworkDefaults(name)
		if build != "" || start != "" {
			t.Errorf("FrameworkDefaults(%q) = (%q, %q), want empty strings", name, build, start)
		}
	}
}
