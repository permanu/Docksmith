package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/permanu/docksmith/registry"
)

func registryURL(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	return cmp.Or(os.Getenv("DOCKSMITH_REGISTRY"), registry.DefaultRegistryURL)
}

func runRegistry(cfg config, args []string) {
	fs := flag.NewFlagSet("registry", flag.ExitOnError)
	offline := fs.Bool("offline", false, "use cached index only, no network call")
	regURL := fs.String("registry", "", "override registry index URL")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: docksmith registry [flags] <subcommand>

Subcommands:
  search <query>   search available community frameworks
  install <name>   install a framework YAML to ~/.docksmith/frameworks/

Flags:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	sub := fs.Args()
	if len(sub) == 0 {
		fs.Usage()
		os.Exit(1)
	}

	url := registryURL(*regURL)

	switch sub[0] {
	case "search":
		if err := execSearch(cfg, url, *offline, sub[1:], os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
	case "install":
		if err := execInstall(url, *offline, sub[1:], os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown registry subcommand %q\n\n", sub[0])
		fs.Usage()
		os.Exit(1)
	}
}

func execSearch(cfg config, url string, offline bool, args []string, out, errw io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := ""
	if fs.NArg() > 0 {
		query = fs.Arg(0)
	}

	idx, err := registry.FetchIndex(url, offline)
	if err != nil {
		return err
	}

	results := registry.Search(idx, query)
	if len(results) == 0 {
		if query != "" {
			fmt.Fprintf(errw, "no frameworks found matching %q\n", query)
		} else {
			fmt.Fprintln(errw, "no frameworks found")
		}
		return nil
	}

	if cfg.isJSON() {
		data, err := json.MarshalIndent(searchResultsJSON(results), "", "  ")
		if err != nil {
			return fmt.Errorf("marshal results: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	fmt.Fprintf(out, "%-20s %-10s %s\n", "NAME", "RUNTIME", "DESCRIPTION")
	for _, e := range results {
		fmt.Fprintf(out, "%-20s %-10s %s\n", e.Name, e.Runtime, e.Description)
	}
	return nil
}

type errNotFound struct {
	name        string
	suggestions []string
}

func (e *errNotFound) Error() string {
	msg := fmt.Sprintf("framework %q not found in registry", e.name)
	if len(e.suggestions) > 0 {
		msg += "\ndid you mean: " + strings.Join(e.suggestions, ", ")
	}
	return msg
}

func execInstall(url string, offline bool, args []string, out, errw io.Writer) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite if already installed")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("install requires a framework name")
	}
	name := fs.Arg(0)

	idx, err := registry.FetchIndex(url, offline)
	if err != nil {
		return err
	}

	entry, ok := findEntry(idx, name)
	if !ok {
		return &errNotFound{name: name, suggestions: suggestNames(idx, name)}
	}

	if !*force {
		dest := filepath.Join(frameworksDir(), entry.Name+".yaml")
		if _, err := os.Stat(dest); err == nil {
			fmt.Fprintf(errw, "%s already installed at %s (use --force to overwrite)\n", entry.Name, dest)
			return nil
		}
	}

	fmt.Fprintf(errw, "Downloading %s@%s...\n", entry.Name, entry.Version)

	dest, err := registry.InstallFramework(entry)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Installed to %s\n", dest)
	return nil
}

type searchResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Runtime     string `json:"runtime"`
	Description string `json:"description"`
	Author      string `json:"author"`
}

func searchResultsJSON(entries []registry.Entry) []searchResult {
	out := make([]searchResult, len(entries))
	for i, e := range entries {
		out[i] = searchResult{
			Name:        e.Name,
			Version:     e.Version,
			Runtime:     e.Runtime,
			Description: e.Description,
			Author:      e.Author,
		}
	}
	return out
}

// findEntry looks up a framework by name (case-insensitive).
func findEntry(idx *registry.Index, name string) (registry.Entry, bool) {
	lower := strings.ToLower(name)
	for k, e := range idx.Frameworks {
		if strings.ToLower(k) == lower {
			e.Name = k
			return e, true
		}
	}
	return registry.Entry{}, false
}

// suggestNames returns up to 3 framework names containing query as substring.
func suggestNames(idx *registry.Index, query string) []string {
	q := strings.ToLower(query)
	var matches []string
	for name := range idx.Frameworks {
		if strings.Contains(strings.ToLower(name), q) ||
			strings.Contains(q, strings.ToLower(name)) {
			matches = append(matches, name)
		}
	}
	if len(matches) > 3 {
		matches = matches[:3]
	}
	return matches
}

func frameworksDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docksmith", "frameworks")
}
