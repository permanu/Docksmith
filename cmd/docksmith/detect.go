package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/permanu/docksmith"
)

func runDetect(cfg config, args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	autoInstall := fs.Bool("auto-install", false, "install from registry if no framework detected")
	_ = fs.Parse(args)

	dir := targetDir(fs.Args())
	fw, err := docksmith.Detect(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fw.Name == "static" {
		fw = maybeRegistryInstall(fw, dir, *autoInstall, cfg.quiet)
	}

	if cfg.isJSON() {
		data, err := json.MarshalIndent(fw, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		fmt.Println(string(data))
		return
	}

	printFramework(fw)
}

func printFramework(fw *docksmith.Framework) {
	fmt.Printf("framework:  %s\n", fw.Name)
	fmt.Printf("port:       %d\n", fw.Port)

	if fw.NodeVersion != "" {
		fmt.Printf("node:       %s\n", fw.NodeVersion)
	}
	if fw.PackageManager != "" {
		fmt.Printf("pm:         %s\n", fw.PackageManager)
	}
	if fw.PythonVersion != "" {
		fmt.Printf("python:     %s\n", fw.PythonVersion)
	}
	if fw.PythonPM != "" {
		fmt.Printf("python-pm:  %s\n", fw.PythonPM)
	}
	if fw.GoVersion != "" {
		fmt.Printf("go:         %s\n", fw.GoVersion)
	}
	if fw.BunVersion != "" {
		fmt.Printf("bun:        %s\n", fw.BunVersion)
	}
	if fw.DenoVersion != "" {
		fmt.Printf("deno:       %s\n", fw.DenoVersion)
	}
	if fw.PHPVersion != "" {
		fmt.Printf("php:        %s\n", fw.PHPVersion)
	}
	if fw.JavaVersion != "" {
		fmt.Printf("java:       %s\n", fw.JavaVersion)
	}
	if fw.DotnetVersion != "" {
		fmt.Printf("dotnet:     %s\n", fw.DotnetVersion)
	}
	if fw.BuildCommand != "" {
		fmt.Printf("build:      %s\n", fw.BuildCommand)
	}
	if fw.StartCommand != "" {
		fmt.Printf("start:      %s\n", fw.StartCommand)
	}
	if fw.OutputDir != "" {
		fmt.Printf("output-dir: %s\n", fw.OutputDir)
	}
}
