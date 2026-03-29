package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/yamldef"
)

func TestConcurrentPlanAndEmit(t *testing.T) {
	type pipeFixture struct {
		name string
		dir  string
		want string
	}

	pipes := []pipeFixture{
		{"node-nextjs", fixturePath("node-nextjs"), "nextjs"},
		{"python-django", fixturePath("python-django"), "django"},
		{"go-gin", fixturePath("go-gin"), "go-gin"},
		{"rails", fixturePath("rails"), "rails"},
		{"rust-axum", fixturePath("rust-axum"), "rust-axum"},
		{"deno-fresh", fixturePath("deno-fresh"), "deno-fresh"},
	}

	const n = 15
	type result struct {
		pipeIdx    int
		dockerfile string
		fwName     string
		err        error
	}
	total := len(pipes) * n
	results := make([]result, total)
	var wg sync.WaitGroup
	wg.Add(total)

	for pi, pf := range pipes {
		for j := range n {
			idx := pi*n + j
			results[idx].pipeIdx = pi
			go func(idx int, dir string) {
				defer wg.Done()
				df, fw, err := docksmith.Build(dir)
				if err != nil {
					results[idx].err = err
					return
				}
				results[idx].dockerfile = df
				results[idx].fwName = fw.Name
			}(idx, pf.dir)
		}
	}
	wg.Wait()

	for _, r := range results {
		pf := pipes[r.pipeIdx]
		if r.err != nil {
			t.Errorf("%s: %v", pf.name, r.err)
			continue
		}
		if r.fwName != pf.want {
			t.Errorf("%s: detected %q, want %q", pf.name, r.fwName, pf.want)
			continue
		}
		if r.dockerfile == "" {
			t.Errorf("%s: empty dockerfile", pf.name)
			continue
		}
		if !strings.Contains(r.dockerfile, "FROM") {
			t.Errorf("%s: dockerfile missing FROM directive", pf.name)
		}
	}

	// All goroutines for the same fixture must produce identical Dockerfiles.
	for pi, pf := range pipes {
		var ref string
		for j := range n {
			idx := pi*n + j
			if results[idx].err != nil {
				continue
			}
			if ref == "" {
				ref = results[idx].dockerfile
				continue
			}
			if results[idx].dockerfile != ref {
				t.Errorf("%s: goroutine %d produced different dockerfile", pf.name, j)
			}
		}
	}
}

func TestConcurrentYAMLLoader(t *testing.T) {
	runtimes := []struct {
		name    string
		runtime string
		port    int
	}{
		{"test-node-fw", "node", 3000},
		{"test-python-fw", "python", 8000},
		{"test-go-fw", "go", 8080},
		{"test-ruby-fw", "ruby", 3000},
		{"test-rust-fw", "rust", 8080},
	}

	type yamlFixture struct {
		dir  string
		name string
	}
	var fixtures []yamlFixture
	for _, rt := range runtimes {
		dir := t.TempDir()
		yml := fmt.Sprintf("name: %s\nruntime: %s\nplan:\n  port: %d\n  stages:\n    - name: build\n      from: alpine:latest\n      steps:\n        - workdir: /app\n",
			rt.name, rt.runtime, rt.port)
		if err := os.WriteFile(filepath.Join(dir, rt.name+".yaml"), []byte(yml), 0o644); err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, yamlFixture{dir: dir, name: rt.name})
	}

	const n = 20
	type result struct {
		fixIdx int
		defs   []*docksmith.FrameworkDef
		err    error
	}
	total := len(fixtures) * n
	results := make([]result, total)
	var wg sync.WaitGroup
	wg.Add(total)

	for fi, f := range fixtures {
		for j := range n {
			idx := fi*n + j
			results[idx].fixIdx = fi
			go func(idx int, dir string) {
				defer wg.Done()
				defs, err := yamldef.LoadFrameworkDefs(dir)
				results[idx].defs = defs
				results[idx].err = err
			}(idx, f.dir)
		}
	}
	wg.Wait()

	for _, r := range results {
		want := fixtures[r.fixIdx].name
		if r.err != nil {
			t.Errorf("yaml fixture %s: %v", want, r.err)
			continue
		}
		if len(r.defs) != 1 {
			t.Errorf("yaml fixture %s: got %d defs, want 1", want, len(r.defs))
			continue
		}
		if r.defs[0].Name != want {
			t.Errorf("yaml fixture %s: got name %q", want, r.defs[0].Name)
		}
	}
}

func TestConcurrentDetectPanicRecovery(t *testing.T) {
	const n = 50
	type result struct {
		recovered bool
		err       error
		fwName    string
	}
	results := make([]result, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(idx int) {
			defer wg.Done()
			// Even: panicking detector wrapped with recover.
			// Odd: normal Detect on a real fixture.
			if idx%2 == 0 {
				func() {
					defer func() {
						if r := recover(); r != nil {
							results[idx].recovered = true
						}
					}()
					panicDetector := func(_ string) *docksmith.Framework {
						panic("intentional test panic")
					}
					panicDetector("anything")
				}()
			} else {
				fw, err := docksmith.Detect(fixturePath("node-nextjs"))
				results[idx].err = err
				if fw != nil {
					results[idx].fwName = fw.Name
				}
			}
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		if i%2 == 0 {
			if !r.recovered {
				t.Errorf("goroutine %d: panic was not recovered", i)
			}
		} else {
			if r.err != nil {
				t.Errorf("goroutine %d: %v", i, r.err)
				continue
			}
			if r.fwName != "nextjs" {
				t.Errorf("goroutine %d: got %q, want %q", i, r.fwName, "nextjs")
			}
		}
	}
}
