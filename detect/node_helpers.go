package detect

import (
	"encoding/json"
	"github.com/permanu/docksmith/core"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// packageJSON holds the fields we care about for Node version and PM detection.
type packageJSON struct {
	Engines struct {
		Node string `json:"node"`
	} `json:"engines"`
	Volta struct {
		Node string `json:"node"`
	} `json:"volta"`
	PackageManager string `json:"packageManager"`
}

func detectNodeVersion(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, ".nvmrc")); err == nil {
		if v := parseVersionString(string(data)); v != "" {
			return v
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".node-version")); err == nil {
		if v := parseVersionString(string(data)); v != "" {
			return v
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg packageJSON
		if json.Unmarshal(data, &pkg) == nil {
			if v := extractMajorVersion(pkg.Engines.Node); v != "" {
				return v
			}
			if v := extractMajorVersion(pkg.Volta.Node); v != "" {
				return v
			}
		}
	}
	return "22"
}

// detectPackageManager reads the package manager from project files.
// Priority: packageManager field > lockfiles (bun > pnpm > yarn > npm) > default "npm".
//
// The lockfile priority intentionally favours bun over pnpm: Permanu's own
// frontends use bun (feedback_bun_not_npm.md), and a project that ships a
// bun.lockb is declaring bun as the source of truth even if a stale
// pnpm-lock.yaml lingers in the tree. Use LockfileConflicts to surface
// ambiguity to callers that want to warn the operator.
func detectPackageManager(dir string) string {
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg packageJSON
		if json.Unmarshal(data, &pkg) == nil && pkg.PackageManager != "" {
			name := strings.Split(pkg.PackageManager, "@")[0]
			switch name {
			case "pnpm", "yarn", "bun", "npm":
				return name
			}
		}
	}
	if hasFile(dir, "bun.lockb") || hasFile(dir, "bun.lock") {
		return "bun"
	}
	if hasFile(dir, "pnpm-lock.yaml") {
		return "pnpm"
	}
	if hasFile(dir, "yarn.lock") {
		return "yarn"
	}
	if hasFile(dir, "package-lock.json") {
		return "npm"
	}
	return "npm"
}

// LockfileConflicts returns the names of every Node lockfile present in dir.
// Callers should warn the operator whenever the slice has length > 1: a project
// with multiple lockfiles is ambiguous, and detectPackageManager picks one
// deterministically (see priority order) but the operator should commit to a
// single package manager.
func LockfileConflicts(dir string) []string {
	var found []string
	for _, name := range []string{"bun.lockb", "bun.lock", "pnpm-lock.yaml", "yarn.lock", "package-lock.json"} {
		if hasFile(dir, name) {
			found = append(found, name)
		}
	}
	if len(found) <= 1 {
		return nil
	}
	return found
}

// PMRunBuild returns the package-manager-specific build command.
func PMRunBuild(pm string) string {
	switch pm {
	case "pnpm":
		return "pnpm run build"
	case "yarn":
		return "yarn run build"
	case "bun":
		return "bun run build"
	default:
		return "npm run build"
	}
}

// PMInstallCommand returns a deterministic-first install command that falls back
// when lockfiles are stale — production repos often have slightly out-of-date
// lockfiles that --frozen-lockfile rejects.
func PMInstallCommand(pm string) string {
	switch pm {
	case "pnpm":
		return "pnpm install --frozen-lockfile || pnpm install"
	case "yarn":
		return "yarn install --frozen-lockfile || yarn install"
	case "bun":
		return "bun install --frozen-lockfile || bun install"
	default:
		return "if [ -f package-lock.json ]; then npm ci || npm install; else npm install; fi"
	}
}

// PMRunStart returns the package-manager-specific start command.
func PMRunStart(pm string) string {
	switch pm {
	case "pnpm":
		return "pnpm start"
	case "yarn":
		return "yarn start"
	case "bun":
		return "bun start"
	default:
		return "npm start"
	}
}

// PMRunInstall returns a plain (non-frozen) install for backend frameworks like
// Express where there's no build step and lockfile discipline is less strict.
func PMRunInstall(pm string) string {
	switch pm {
	case "pnpm":
		return "pnpm install"
	case "yarn":
		return "yarn install"
	case "bun":
		return "bun install"
	default:
		return "npm install"
	}
}

// NodeVersionAtLeast reports whether the node version string is >= min.
// An empty version returns true because the default is latest, which is >= 22.
// If parsing fails the function returns true (assume latest).
func NodeVersionAtLeast(ver string, min int) bool {
	if ver == "" {
		return true
	}
	// ver may be "22", "22.1", "20.11.0", etc. — take the first segment.
	major := strings.SplitN(ver, ".", 2)[0]
	major = strings.TrimSpace(major)
	n, err := strconv.Atoi(major)
	if err != nil {
		return true // unparseable — assume latest
	}
	return n >= min
}

func newNodeFramework(dir, name, buildCmd, startCmd string, port int, outputDir string) *core.Framework {
	pm := detectPackageManager(dir)
	return &core.Framework{
		Name:           name,
		BuildCommand:   buildCmd,
		StartCommand:   startCmd,
		Port:           port,
		OutputDir:      outputDir,
		NodeVersion:    detectNodeVersion(dir),
		PackageManager: pm,
	}
}
