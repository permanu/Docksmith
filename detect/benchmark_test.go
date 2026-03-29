package detect_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/permanu/docksmith/detect"
	"github.com/permanu/docksmith/yamldef"
)

var runtimeFixtures = []struct {
	name    string
	fixture string
}{
	{"node", "node-nextjs"},
	{"python", "python-django"},
	{"go", "go-std-root"},
	{"ruby", "rails"},
	{"php", "laravel"},
	{"rust", "rust-actix"},
	{"elixir", "elixir-phoenix"},
	{"deno", "deno-plain"},
	{"bun", "node-astro"},
}

func BenchmarkDetect(b *testing.B) {
	for _, rt := range runtimeFixtures {
		dir := filepath.Join("testdata", "fixtures", rt.fixture)
		if _, err := os.Stat(dir); err != nil {
			b.Logf("skipping %s: fixture %s not found", rt.name, rt.fixture)
			continue
		}
		b.Run(rt.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_, _ = detect.Detect(dir)
			}
		})
	}
}

func BenchmarkDetectLargeProject(b *testing.B) {
	dir := b.TempDir()
	setupLargeProject(b, dir)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = detect.Detect(dir)
	}
}

func BenchmarkDetectWithConfig(b *testing.B) {
	dir := filepath.Join("testdata", "fixtures", "config-priority-toml")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("fixture not found: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = detect.Detect(dir)
	}
}

func BenchmarkDetectYAMLFramework(b *testing.B) {
	frameworksDir := filepath.Join("..", "frameworks")
	defs, err := yamldef.LoadFrameworkDefs(frameworksDir)
	if err != nil {
		b.Fatalf("load YAML defs: %v", err)
	}
	if len(defs) == 0 {
		b.Skip("no YAML framework definitions found")
	}

	dir := filepath.Join("testdata", "fixtures", "node-nextjs")
	if _, err := os.Stat(dir); err != nil {
		b.Skipf("fixture not found: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, def := range defs {
			yamldef.EvalDetectRules(dir, def.Detect)
		}
	}
}

func setupLargeProject(tb testing.TB, dir string) {
	tb.Helper()

	writeFile(tb, dir, "package.json", `{"name":"bench","dependencies":{"express":"4.18.0"}}`)
	writeFile(tb, dir, "go.mod", "module bench\n\ngo 1.22\n")

	// ~1MB package-lock.json
	chunk := `"fake-pkg-%d":{"version":"1.0.%d","resolved":"https://registry.npmjs.org/fake/-/fake-1.0.%d.tgz"},`
	var lockBuf strings.Builder
	lockBuf.WriteString(`{"name":"bench","lockfileVersion":3,"packages":{`)
	for i := range 4000 {
		fmt.Fprintf(&lockBuf, chunk, i, i, i)
	}
	lockBuf.WriteString(`"end":{"version":"0.0.0"}}}`)
	writeFile(tb, dir, "package-lock.json", lockBuf.String())

	// ~500KB go.sum
	var sumBuf strings.Builder
	for i := range 5000 {
		fmt.Fprintf(&sumBuf, "github.com/fake/pkg%d v1.0.0 h1:%s=\n", i, strings.Repeat("A", 44))
		fmt.Fprintf(&sumBuf, "github.com/fake/pkg%d v1.0.0/go.mod h1:%s=\n", i, strings.Repeat("B", 44))
	}
	writeFile(tb, dir, "go.sum", sumBuf.String())

	// 50+ files across nested directories.
	for i := range 10 {
		nested := filepath.Join(dir, fmt.Sprintf("src/level1/level2/pkg%d", i))
		if err := os.MkdirAll(nested, 0o755); err != nil {
			tb.Fatal(err)
		}
		for j := range 5 {
			name := fmt.Sprintf("file%d.go", j)
			writeFile(tb, nested, name, fmt.Sprintf("package pkg%d\n", i))
		}
	}

	writeFile(tb, dir, "requirements.txt", "flask==3.0.0\nrequests==2.31.0\n")
}

func writeFile(tb testing.TB, dir, name, content string) {
	tb.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
}
