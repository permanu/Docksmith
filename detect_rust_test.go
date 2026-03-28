package docksmith

import (
	"path/filepath"
	"testing"
)

func TestDetectRustActix(t *testing.T) {
	fw := detectRustActix(filepath.Join("testdata", "fixtures", "rust-actix"))
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "rust-actix" {
		t.Errorf("Name = %q, want rust-actix", fw.Name)
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
	if fw.BuildCommand != "cargo build --release" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
	// Fixture has [package] name = "my-app", so binary must reflect it.
	if fw.StartCommand != "./target/release/my-app" {
		t.Errorf("StartCommand = %q, want ./target/release/my-app", fw.StartCommand)
	}
}

func TestDetectRustActix_NoMatch(t *testing.T) {
	if fw := detectRustActix(filepath.Join("testdata", "fixtures", "rust-no-framework")); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectRustActix_MissingCargo(t *testing.T) {
	if fw := detectRustActix(filepath.Join("testdata", "fixtures", "empty-dir")); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectRustAxum(t *testing.T) {
	fw := detectRustAxum(filepath.Join("testdata", "fixtures", "rust-axum"))
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "rust-axum" {
		t.Errorf("Name = %q, want rust-axum", fw.Name)
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.BuildCommand != "cargo build --release" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
	// Fixture has [package] name = "my-app", so binary must reflect it.
	if fw.StartCommand != "./target/release/my-app" {
		t.Errorf("StartCommand = %q, want ./target/release/my-app", fw.StartCommand)
	}
}

func TestDetectRustAxum_NoMatch(t *testing.T) {
	if fw := detectRustAxum(filepath.Join("testdata", "fixtures", "rust-no-framework")); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectRustAxum_MissingCargo(t *testing.T) {
	if fw := detectRustAxum(filepath.Join("testdata", "fixtures", "empty-dir")); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

// Actix takes priority over Axum when both deps appear in the same Cargo.toml.
func TestDetectRust_ActixPriorityOverAxum(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[dependencies]\nactix-web = \"4\"\naxum = \"0.7\"\n")

	fw, err := Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "rust-actix" {
		t.Errorf("Name = %q, want rust-actix", fw.Name)
	}
}

// TestCargoPackageName_ActixCustomName verifies that a custom package name
// flows into the StartCommand for actix-web projects.
func TestCargoPackageName_ActixCustomName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml",
		"[package]\nname = \"myserver\"\nversion = \"0.1.0\"\n\n[dependencies]\nactix-web = \"4\"\n",
	)
	fw := detectRustActix(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.StartCommand != "./target/release/myserver" {
		t.Errorf("StartCommand = %q, want ./target/release/myserver", fw.StartCommand)
	}
}

// TestCargoPackageName_AxumCustomName verifies that a custom package name
// flows into the StartCommand for axum projects.
func TestCargoPackageName_AxumCustomName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml",
		"[package]\nname = \"myserver\"\nversion = \"0.1.0\"\n\n[dependencies]\naxum = \"0.7\"\ntokio = { version = \"1\", features = [\"full\"] }\n",
	)
	fw := detectRustAxum(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.StartCommand != "./target/release/myserver" {
		t.Errorf("StartCommand = %q, want ./target/release/myserver", fw.StartCommand)
	}
}

// TestCargoPackageName_NoPackageSection verifies fallback to "app" when
// Cargo.toml has no [package] section.
func TestCargoPackageName_NoPackageSection(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml",
		"[dependencies]\nactix-web = \"4\"\n",
	)
	fw := detectRustActix(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.StartCommand != "./target/release/app" {
		t.Errorf("StartCommand = %q, want ./target/release/app", fw.StartCommand)
	}
}

// TestCargoPackageName_FallbackMissingFile verifies fallback to "app" when
// Cargo.toml does not exist at all (edge case via cargoPackageName directly).
func TestCargoPackageName_FallbackMissingFile(t *testing.T) {
	name := cargoPackageName(t.TempDir())
	if name != "app" {
		t.Errorf("cargoPackageName = %q, want app", name)
	}
}
