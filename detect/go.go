package detect

import (
	"github.com/permanu/docksmith/core"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// preferredCmdNames are common entrypoint names used to disambiguate when a
// repo has multiple cmd/*/main.go binaries and no module-name match.
var preferredCmdNames = []string{"server", "app", "api", "service", "main", "web"}

func init() {
	// RegisterDetector prepends, so the LAST registered detector is checked
	// FIRST. We want gin/echo/fiber matched before stdlib (since stdlib also
	// matches any go.mod with a main package). Register stdlib first.
	RegisterDetector("go", detectGoStd)
	RegisterDetector("go-fiber", detectGoFiber)
	RegisterDetector("go-echo", detectGoEcho)
	RegisterDetector("go-gin", detectGoGin)
}

func detectGoVersion(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		re := regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".go-version")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			parts := strings.SplitN(v, ".", 3)
			if len(parts) >= 2 {
				return parts[0] + "." + parts[1]
			}
			return v
		}
	}
	return "1.25"
}

// findGoMainPackages scans cmd/, bin/, and internal/cmd/ for main packages.
// Returns relative import paths like "./cmd/server" sorted alphabetically.
// Multiple results indicate a monorepo with multiple binaries.
func findGoMainPackages(dir string) []string {
	var found []string
	for _, candidate := range []string{"cmd", "bin", "internal/cmd"} {
		candidatePath := filepath.Join(dir, candidate)
		if info, err := os.Stat(candidatePath); err != nil || !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(candidatePath)
		if err != nil {
			continue
		}
		subdirHit := false
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subdir := filepath.Join(candidatePath, entry.Name())
			goFiles, _ := filepath.Glob(filepath.Join(subdir, "*.go"))
			for _, gf := range goFiles {
				if fileContains(gf, "package main") {
					found = append(found, "./"+filepath.ToSlash(filepath.Join(candidate, entry.Name())))
					subdirHit = true
					break
				}
			}
		}
		if subdirHit {
			continue
		}
		// candidate dir itself (e.g. ./cmd/main.go without a subdir)
		goFiles, _ := filepath.Glob(filepath.Join(candidatePath, "*.go"))
		for _, gf := range goFiles {
			if fileContains(gf, "package main") {
				found = append(found, "./"+candidate)
				break
			}
		}
	}
	sort.Strings(found)
	return found
}

// findGoMainPackage returns a single main package path from cmd/, bin/, or
// internal/cmd/. When multiple candidates exist, it disambiguates using:
//  1. exact match against the go.mod module's last path segment, then
//  2. common entrypoint names (server, app, api, service, main, web).
//
// Returns "" when no main is found OR when multiple candidates are
// genuinely ambiguous — callers must surface an actionable error in that
// case (see nearMissChecks "ambiguous Go monorepo" entry). Users override
// via docksmith.toml: build.command = "go build -o app ./cmd/<name>".
func findGoMainPackage(dir string) string {
	pkgs := findGoMainPackages(dir)
	switch len(pkgs) {
	case 0:
		return ""
	case 1:
		return pkgs[0]
	}
	// Multiple — try module-name match first.
	if mod := goModuleBaseName(dir); mod != "" {
		for _, p := range pkgs {
			if filepath.Base(p) == mod {
				return p
			}
		}
	}
	// Then preferred entrypoint names, in priority order.
	for _, want := range preferredCmdNames {
		for _, p := range pkgs {
			if filepath.Base(p) == want {
				return p
			}
		}
	}
	return ""
}

// goModuleBaseName returns the last path segment of the module declaration
// in go.mod (e.g. "module github.com/foo/bar" → "bar"). Empty on miss.
func goModuleBaseName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^module\s+(\S+)`)
	m := re.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return ""
	}
	return path.Base(m[1])
}

// goBuildPath resolves the build target for a Go project: "." when a root
// main.go exists, the resolved cmd/<name> path otherwise. Returns "" when no
// main package can be determined unambiguously.
func goBuildPath(dir string) string {
	if hasFile(dir, "main.go") {
		return "."
	}
	return findGoMainPackage(dir)
}

func detectGoGin(dir string) *core.Framework {
	if hasFile(dir, "go.mod") && fileContains(filepath.Join(dir, "go.mod"), "gin-gonic/gin") {
		bp := goBuildPath(dir)
		if bp == "" {
			return nil
		}
		return &core.Framework{
			Name:         "go-gin",
			BuildCommand: "go build -o app " + bp,
			StartCommand: "./app",
			Port:         8080,
			GoVersion:    detectGoVersion(dir),
		}
	}
	return nil
}

func detectGoEcho(dir string) *core.Framework {
	if hasFile(dir, "go.mod") && fileContains(filepath.Join(dir, "go.mod"), "labstack/echo") {
		bp := goBuildPath(dir)
		if bp == "" {
			return nil
		}
		return &core.Framework{
			Name:         "go-echo",
			BuildCommand: "go build -o app " + bp,
			StartCommand: "./app",
			Port:         8080,
			GoVersion:    detectGoVersion(dir),
		}
	}
	return nil
}

func detectGoFiber(dir string) *core.Framework {
	if hasFile(dir, "go.mod") && fileContains(filepath.Join(dir, "go.mod"), "gofiber/fiber") {
		bp := goBuildPath(dir)
		if bp == "" {
			return nil
		}
		return &core.Framework{
			Name:         "go-fiber",
			BuildCommand: "go build -o app " + bp,
			StartCommand: "./app",
			Port:         3000,
			GoVersion:    detectGoVersion(dir),
		}
	}
	return nil
}

func detectGoStd(dir string) *core.Framework {
	if !hasFile(dir, "go.mod") {
		return nil
	}
	if hasFile(dir, "main.go") {
		return &core.Framework{
			Name:         "go",
			BuildCommand: "go build -o app .",
			StartCommand: "./app",
			Port:         8080,
			GoVersion:    detectGoVersion(dir),
		}
	}
	if mainPkg := findGoMainPackage(dir); mainPkg != "" {
		return &core.Framework{
			Name:         "go",
			BuildCommand: "go build -o app " + mainPkg,
			StartCommand: "./app",
			Port:         8080,
			GoVersion:    detectGoVersion(dir),
		}
	}
	return nil
}
