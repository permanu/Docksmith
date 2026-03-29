package detect

import (
	"encoding/json"
	"github.com/permanu/docksmith/core"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Bun detectors must run before Node — bun projects also have package.json.
	RegisterDetector("bun-elysia", detectBunElysia)
	RegisterDetector("bun-hono", detectBunHono)
	RegisterDetector("bun-plain", detectBunPlain)
}

func detectBunVersion(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var p struct {
			Engines struct {
				Bun string `json:"bun"`
			} `json:"engines"`
		}
		if err := json.Unmarshal(data, &p); err == nil && p.Engines.Bun != "" {
			return cleanBunVersion(p.Engines.Bun)
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".bun-version")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return cleanBunVersion(v)
		}
	}
	return "1"
}

func hasBunLockfile(dir string) bool {
	return hasFile(dir, "bun.lockb") || hasFile(dir, "bun.lock")
}

// cleanBunVersion strips semver prefixes and returns major.minor (or just major).
func cleanBunVersion(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, ">=^~v")
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return s
}

func detectStartScript(dir, fallback string) string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return fallback
	}
	var p struct {
		Scripts struct {
			Start string `json:"start"`
		} `json:"scripts"`
	}
	if err := json.Unmarshal(data, &p); err != nil || p.Scripts.Start == "" {
		return fallback
	}
	s := p.Scripts.Start
	if strings.HasPrefix(s, "bun ") {
		return s
	}
	return "bun run " + s
}

func detectBunElysia(dir string) *core.Framework {
	pkg := filepath.Join(dir, "package.json")
	if !hasFile(dir, "package.json") || !hasBunLockfile(dir) {
		return nil
	}
	if !fileContains(pkg, "elysia") {
		return nil
	}
	return &core.Framework{
		Name:         "bun-elysia",
		BuildCommand: "bun install --frozen-lockfile",
		StartCommand: detectStartScript(dir, "bun run src/index.ts"),
		Port:         3000,
		BunVersion:   detectBunVersion(dir),
	}
}

func detectBunHono(dir string) *core.Framework {
	pkg := filepath.Join(dir, "package.json")
	if !hasFile(dir, "package.json") || !hasBunLockfile(dir) {
		return nil
	}
	if !fileContains(pkg, "hono") {
		return nil
	}
	return &core.Framework{
		Name:         "bun-hono",
		BuildCommand: "bun install --frozen-lockfile",
		StartCommand: detectStartScript(dir, "bun run src/index.ts"),
		Port:         3000,
		BunVersion:   detectBunVersion(dir),
	}
}

func detectBunPlain(dir string) *core.Framework {
	if !hasBunLockfile(dir) {
		return nil
	}
	if hasFile(dir, "package-lock.json") {
		return nil
	}
	nodeConfigs := []string{
		"next.config.js", "next.config.mjs", "next.config.ts",
		"nuxt.config.ts", "nuxt.config.js",
		"svelte.config.js", "svelte.config.ts",
		"astro.config.mjs", "astro.config.ts",
		"remix.config.js", "remix.config.ts",
		"gatsby-config.js", "gatsby-config.ts",
		"vite.config.js", "vite.config.ts", "vite.config.mjs",
		"angular.json", "vue.config.js", "nest-cli.json",
	}
	for _, cfg := range nodeConfigs {
		if hasFile(dir, cfg) {
			return nil
		}
	}
	return &core.Framework{
		Name:         "bun",
		BuildCommand: "bun install --frozen-lockfile",
		StartCommand: detectStartScript(dir, "bun run index.ts"),
		Port:         3000,
		BunVersion:   detectBunVersion(dir),
	}
}
