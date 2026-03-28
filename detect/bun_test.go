package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBunVersion_EnginesField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"engines":{"bun":">=1.1.0"}}`)
	if got := detectBunVersion(dir); got != "1.1" {
		t.Errorf("detectBunVersion = %q, want %q", got, "1.1")
	}
}

func TestDetectBunVersion_BunVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".bun-version", "1.2.3\n")
	if got := detectBunVersion(dir); got != "1.2" {
		t.Errorf("detectBunVersion = %q, want %q", got, "1.2")
	}
}

func TestDetectBunVersion_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectBunVersion(dir); got != "1" {
		t.Errorf("detectBunVersion = %q, want %q", got, "1")
	}
}

func TestHasBunLockfile_LockB(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	if !hasBunLockfile(dir) {
		t.Error("hasBunLockfile = false, want true for bun.lockb")
	}
}

func TestHasBunLockfile_LockText(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lock", "")
	if !hasBunLockfile(dir) {
		t.Error("hasBunLockfile = false, want true for bun.lock")
	}
}

func TestHasBunLockfile_Absent(t *testing.T) {
	dir := t.TempDir()
	if hasBunLockfile(dir) {
		t.Error("hasBunLockfile = true, want false when no lockfile present")
	}
}

func TestDetectBunElysia(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{"dependencies":{"elysia":"^1.0.0"}}`)

	fw := detectBunElysia(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "bun-elysia" {
		t.Errorf("Name = %q, want %q", fw.Name, "bun-elysia")
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.BunVersion == "" {
		t.Error("BunVersion is empty")
	}
	if fw.BuildCommand != "bun install --frozen-lockfile" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectBunElysia_NoLockfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"elysia":"^1.0.0"}}`)
	if fw := detectBunElysia(dir); fw != nil {
		t.Errorf("got %q, want nil without lockfile", fw.Name)
	}
}

func TestDetectBunElysia_NoElysiaInPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{"dependencies":{"express":"^4.0.0"}}`)
	if fw := detectBunElysia(dir); fw != nil {
		t.Errorf("got %q, want nil without elysia dep", fw.Name)
	}
}

func TestDetectBunHono(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{"dependencies":{"hono":"^4.0.0"}}`)

	fw := detectBunHono(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "bun-hono" {
		t.Errorf("Name = %q, want %q", fw.Name, "bun-hono")
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.BuildCommand != "bun install --frozen-lockfile" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectBunHono_NoLockfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"hono":"^4.0.0"}}`)
	if fw := detectBunHono(dir); fw != nil {
		t.Errorf("got %q, want nil without lockfile", fw.Name)
	}
}

func TestDetectBunHono_NoHonoInPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{"dependencies":{"express":"^4.0.0"}}`)
	if fw := detectBunHono(dir); fw != nil {
		t.Errorf("got %q, want nil without hono dep", fw.Name)
	}
}

func TestDetectBunPlain(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "index.ts", `console.log("hello")`)

	fw := detectBunPlain(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "bun" {
		t.Errorf("Name = %q, want %q", fw.Name, "bun")
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.BuildCommand != "bun install --frozen-lockfile" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectBunPlain_NoLockfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", "")
	if fw := detectBunPlain(dir); fw != nil {
		t.Errorf("got %q, want nil without bun lockfile", fw.Name)
	}
}

func TestDetectBunPlain_NodeLockfileTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package-lock.json", "{}")
	if fw := detectBunPlain(dir); fw != nil {
		t.Errorf("got %q, want nil when package-lock.json present", fw.Name)
	}
}

func TestDetectBunPlain_NodeFrameworkConfigSkipped(t *testing.T) {
	nodeConfigs := []string{
		"next.config.js",
		"next.config.mjs",
		"next.config.ts",
		"nuxt.config.ts",
		"vite.config.js",
		"angular.json",
	}
	for _, cfg := range nodeConfigs {
		t.Run(cfg, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "bun.lockb", "")
			writeFile(t, dir, cfg, "")
			if fw := detectBunPlain(dir); fw != nil {
				t.Errorf("config=%q: got %q, want nil", cfg, fw.Name)
			}
		})
	}
}

func TestDetectBunPlain_CustomStartScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{"scripts":{"start":"bun run src/server.ts"}}`)

	fw := detectBunPlain(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.StartCommand != "bun run src/server.ts" {
		t.Errorf("StartCommand = %q, want %q", fw.StartCommand, "bun run src/server.ts")
	}
}

func TestDetectBunElysia_UsesStartScriptFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{
		"dependencies":{"elysia":"^1.0.0"},
		"scripts":{"start":"bun run src/app.ts"}
	}`)

	fw := detectBunElysia(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.StartCommand != "bun run src/app.ts" {
		t.Errorf("StartCommand = %q, want %q", fw.StartCommand, "bun run src/app.ts")
	}
}

func TestDetectBunHono_UsesStartScriptFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	writeFile(t, dir, "package.json", `{
		"dependencies":{"hono":"^4.0.0"},
		"scripts":{"start":"bun run src/index.ts"}
	}`)

	fw := detectBunHono(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.StartCommand != "bun run src/index.ts" {
		t.Errorf("StartCommand = %q, want %q", fw.StartCommand, "bun run src/index.ts")
	}
}

// writeFile creates a file in dir with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
