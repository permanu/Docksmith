package detect

import (
	"fmt"
	"github.com/permanu/docksmith/core"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func init() {
	// Actix before Axum — both use Cargo.toml, actix-web is more specific.
	RegisterDetector("rust-axum", detectRustAxum)
	RegisterDetector("rust-actix", detectRustActix)
}

// cargoToml is a minimal struct for parsing Cargo.toml.
type cargoToml struct {
	Package struct {
		Name string `toml:"name"`
	} `toml:"package"`
	Bin []struct {
		Name string `toml:"name"`
	} `toml:"bin"`
}

// cargoPackageName returns the binary name for the project.
// When [[bin]] is present, the first entry's name takes precedence over
// [package] name (it's an explicit binary target). Falls back to "app".
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
	if len(cargo.Bin) > 0 && cargo.Bin[0].Name != "" {
		return cargo.Bin[0].Name
	}
	if cargo.Package.Name == "" {
		return "app"
	}
	return cargo.Package.Name
}

func detectRustActix(dir string) *core.Framework {
	if !hasFile(dir, "Cargo.toml") || !fileContains(filepath.Join(dir, "Cargo.toml"), "actix-web") {
		return nil
	}
	binName := cargoPackageName(dir)
	return &core.Framework{
		Name:         "rust-actix",
		BuildCommand: "cargo build --release",
		StartCommand: fmt.Sprintf("./target/release/%s", binName),
		Port:         8080,
	}
}

func detectRustAxum(dir string) *core.Framework {
	if !hasFile(dir, "Cargo.toml") || !fileContains(filepath.Join(dir, "Cargo.toml"), "axum") {
		return nil
	}
	binName := cargoPackageName(dir)
	return &core.Framework{
		Name:         "rust-axum",
		BuildCommand: "cargo build --release",
		StartCommand: fmt.Sprintf("./target/release/%s", binName),
		Port:         3000,
	}
}
