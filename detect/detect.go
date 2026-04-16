// Package detect identifies application frameworks from project files.
// It scans for package.json, go.mod, requirements.txt, Cargo.toml, and
// similar markers to determine the framework, runtime version, and
// package manager. 45 detectors are registered at init time; custom
// detectors can be added via RegisterDetector.
package detect

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/permanu/docksmith/config"
	"github.com/permanu/docksmith/core"
)

// staticFileExtensions are file extensions that indicate actual web-servable content.
var staticFileExtensions = map[string]bool{
	".html": true, ".htm": true, ".css": true, ".js": true,
	".svg": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true, ".ico": true, ".json": true,
	".xml": true, ".woff": true, ".woff2": true, ".ttf": true,
}

// hasServableContent checks whether a directory contains files that look like
// a static website (HTML, CSS, JS, images). Returns false for empty dirs or
// dirs with only non-web files — these should not silently deploy as static.
func hasServableContent(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(e.Name(), "index.html") {
			return true
		}
	}
	return false
}

// NamedDetector pairs a name with a detection function for registry ordering.
type NamedDetector struct {
	Name   string
	Detect core.DetectorFunc
}

// DetectOptions configures detection behavior.
type DetectOptions struct {
	// ConfigFileNames lists filenames to treat as user config (e.g. "docksmith.toml").
	ConfigFileNames []string
	// AutoFetch is called when all detectors fail to match. The callback
	// should search the community registry, install a matching framework def,
	// and return the re-detected Framework. Return (nil, nil) to fall through
	// to static/error handling. Wired by the root package to avoid import cycles.
	AutoFetch func(dir string) (*core.Framework, error)
	// Hint, when set, receives a message suggesting a registry search when
	// detection fails and AutoFetch is nil.
	Hint func(msg string)
}

// detectors is the ordered registry. Individual runtime detectors
// append to this at init time. Dockerfile and static fallback are handled inline.
var (
	detectorsMu sync.RWMutex
	detectors   []NamedDetector
)

// Detect analyzes dir and returns the detected framework.
// Returns a static-site framework as fallback if nothing matches.
func Detect(dir string) (*core.Framework, error) {
	return DetectWithOptions(dir, DetectOptions{})
}

// DetectWithOptions runs detection with custom options.
// Returns an error when a config file exists but is invalid — this prevents
// silent fallthrough to auto-detection when the user intended a specific config.
func DetectWithOptions(dir string, opts DetectOptions) (*core.Framework, error) {
	// User config — highest priority of all.
	fw, cfgErr := loadConfigFramework(dir, opts)
	if cfgErr != nil {
		return nil, cfgErr
	}
	if fw != nil {
		return fw, nil
	}

	// User Dockerfile — second priority before auto-detection.
	if hasFile(dir, "Dockerfile") {
		return &core.Framework{Name: "dockerfile", Port: 8080}, nil
	}

	detectorsMu.RLock()
	snapshot := make([]NamedDetector, len(detectors))
	copy(snapshot, detectors)
	detectorsMu.RUnlock()

	for _, nd := range snapshot {
		if fw := nd.Detect(dir); fw != nil {
			slog.Debug("framework detected", "detector", nd.Name, "framework", fw.Name)
			return fw, nil
		}
	}

	// Try the community registry before giving up.
	if opts.AutoFetch != nil {
		if fw, err := opts.AutoFetch(dir); err != nil {
			return nil, err
		} else if fw != nil {
			return fw, nil
		}
	}

	if opts.AutoFetch == nil && opts.Hint != nil {
		q := SearchQueryFromDir(dir)
		if q != "" {
			opts.Hint(fmt.Sprintf("Unknown project. Run `docksmith registry search %s` to find community definitions.", q))
		}
	}

	if hasServableContent(dir) {
		return &core.Framework{Name: "static", Port: 80, OutputDir: "."}, nil
	}

	return nil, buildDetectionError(dir)
}

// loadConfigFramework loads the user config and converts it to a Framework.
// Returns (nil, nil) when no config file exists.
// Returns (nil, err) when a config file exists but is invalid.
func loadConfigFramework(dir string, opts DetectOptions) (*core.Framework, error) {
	names := opts.ConfigFileNames
	if len(names) == 0 {
		names = config.DefaultFileNames
	}
	cfg, err := config.LoadWithNames(dir, names)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return ConfigToFramework(cfg), nil
}

// RegisterDetector prepends d to the registry, giving it the highest priority
// among registered detectors.
func RegisterDetector(name string, d core.DetectorFunc) {
	detectorsMu.Lock()
	detectors = append([]NamedDetector{{name, d}}, detectors...)
	detectorsMu.Unlock()
}

// RegisterDetectorBefore inserts d immediately before the named detector.
// If before is not found, d is prepended.
func RegisterDetectorBefore(before, name string, d core.DetectorFunc) {
	detectorsMu.Lock()
	defer detectorsMu.Unlock()
	for i, nd := range detectors {
		if nd.Name == before {
			updated := make([]NamedDetector, 0, len(detectors)+1)
			updated = append(updated, detectors[:i]...)
			updated = append(updated, NamedDetector{name, d})
			updated = append(updated, detectors[i:]...)
			detectors = updated
			return
		}
	}
	detectors = append([]NamedDetector{{name, d}}, detectors...)
}

// ConfigToFramework converts a Config to a Framework for Dockerfile generation.
func ConfigToFramework(c *config.Config) *core.Framework {
	if c.Dockerfile != "" {
		return &core.Framework{
			Name:      "dockerfile",
			OutputDir: c.Dockerfile,
		}
	}

	fw := &core.Framework{
		Name:         runtimeToFrameworkName(c.Runtime),
		BuildCommand: c.Build.Command,
		StartCommand: c.Start.Command,
		Port:         c.RuntimeConfig.Expose,
		SystemDeps:   c.Install.SystemDeps,
	}

	switch c.Runtime {
	case "node":
		fw.NodeVersion = c.Version
		fw.PackageManager = c.PackageManager
	case "python":
		fw.PythonVersion = c.Version
		fw.PythonPM = c.PackageManager
	case "go":
		fw.GoVersion = c.Version
		if fw.BuildCommand == "" {
			fw.BuildCommand = "go build -o app ."
		}
	case "php":
		fw.PHPVersion = c.Version
	case "java":
		fw.JavaVersion = c.Version
	case "dotnet":
		fw.DotnetVersion = c.Version
	case "deno":
		fw.DenoVersion = c.Version
	case "bun":
		fw.BunVersion = c.Version
		fw.PackageManager = "bun"
	}

	return fw
}

// markerFileList is the ordered set of files scanned during detection.
// Used to populate DetectionError.FilesChecked for user diagnostics.
var markerFileList = []string{
	"Dockerfile",
	"docksmith.toml", "docksmith.yaml", "docksmith.yml", "docksmith.json",
	"package.json", "go.mod", "requirements.txt", "pyproject.toml", "Pipfile",
	"Cargo.toml", "Gemfile", "composer.json", "mix.exs",
	"pom.xml", "build.gradle", "build.gradle.kts", "*.csproj",
	"deno.json", "deno.jsonc", "bun.lockb", "bun.lock",
	"main.go", "index.html",
}

// nearMissChecks scans the directory for partial matches — files that indicate
// a runtime but are insufficient for full detection.
func nearMissChecks(dir string) []core.NearMiss {
	var misses []core.NearMiss

	// Go: go.mod without a main package
	if hasFile(dir, "go.mod") {
		if !hasFile(dir, "main.go") && findGoMainPackage(dir) == "" {
			misses = append(misses, core.NearMiss{
				Runtime: "go",
				Found:   "go.mod",
				Missing: "main package (main.go or cmd/*/main.go)",
				Hint:    "is this a library? use --framework go --entrypoint cmd/server",
			})
		}
	}

	// Node: package.json without any framework-specific marker
	if hasFile(dir, "package.json") {
		pkg := filepath.Join(dir, "package.json")
		knownMarkers := []string{
			`"next"`, `"nuxt"`, "@sveltejs/kit", `"astro"`, "@remix-run",
			`"gatsby"`, "react-scripts", "@angular/core", "@vue/cli-service",
			"solid-start", "@solidjs/start", "@nestjs/core", `"express"`, `"fastify"`,
			`"vite"`,
		}
		hasKnown := false
		for _, m := range knownMarkers {
			if fileContains(pkg, m) {
				hasKnown = true
				break
			}
		}
		if !hasKnown {
			misses = append(misses, core.NearMiss{
				Runtime: "node",
				Found:   "package.json",
				Missing: "recognized framework dependency (express, next, fastify, etc.)",
				Hint:    "add a start script in package.json or use --framework node",
			})
		}
	}

	// Python: requirements.txt/pyproject.toml without flask/fastapi/django
	for _, pyFile := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if hasFile(dir, pyFile) {
			pyPath := filepath.Join(dir, pyFile)
			hasFramework := false
			for _, marker := range []string{"flask", "fastapi", "django"} {
				if fileContains(pyPath, marker) {
					hasFramework = true
					break
				}
			}
			if !hasFramework && !hasFile(dir, "manage.py") {
				misses = append(misses, core.NearMiss{
					Runtime: "python",
					Found:   pyFile,
					Missing: "web framework (flask, fastapi, or django)",
					Hint:    "add flask/fastapi to " + pyFile + " or use --framework python",
				})
			}
			break // only report once for python
		}
	}

	// Rust: Cargo.toml without actix-web or axum
	if hasFile(dir, "Cargo.toml") {
		cargo := filepath.Join(dir, "Cargo.toml")
		if !fileContains(cargo, "actix-web") && !fileContains(cargo, "axum") {
			misses = append(misses, core.NearMiss{
				Runtime: "rust",
				Found:   "Cargo.toml",
				Missing: "recognized web framework (actix-web or axum)",
				Hint:    "use --framework rust or add a docksmith.toml with runtime = \"rust\"",
			})
		}
	}

	// Ruby: Gemfile without rails config or sinatra
	if hasFile(dir, "Gemfile") {
		gemfile := filepath.Join(dir, "Gemfile")
		if !hasFile(dir, "config/routes.rb") && !fileContains(gemfile, "sinatra") {
			misses = append(misses, core.NearMiss{
				Runtime: "ruby",
				Found:   "Gemfile",
				Missing: "Rails structure (config/routes.rb) or sinatra dependency",
				Hint:    "use --framework ruby or add a docksmith.toml with runtime = \"ruby\"",
			})
		}
	}

	// Java: pom.xml/build.gradle without spring-boot/quarkus/micronaut
	for _, javaFile := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		if hasFile(dir, javaFile) {
			jPath := filepath.Join(dir, javaFile)
			if !fileContains(jPath, "spring-boot") && !fileContains(jPath, "quarkus") && !fileContains(jPath, "micronaut") {
				// Maven/Gradle generic detectors exist, so this is only a near-miss
				// if the generic detector also failed (no pom.xml at all for gradle case).
				// Actually, maven/gradle generic detectors DO match, so this shouldn't
				// fire. Only add near-miss if the file was somehow not caught.
				// Skip — the generic detectors handle this.
			}
			break
		}
	}

	// Elixir: mix.exs without phoenix
	if hasFile(dir, "mix.exs") {
		if !fileContains(filepath.Join(dir, "mix.exs"), "phoenix") {
			misses = append(misses, core.NearMiss{
				Runtime: "elixir",
				Found:   "mix.exs",
				Missing: "phoenix dependency",
				Hint:    "use --framework elixir or add a docksmith.toml with runtime = \"elixir\"",
			})
		}
	}

	// PHP: composer.json without laravel/symfony/slim/index.php
	if hasFile(dir, "composer.json") && !hasFile(dir, "artisan") && !hasFile(dir, "index.php") &&
		!hasFile(dir, "symfony.lock") && !hasFile(dir, "config/bundles.php") {
		composer := filepath.Join(dir, "composer.json")
		if !fileContains(composer, "slim/slim") && !fileContains(composer, "symfony/framework-bundle") {
			misses = append(misses, core.NearMiss{
				Runtime: "php",
				Found:   "composer.json",
				Missing: "recognized framework (laravel, symfony, slim) or index.php",
				Hint:    "add index.php or use --framework php",
			})
		}
	}

	return misses
}

// buildDetectionError constructs a rich DetectionError with near-miss info.
func buildDetectionError(dir string) *core.DetectionError {
	// Determine which marker files actually exist for the "scanned for" list.
	var checked []string
	for _, f := range markerFileList {
		if strings.Contains(f, "*") {
			matches, _ := filepath.Glob(filepath.Join(dir, f))
			if len(matches) > 0 {
				checked = append(checked, f)
			}
		} else if hasFile(dir, f) {
			checked = append(checked, f+" (found)")
		} else {
			checked = append(checked, f)
		}
	}

	return &core.DetectionError{
		Dir:          dir,
		FilesChecked: checked,
		NearMisses:   nearMissChecks(dir),
	}
}

func runtimeToFrameworkName(runtime string) string {
	switch runtime {
	case "node":
		return "express"
	case "python":
		return "flask"
	case "go":
		return "go-std"
	case "php":
		return "php"
	case "java":
		return "maven"
	case "dotnet":
		return "aspnet-core"
	case "rust":
		return "rust-generic"
	case "ruby":
		return "rails"
	case "elixir":
		return "elixir-phoenix"
	case "deno":
		return "deno"
	case "bun":
		return "bun"
	case "static":
		return "static"
	default:
		return runtime
	}
}
