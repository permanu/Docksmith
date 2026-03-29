package integration_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/permanu/docksmith"
)

// fixtureExpect maps fixture dir name -> expected framework name.
var fixtureExpect = map[string]string{
	"node-nextjs":     "nextjs",
	"node-express":    "express",
	"node-nuxt":       "nuxt",
	"node-sveltekit":  "sveltekit",
	"node-remix":      "remix",
	"node-astro":      "astro",
	"python-django":   "django",
	"python-flask":    "flask",
	"python-fastapi":  "fastapi",
	"go-std-root":     "go",
	"go-gin":          "go-gin",
	"go-echo":         "go-echo",
	"go-fiber":        "go-fiber",
	"rust-actix":      "rust-actix",
	"rust-axum":       "rust-axum",
	"rails":           "rails",
	"laravel":         "laravel",
	"elixir-phoenix":  "elixir-phoenix",
	"deno-fresh":      "deno-fresh",
	"deno-oak":        "deno-oak",
	"node-angular":    "angular",
	"node-gatsby":     "gatsby",
	"node-cra":        "create-react-app",
	"node-vite":       "vite",
	"node-nestjs":     "nestjs",
	"node-fastify":    "fastify",
	"node-solidstart": "solidstart",
	"node-vuecli":     "vue-cli",
}

func TestConcurrentDetectStress(t *testing.T) {
	const goroutinesPerFixture = 4
	type result struct {
		fixture string
		want    string
		got     string
		err     error
	}

	fixtures := fixtureEntries(t)
	total := len(fixtures) * goroutinesPerFixture
	results := make([]result, total)
	var wg sync.WaitGroup
	wg.Add(total)

	for i, f := range fixtures {
		for j := range goroutinesPerFixture {
			idx := i*goroutinesPerFixture + j
			results[idx].fixture = f.name
			results[idx].want = f.want
			go func(idx int, dir string) {
				defer wg.Done()
				fw, err := docksmith.Detect(dir)
				if err != nil {
					results[idx].err = err
					return
				}
				results[idx].got = fw.Name
			}(idx, f.dir)
		}
	}
	wg.Wait()

	for _, r := range results {
		if r.err != nil {
			t.Errorf("fixture %s: %v", r.fixture, r.err)
			continue
		}
		if r.got != r.want {
			t.Errorf("fixture %s: got %q, want %q", r.fixture, r.got, r.want)
		}
	}
}

func TestConcurrentDetectWithConfig(t *testing.T) {
	type configFixture struct {
		dir    string
		opts   docksmith.DetectOptions
		wantRT string
	}

	cfgFixtures := []configFixture{
		{
			dir:    fixturePath("config-yaml"),
			opts:   docksmith.DetectOptions{},
			wantRT: "express", // runtime: node -> express
		},
		{
			dir:    fixturePath("config-json"),
			opts:   docksmith.DetectOptions{},
			wantRT: "flask", // runtime: python -> flask
		},
		{
			dir:    fixturePath("config-custom-name"),
			opts:   docksmith.DetectOptions{ConfigFileNames: []string{"deploy.yaml"}},
			wantRT: "rails", // runtime: ruby -> rails
		},
	}

	const n = 30
	type result struct {
		cfgIdx int
		err    error
		got    string
	}
	total := len(cfgFixtures) * n
	results := make([]result, total)
	var wg sync.WaitGroup
	wg.Add(total)

	for ci, cf := range cfgFixtures {
		for j := range n {
			idx := ci*n + j
			results[idx].cfgIdx = ci
			go func(idx int, dir string, opts docksmith.DetectOptions) {
				defer wg.Done()
				fw, err := docksmith.DetectWithOptions(dir, opts)
				if err != nil {
					results[idx].err = err
					return
				}
				results[idx].got = fw.Name
			}(idx, cf.dir, cf.opts)
		}
	}
	wg.Wait()

	for _, r := range results {
		want := cfgFixtures[r.cfgIdx].wantRT
		if r.err != nil {
			t.Errorf("config fixture %d: %v", r.cfgIdx, r.err)
			continue
		}
		if r.got != want {
			t.Errorf("config fixture %d: got %q, want %q", r.cfgIdx, r.got, want)
		}
	}
}

// --- helpers ---

type fixtureEntry struct {
	name string
	dir  string
	want string
}

func fixtureEntries(t *testing.T) []fixtureEntry {
	t.Helper()
	var out []fixtureEntry
	for name, want := range fixtureExpect {
		dir := fixturePath(name)
		if _, err := os.Stat(dir); err != nil {
			t.Logf("skipping missing fixture %s", name)
			continue
		}
		out = append(out, fixtureEntry{name: name, dir: dir, want: want})
	}
	if len(out) == 0 {
		t.Fatal("no fixtures found")
	}
	return out
}

func fixturePath(name string) string {
	return filepath.Join("..", "..", "testdata", "fixtures", name)
}
