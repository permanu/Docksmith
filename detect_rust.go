package docksmith

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func init() {
	// Actix before Axum — both use Cargo.toml, actix-web is more specific.
	RegisterDetector("rust-axum", detectRustAxum)
	RegisterDetector("rust-actix", detectRustActix)
}

// cargoToml is a minimal struct for parsing the [package] section of Cargo.toml.
type cargoToml struct {
	Package struct {
		Name string `toml:"name"`
	} `toml:"package"`
}

// cargoPackageName reads [package] name from Cargo.toml in dir.
// Returns "app" if the file is missing, unreadable, or has no name.
func cargoPackageName(dir string) string {
	path := filepath.Join(dir, "Cargo.toml")
	data, err := readFileLimited(path)
	if err != nil {
		return "app"
	}
	var cargo cargoToml
	if err := toml.Unmarshal(data, &cargo); err != nil {
		return "app"
	}
	if cargo.Package.Name == "" {
		return "app"
	}
	return cargo.Package.Name
}

func detectRustActix(dir string) *Framework {
	if !hasFile(dir, "Cargo.toml") || !fileContains(filepath.Join(dir, "Cargo.toml"), "actix-web") {
		return nil
	}
	binName := cargoPackageName(dir)
	return &Framework{
		Name:         "rust-actix",
		BuildCommand: "cargo build --release",
		StartCommand: fmt.Sprintf("./target/release/%s", binName),
		Port:         8080,
	}
}

func detectRustAxum(dir string) *Framework {
	if !hasFile(dir, "Cargo.toml") || !fileContains(filepath.Join(dir, "Cargo.toml"), "axum") {
		return nil
	}
	binName := cargoPackageName(dir)
	return &Framework{
		Name:         "rust-axum",
		BuildCommand: "cargo build --release",
		StartCommand: fmt.Sprintf("./target/release/%s", binName),
		Port:         3000,
	}
}
