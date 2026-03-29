package integration_test

import (
	"sync"
	"testing"

	"github.com/permanu/docksmith"
)

func TestConcurrentDetect(t *testing.T) {
	const n = 100
	var wg sync.WaitGroup
	results := make([]*docksmith.Framework, n)
	errs := make([]error, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = docksmith.Detect("../../testdata/fixtures/node-nextjs")
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
			dockerfiles[idx], _, errs[idx] = docksmith.Build("../../testdata/fixtures/python-django")
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
	plan := &docksmith.BuildPlan{
		Framework: "test",
		Expose:    8080,
		Stages: []docksmith.Stage{{
			Name: "build",
			From: "node:20-alpine",
			Steps: []docksmith.Step{
				{Type: docksmith.StepWorkdir, Args: []string{"/app"}},
				{Type: docksmith.StepCopy, Args: []string{".", "."}},
				{Type: docksmith.StepRun, Args: []string{"npm install"}},
				{Type: docksmith.StepCmd, Args: []string{"npm start"}},
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
			results[idx] = docksmith.EmitDockerfile(plan)
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
	configs := make([]*docksmith.Config, n)
	errs := make([]error, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			configs[idx], errs[idx] = docksmith.LoadConfig("../../testdata/fixtures/empty-dir")
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
