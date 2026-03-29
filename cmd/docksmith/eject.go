package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/permanu/docksmith"
)

func runEject(_ config, args []string) {
	fs := flag.NewFlagSet("eject", flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite existing Dockerfile and .dockerignore")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: docksmith eject [--force] [path]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	dir := targetDir(fs.Args())

	fw, detectErr := docksmith.Detect(dir)
	if detectErr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", detectErr)
		os.Exit(1)
	}
	if fw.Name == "dockerfile" {
		fmt.Fprintln(os.Stderr, "error: project already has a Dockerfile — nothing to eject")
		os.Exit(1)
	}

	dockerfile, genErr := docksmith.GenerateDockerfile(fw)
	if genErr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", genErr)
		os.Exit(2)
	}
	if dockerfile == "" {
		fmt.Fprintf(os.Stderr, "error: could not generate Dockerfile for framework %q\n", fw.Name)
		os.Exit(2)
	}

	dockerfilePath, err := writeProjectFile(dir, "Dockerfile", []byte(dockerfile), 0o644, *force)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			path, pathErr := projectFilePath(dir, "Dockerfile")
			if pathErr != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", pathErr)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "error: Dockerfile already exists at %s (use --force to overwrite)\n", path)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: write Dockerfile: %v\n", err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", dockerfilePath)

	ignore := docksmith.GenerateDockerignore(fw)
	ignorePath, err := writeProjectFile(dir, ".dockerignore", []byte(ignore), 0o644, *force)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			path, pathErr := projectFilePath(dir, ".dockerignore")
			if pathErr != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", pathErr)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "error: .dockerignore already exists at %s (use --force to overwrite)\n", path)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: write .dockerignore: %v\n", err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", ignorePath)
}
