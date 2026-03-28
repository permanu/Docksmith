package docksmith

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLaravel(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "laravel")
	fw := detectLaravel(dir)
	if fw == nil {
		t.Fatal("got nil, want laravel framework")
	}
	if fw.Name != "laravel" {
		t.Errorf("Name = %q, want %q", fw.Name, "laravel")
	}
	if fw.Port != 8000 {
		t.Errorf("Port = %d, want 8000", fw.Port)
	}
	if fw.PHPVersion != "8.2" {
		t.Errorf("PHPVersion = %q, want %q", fw.PHPVersion, "8.2")
	}
}

func TestDetectLaravel_NoArtisan(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "symfony")
	if fw := detectLaravel(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectWordPress(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "wordpress")
	fw := detectWordPress(dir)
	if fw == nil {
		t.Fatal("got nil, want wordpress framework")
	}
	if fw.Name != "wordpress" {
		t.Errorf("Name = %q, want %q", fw.Name, "wordpress")
	}
	if fw.Port != 80 {
		t.Errorf("Port = %d, want 80", fw.Port)
	}
	// No composer.json in this fixture — defaults to 8.3.
	if fw.PHPVersion != "8.3" {
		t.Errorf("PHPVersion = %q, want %q", fw.PHPVersion, "8.3")
	}
}

func TestDetectWordPress_WpContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "wp-content"), 0o755); err != nil {
		t.Fatal(err)
	}
	fw := detectWordPress(dir)
	if fw == nil {
		t.Fatal("got nil, want wordpress framework")
	}
	if fw.Name != "wordpress" {
		t.Errorf("Name = %q, want %q", fw.Name, "wordpress")
	}
}

func TestDetectWordPress_NoMarkers(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "empty-dir")
	if fw := detectWordPress(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectSymfony(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "symfony")
	fw := detectSymfony(dir)
	if fw == nil {
		t.Fatal("got nil, want symfony framework")
	}
	if fw.Name != "symfony" {
		t.Errorf("Name = %q, want %q", fw.Name, "symfony")
	}
	if fw.Port != 8000 {
		t.Errorf("Port = %d, want 8000", fw.Port)
	}
	if fw.PHPVersion != "8.1" {
		t.Errorf("PHPVersion = %q, want %q", fw.PHPVersion, "8.1")
	}
}

func TestDetectSymfony_LockFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "symfony.lock"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	fw := detectSymfony(dir)
	if fw == nil {
		t.Fatal("got nil, want symfony framework")
	}
}

func TestDetectSlim(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "slim")
	fw := detectSlim(dir)
	if fw == nil {
		t.Fatal("got nil, want slim framework")
	}
	if fw.Name != "slim" {
		t.Errorf("Name = %q, want %q", fw.Name, "slim")
	}
	if fw.PHPVersion != "8.1" {
		t.Errorf("PHPVersion = %q, want %q", fw.PHPVersion, "8.1")
	}
}

func TestDetectSlim_NoSlimDep(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "symfony")
	if fw := detectSlim(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectPlainPHP(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "plain-php")
	fw := detectPlainPHP(dir)
	if fw == nil {
		t.Fatal("got nil, want php framework")
	}
	if fw.Name != "php" {
		t.Errorf("Name = %q, want %q", fw.Name, "php")
	}
	if fw.Port != 80 {
		t.Errorf("Port = %d, want 80", fw.Port)
	}
	if fw.PHPVersion != "8.3" {
		t.Errorf("PHPVersion = %q, want %q", fw.PHPVersion, "8.3")
	}
}

func TestDetectPlainPHP_NoIndexPHP(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "empty-dir")
	if fw := detectPlainPHP(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectPHPVersion_ComposerJSON(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "laravel")
	v := detectPHPVersion(dir)
	if v != "8.2" {
		t.Errorf("got %q, want %q", v, "8.2")
	}
}

func TestDetectPHPVersion_PhpVersionFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".php-version"), []byte("8.1.12\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := detectPHPVersion(dir)
	if v != "8.1" {
		t.Errorf("got %q, want %q", v, "8.1")
	}
}

func TestDetectPHPVersion_Default(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "empty-dir")
	v := detectPHPVersion(dir)
	if v != "8.3" {
		t.Errorf("got %q, want %q", v, "8.3")
	}
}

func TestParsePHPConstraint(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"^8.2", "8.2"},
		{">=8.1", "8.1"},
		{"~8.0", "8.0"},
		{"^8.2.0", "8.2"},
		{">=8.0,<9.0", "8.0"},
		{"8.3.*", "8.3"},
		{"8", "8"},
		{"", ""},
		{"*", ""},
	}
	for _, tc := range cases {
		got := parsePHPConstraint(tc.in)
		if got != tc.want {
			t.Errorf("parsePHPConstraint(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
