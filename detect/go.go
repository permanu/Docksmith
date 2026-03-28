package detect

import (
	"github.com/permanu/docksmith/core"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	// Specific Go frameworks before stdlib — stdlib matches any go.mod with main.
	RegisterDetector("go-gin", detectGoGin)
	RegisterDetector("go-echo", detectGoEcho)
	RegisterDetector("go-fiber", detectGoFiber)
	RegisterDetector("go", detectGoStd)
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

// findGoMainPackage searches cmd/, bin/, and internal/cmd/ for a main package.
// Returns a relative import path like "./cmd/server", or empty string.
func findGoMainPackage(dir string) string {
	for _, candidate := range []string{"cmd", "bin", "internal/cmd"} {
		candidatePath := filepath.Join(dir, candidate)
		if info, err := os.Stat(candidatePath); err != nil || !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(candidatePath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subdir := filepath.Join(candidatePath, entry.Name())
			goFiles, _ := filepath.Glob(filepath.Join(subdir, "*.go"))
			for _, gf := range goFiles {
				if fileContains(gf, "package main") {
					return "./" + filepath.Join(candidate, entry.Name())
				}
			}
		}
		// candidate dir itself (e.g. ./cmd/main.go without a subdir)
		goFiles, _ := filepath.Glob(filepath.Join(candidatePath, "*.go"))
		for _, gf := range goFiles {
			if fileContains(gf, "package main") {
				return "./" + candidate
			}
		}
	}
	return ""
}

func detectGoGin(dir string) *core.Framework {
	if hasFile(dir, "go.mod") && fileContains(filepath.Join(dir, "go.mod"), "gin-gonic/gin") {
		return &core.Framework{
			Name:         "go-gin",
			BuildCommand: "go build -o app .",
			StartCommand: "./app",
			Port:         8080,
			GoVersion:    detectGoVersion(dir),
		}
	}
	return nil
}

func detectGoEcho(dir string) *core.Framework {
	if hasFile(dir, "go.mod") && fileContains(filepath.Join(dir, "go.mod"), "labstack/echo") {
		return &core.Framework{
			Name:         "go-echo",
			BuildCommand: "go build -o app .",
			StartCommand: "./app",
			Port:         8080,
			GoVersion:    detectGoVersion(dir),
		}
	}
	return nil
}

func detectGoFiber(dir string) *core.Framework {
	if hasFile(dir, "go.mod") && fileContains(filepath.Join(dir, "go.mod"), "gofiber/fiber") {
		return &core.Framework{
			Name:         "go-fiber",
			BuildCommand: "go build -o app .",
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
