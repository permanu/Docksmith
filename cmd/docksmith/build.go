package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/permanu/docksmith"
)

func runBuild(cfg config, args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	autoInstall := fs.Bool("auto-install", false, "install from registry if no framework detected")
	_ = fs.Parse(args)

	dir := targetDir(fs.Args())

	dockerfile, fw, err := docksmith.Build(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if fw.Name == "static" {
		after := maybeRegistryInstall(fw, dir, *autoInstall, cfg.quiet)
		if after.Name != "static" {
			dockerfile, fw, err = docksmith.Build(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(2)
			}
		}
	}

	if fw.Name == "dockerfile" {
		if !cfg.quiet {
			fmt.Fprintln(os.Stderr, "using existing Dockerfile")
		}
		runDockerBuild(dir, dir, cfg.quiet)
		return
	}

	if !cfg.quiet {
		fmt.Fprintf(os.Stderr, "detected: %s\n", fw.Name)
	}

	tmp, err := os.MkdirTemp("", "docksmith-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	defer os.RemoveAll(tmp)

	dfPath := filepath.Join(tmp, "Dockerfile")
	if err := os.WriteFile(dfPath, []byte(dockerfile), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write Dockerfile: %v\n", err)
		os.Exit(2)
	}

	diPath := filepath.Join(tmp, ".dockerignore")
	if err := os.WriteFile(diPath, []byte(docksmith.GenerateDockerignore(fw)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write .dockerignore: %v\n", err)
		os.Exit(2)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	runDockerBuild(absDir, tmp, cfg.quiet)
}

func runDockerBuild(contextDir, dockerfileDir string, quiet bool) {
	dfPath := filepath.Join(dockerfileDir, "Dockerfile")
	cmd := exec.Command("docker", "build", "-f", dfPath, contextDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if !quiet {
		fmt.Fprintf(os.Stderr, "running: docker build -f %s %s\n", dfPath, contextDir)
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}
