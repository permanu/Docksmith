package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/permanu/docksmith"
	"github.com/permanu/docksmith/registry"
)

// maybeRegistryInstall checks the registry for a matching framework when
// detection falls back to "static". With autoInstall it downloads and
// re-detects; otherwise it prints a hint and returns the original fw.
func maybeRegistryInstall(fw *docksmith.Framework, dir string, autoInstall, quiet bool) *docksmith.Framework {
	idx, err := registry.FetchIndex(registry.DefaultRegistryURL, false)
	if err != nil {
		return fw
	}

	hint := filepath.Base(dir)
	results := registry.Search(idx, hint)
	if len(results) == 0 {
		return fw
	}

	match := results[0]

	if !autoInstall {
		if !quiet {
			fmt.Fprintf(os.Stderr,
				"No framework detected. Registry has %q — run `docksmith registry install %s`\n",
				match.Name, match.Name,
			)
		}
		return fw
	}

	fmt.Fprintf(os.Stderr, "Downloading %s@%s...\n", match.Name, match.Version)
	dest, err := registry.InstallFramework(match)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: auto-install failed: %v\n", err)
		return fw
	}
	fmt.Fprintf(os.Stderr, "Installed to %s\n", dest)

	if err := docksmith.LoadAndRegisterFrameworks(userFrameworksInstallDir()); err != nil && !quiet {
		fmt.Fprintf(os.Stderr, "warn: %v\n", err)
	}
	redetected, redetectErr := docksmith.Detect(dir)
	if redetectErr != nil {
		fmt.Fprintf(os.Stderr, "warn: re-detect failed: %v\n", redetectErr)
		return fw
	}
	return redetected
}

func userFrameworksInstallDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docksmith", "frameworks")
}
