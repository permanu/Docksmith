package detect

import (
	"path/filepath"
	"testing"

	"github.com/permanu/docksmith/core"
)

func TestDetectNextJS(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "node-nextjs")
	fw := detectNextJS(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "nextjs" {
		t.Errorf("Name = %q, want %q", fw.Name, "nextjs")
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.NodeVersion == "" {
		t.Error("NodeVersion is empty")
	}
}

func TestDetectNuxt(t *testing.T) {
	fw := detectNuxt(filepath.Join("testdata", "fixtures", "node-nuxt"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "nuxt" {
		t.Errorf("Name = %q, want %q", fw.Name, "nuxt")
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
}

func TestDetectSvelteKit(t *testing.T) {
	fw := detectSvelteKit(filepath.Join("testdata", "fixtures", "node-sveltekit"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "sveltekit" {
		t.Errorf("Name = %q, want %q", fw.Name, "sveltekit")
	}
}

func TestDetectAstro(t *testing.T) {
	fw := detectAstro(filepath.Join("testdata", "fixtures", "node-astro"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "astro" {
		t.Errorf("Name = %q, want %q", fw.Name, "astro")
	}
	if fw.Port != 4321 {
		t.Errorf("Port = %d, want 4321", fw.Port)
	}
}

func TestDetectRemix(t *testing.T) {
	fw := detectRemix(filepath.Join("testdata", "fixtures", "node-remix"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "remix" {
		t.Errorf("Name = %q, want %q", fw.Name, "remix")
	}
}

func TestDetectGatsby(t *testing.T) {
	fw := detectGatsby(filepath.Join("testdata", "fixtures", "node-gatsby"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "gatsby" {
		t.Errorf("Name = %q, want %q", fw.Name, "gatsby")
	}
	if fw.Port != 9000 {
		t.Errorf("Port = %d, want 9000", fw.Port)
	}
}

func TestDetectVite(t *testing.T) {
	fw := detectVite(filepath.Join("testdata", "fixtures", "node-vite"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "vite" {
		t.Errorf("Name = %q, want %q", fw.Name, "vite")
	}
}

func TestDetectCRA(t *testing.T) {
	fw := detectCRA(filepath.Join("testdata", "fixtures", "node-cra"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "create-react-app" {
		t.Errorf("Name = %q, want %q", fw.Name, "create-react-app")
	}
}

func TestDetectAngular(t *testing.T) {
	fw := detectAngular(filepath.Join("testdata", "fixtures", "node-angular"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "angular" {
		t.Errorf("Name = %q, want %q", fw.Name, "angular")
	}
	if fw.Port != 4200 {
		t.Errorf("Port = %d, want 4200", fw.Port)
	}
}

func TestDetectVueCLI(t *testing.T) {
	fw := detectVueCLI(filepath.Join("testdata", "fixtures", "node-vuecli"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "vue-cli" {
		t.Errorf("Name = %q, want %q", fw.Name, "vue-cli")
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
}

func TestDetectSolidStart(t *testing.T) {
	fw := detectSolidStart(filepath.Join("testdata", "fixtures", "node-solidstart"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "solidstart" {
		t.Errorf("Name = %q, want %q", fw.Name, "solidstart")
	}
}

func TestDetectNestJS(t *testing.T) {
	fw := detectNestJS(filepath.Join("testdata", "fixtures", "node-nestjs"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "nestjs" {
		t.Errorf("Name = %q, want %q", fw.Name, "nestjs")
	}
}

func TestDetectExpress(t *testing.T) {
	fw := detectExpress(filepath.Join("testdata", "fixtures", "node-express"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "express" {
		t.Errorf("Name = %q, want %q", fw.Name, "express")
	}
}

func TestDetectFastify(t *testing.T) {
	fw := detectFastify(filepath.Join("testdata", "fixtures", "node-fastify"))
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "fastify" {
		t.Errorf("Name = %q, want %q", fw.Name, "fastify")
	}
}

// Python project must not be detected as any Node framework.
func TestNodeDetectors_NoFalsePositive_Python(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "requirements.txt", "django==4.2\n")
	nodeWrite(t, dir, "manage.py", "#!/usr/bin/env python\n")

	checks := []struct {
		name string
		fn   func(string) *core.Framework
	}{
		{"nextjs", detectNextJS},
		{"nuxt", detectNuxt},
		{"sveltekit", detectSvelteKit},
		{"astro", detectAstro},
		{"remix", detectRemix},
		{"gatsby", detectGatsby},
		{"vite", detectVite},
		{"cra", detectCRA},
		{"angular", detectAngular},
		{"vuecli", detectVueCLI},
		{"solidstart", detectSolidStart},
		{"nestjs", detectNestJS},
		{"express", detectExpress},
		{"fastify", detectFastify},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if fw := c.fn(dir); fw != nil {
				t.Errorf("false positive: detected %q in Python project", fw.Name)
			}
		})
	}
}
