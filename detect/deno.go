package detect

import (
	"github.com/permanu/docksmith/core"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Deno before Node — deno.json is unambiguous, Node never uses it.
	// Register plain first so specific detectors (fresh, oak) end up ahead of it.
	RegisterDetectorBefore("node", "deno", detectDenoPlain)
	RegisterDetectorBefore("deno", "deno-oak", detectDenoOak)
	RegisterDetectorBefore("deno-oak", "deno-fresh", detectDenoFresh)
}

func detectDenoVersion(dir string) string {
	for _, name := range []string{"deno.json", "deno.jsonc"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var d map[string]any
		if err := json.Unmarshal(data, &d); err == nil {
			if v, ok := d["version"].(string); ok && v != "" {
				return extractMajorVersion(v)
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".dvmrc")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return extractMajorVersion(v)
		}
	}
	return "2"
}

func findDenoEntrypoint(dir string) string {
	for _, name := range []string{"main.ts", "main.tsx", "mod.ts", "src/main.ts", "app.ts", "server.ts"} {
		if hasFile(dir, name) {
			return name
		}
	}
	return "main.ts"
}

func detectDenoFresh(dir string) *core.Framework {
	denoJSON := filepath.Join(dir, "deno.json")
	if hasFile(dir, "deno.json") && fileContains(denoJSON, "$fresh") {
		return &core.Framework{
			Name:         "deno-fresh",
			StartCommand: "deno run -A main.ts",
			Port:         8000,
			DenoVersion:  detectDenoVersion(dir),
		}
	}
	if hasFile(dir, "fresh.config.ts") {
		return &core.Framework{
			Name:         "deno-fresh",
			StartCommand: "deno run -A main.ts",
			Port:         8000,
			DenoVersion:  detectDenoVersion(dir),
		}
	}
	return nil
}

func detectDenoOak(dir string) *core.Framework {
	if !hasFile(dir, "deno.json") {
		return nil
	}
	for _, entry := range []string{"main.ts", "main.tsx", "mod.ts", "src/main.ts", "app.ts", "server.ts"} {
		if fileContains(filepath.Join(dir, entry), "oak") {
			return &core.Framework{
				Name:         "deno-oak",
				StartCommand: "deno run --allow-net --allow-read " + entry,
				Port:         8000,
				DenoVersion:  detectDenoVersion(dir),
			}
		}
	}
	if fileContains(filepath.Join(dir, "deno.json"), "oak") {
		entrypoint := findDenoEntrypoint(dir)
		return &core.Framework{
			Name:         "deno-oak",
			StartCommand: "deno run --allow-net --allow-read " + entrypoint,
			Port:         8000,
			DenoVersion:  detectDenoVersion(dir),
		}
	}
	return nil
}

func detectDenoPlain(dir string) *core.Framework {
	if !hasFile(dir, "deno.json") && !hasFile(dir, "deno.jsonc") {
		return nil
	}
	entrypoint := findDenoEntrypoint(dir)
	return &core.Framework{
		Name:         "deno",
		StartCommand: "deno run -A " + entrypoint,
		Port:         8000,
		DenoVersion:  detectDenoVersion(dir),
	}
}
