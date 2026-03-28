package docksmith

import (
	"sync"
	"testing"
)

func TestConcurrentDetect(t *testing.T) {
	const n = 100
	var wg sync.WaitGroup
	results := make([]*Framework, n)
	errs := make([]error, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = Detect("testdata/fixtures/node-nextjs")
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if results[i] == nil {
			t.Fatalf("goroutine %d: nil framework", i)
		}
		if results[i].Name != results[0].Name {
			t.Fatalf("goroutine %d: got %q, want %q", i, results[i].Name, results[0].Name)
		}
	}
}

func TestConcurrentBuild(t *testing.T) {
	const n = 50
	var wg sync.WaitGroup
	dockerfiles := make([]string, n)
	errs := make([]error, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			dockerfiles[idx], _, errs[idx] = Build("testdata/fixtures/python-django")
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if dockerfiles[i] == "" {
			t.Fatalf("goroutine %d: empty dockerfile", i)
		}
	}
}

func TestConcurrentEmitDockerfile(t *testing.T) {
	plan := &BuildPlan{
		Framework: "test",
		Expose:    8080,
		Stages: []Stage{{
			Name: "build",
			From: "node:20-alpine",
			Steps: []Step{
				{Type: StepWorkdir, Args: []string{"/app"}},
				{Type: StepCopy, Args: []string{".", "."}},
				{Type: StepRun, Args: []string{"npm install"}},
				{Type: StepCmd, Args: []string{"npm start"}},
			},
		}},
	}

	const n = 100
	var wg sync.WaitGroup
	results := make([]string, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			results[idx] = EmitDockerfile(plan)
		}(i)
	}
	wg.Wait()

	for i := range n {
		if results[i] != results[0] {
			t.Fatalf("goroutine %d: output differs from goroutine 0", i)
		}
	}
}

func TestConcurrentLoadConfig(t *testing.T) {
	const n = 50
	var wg sync.WaitGroup
	configs := make([]*Config, n)
	errs := make([]error, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			configs[idx], errs[idx] = LoadConfig("testdata/fixtures/empty-dir")
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if configs[i] != nil {
			t.Fatalf("goroutine %d: expected nil config", i)
		}
	}
}
