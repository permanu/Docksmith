package detect

import (
	"encoding/json"
	"github.com/permanu/docksmith/core"
	"os"
	"path/filepath"
)

func init() {
	// Registered in reverse priority order — RegisterDetector prepends each entry,
	// so the last call here becomes the highest-priority detector.
	// Final order matches the source: nextjs first, fastify last.
	RegisterDetector("fastify", detectFastify)
	RegisterDetector("express", detectExpress)
	RegisterDetector("nestjs", detectNestJS)
	RegisterDetector("solidstart", detectSolidStart)
	RegisterDetector("vue-cli", detectVueCLI)
	RegisterDetector("angular", detectAngular)
	RegisterDetector("create-react-app", detectCRA)
	RegisterDetector("vite", detectVite)
	RegisterDetector("gatsby", detectGatsby)
	RegisterDetector("remix", detectRemix)
	RegisterDetector("astro", detectAstro)
	RegisterDetector("sveltekit", detectSvelteKit)
	RegisterDetector("nuxt", detectNuxt)
	RegisterDetector("nextjs", detectNextJS)
}

func detectNextJS(dir string) *core.Framework {
	if hasFile(dir, "next.config.js") || hasFile(dir, "next.config.mjs") || hasFile(dir, "next.config.ts") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "nextjs", PMRunBuild(pm), PMRunStart(pm), 3000, ".next")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), `"next"`) {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "nextjs", PMRunBuild(pm), PMRunStart(pm), 3000, ".next")
	}
	return nil
}

func detectNuxt(dir string) *core.Framework {
	if hasFile(dir, "nuxt.config.ts") || hasFile(dir, "nuxt.config.js") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "nuxt", PMRunBuild(pm), "node .output/server/index.mjs", 3000, ".output")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), `"nuxt"`) {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "nuxt", PMRunBuild(pm), "node .output/server/index.mjs", 3000, ".output")
	}
	return nil
}

func detectSvelteKit(dir string) *core.Framework {
	if hasFile(dir, "svelte.config.js") || hasFile(dir, "svelte.config.ts") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "sveltekit", PMRunBuild(pm), "node build", 3000, "build")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), "@sveltejs/kit") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "sveltekit", PMRunBuild(pm), "node build", 3000, "build")
	}
	return nil
}

func detectAstro(dir string) *core.Framework {
	if hasFile(dir, "astro.config.mjs") || hasFile(dir, "astro.config.ts") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "astro", PMRunBuild(pm), "node ./dist/server/entry.mjs", 4321, "dist")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), `"astro"`) {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "astro", PMRunBuild(pm), "node ./dist/server/entry.mjs", 4321, "dist")
	}
	return nil
}

func detectRemix(dir string) *core.Framework {
	if hasFile(dir, "remix.config.js") || hasFile(dir, "remix.config.ts") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "remix", PMRunBuild(pm), PMRunStart(pm), 3000, "")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), "@remix-run") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "remix", PMRunBuild(pm), PMRunStart(pm), 3000, "")
	}
	return nil
}

func detectGatsby(dir string) *core.Framework {
	gatsbyStart := func(pm string) string {
		switch pm {
		case "pnpm":
			return "pnpm exec gatsby serve"
		case "bun":
			return "bunx gatsby serve"
		default:
			return "npx gatsby serve"
		}
	}
	if hasFile(dir, "gatsby-config.js") || hasFile(dir, "gatsby-config.ts") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "gatsby", PMRunBuild(pm), gatsbyStart(pm), 9000, "public")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), `"gatsby"`) {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "gatsby", PMRunBuild(pm), gatsbyStart(pm), 9000, "public")
	}
	return nil
}

func detectVite(dir string) *core.Framework {
	serveCmd := func(pm string) string {
		switch pm {
		case "pnpm":
			return "pnpm exec serve dist"
		case "bun":
			return "bunx serve dist"
		default:
			return "npx serve dist"
		}
	}
	if hasFile(dir, "vite.config.js") || hasFile(dir, "vite.config.ts") || hasFile(dir, "vite.config.mjs") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "vite", PMRunBuild(pm), serveCmd(pm), 3000, "dist")
	}
	// Plain vite project without a config file — detect via devDependencies.
	if hasFile(dir, "package.json") {
		data, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err == nil {
			var p struct {
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if json.Unmarshal(data, &p) == nil {
				if _, ok := p.DevDependencies["vite"]; ok {
					pm := detectPackageManager(dir)
					return newNodeFramework(dir, "vite", PMRunBuild(pm), serveCmd(pm), 3000, "dist")
				}
			}
		}
	}
	return nil
}

func detectCRA(dir string) *core.Framework {
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), "react-scripts") {
		pm := detectPackageManager(dir)
		serveCmd := "npx serve -s build"
		switch pm {
		case "pnpm":
			serveCmd = "pnpm exec serve -s build"
		case "bun":
			serveCmd = "bunx serve -s build"
		}
		return newNodeFramework(dir, "create-react-app", PMRunBuild(pm), serveCmd, 3000, "build")
	}
	return nil
}

func detectAngular(dir string) *core.Framework {
	serveCmd := func(pm string) string {
		switch pm {
		case "pnpm":
			return "pnpm exec serve dist/*/browser"
		case "bun":
			return "bunx serve dist/*/browser"
		default:
			return "npx serve dist/*/browser"
		}
	}
	if hasFile(dir, "angular.json") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "angular", PMRunBuild(pm), serveCmd(pm), 4200, "dist")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), "@angular/core") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "angular", PMRunBuild(pm), serveCmd(pm), 4200, "dist")
	}
	return nil
}

func detectVueCLI(dir string) *core.Framework {
	serveCmd := func(pm string) string {
		switch pm {
		case "pnpm":
			return "pnpm exec serve dist"
		case "bun":
			return "bunx serve dist"
		default:
			return "npx serve dist"
		}
	}
	if hasFile(dir, "vue.config.js") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "vue-cli", PMRunBuild(pm), serveCmd(pm), 8080, "dist")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), "@vue/cli-service") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "vue-cli", PMRunBuild(pm), serveCmd(pm), 8080, "dist")
	}
	return nil
}

func detectSolidStart(dir string) *core.Framework {
	if !hasFile(dir, "package.json") {
		return nil
	}
	pkg := filepath.Join(dir, "package.json")
	if fileContains(pkg, "solid-start") || fileContains(pkg, "@solidjs/start") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "solidstart", PMRunBuild(pm), PMRunStart(pm), 3000, "")
	}
	if fileContains(pkg, "solid-js") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "solidjs", PMRunBuild(pm), PMRunStart(pm), 3000, "dist")
	}
	return nil
}

func detectNestJS(dir string) *core.Framework {
	if hasFile(dir, "nest-cli.json") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "nestjs", PMRunBuild(pm), "node dist/main", 3000, "")
	}
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), "@nestjs/core") {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "nestjs", PMRunBuild(pm), "node dist/main", 3000, "")
	}
	return nil
}

func detectExpress(dir string) *core.Framework {
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), `"express"`) {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "express", PMRunInstall(pm), PMRunStart(pm), 3000, "")
	}
	return nil
}

func detectFastify(dir string) *core.Framework {
	if hasFile(dir, "package.json") && fileContains(filepath.Join(dir, "package.json"), `"fastify"`) {
		pm := detectPackageManager(dir)
		return newNodeFramework(dir, "fastify", PMRunInstall(pm), PMRunStart(pm), 3000, "")
	}
	return nil
}
