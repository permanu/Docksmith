package detect

import (
	"path/filepath"
	"testing"
)

func TestDetectRails(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "rails")
	fw := detectRails(dir)
	if fw == nil {
		t.Fatal("got nil, want rails framework")
	}
	if fw.Name != "rails" {
		t.Errorf("Name = %q, want %q", fw.Name, "rails")
	}
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.BuildCommand != "bundle install" {
		t.Errorf("BuildCommand = %q, want %q", fw.BuildCommand, "bundle install")
	}
}

func TestDetectRails_NoRoutesRb(t *testing.T) {
	// Gemfile present but no config/routes.rb — not Rails.
	dir := filepath.Join("testdata", "fixtures", "sinatra")
	if fw := detectRails(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectRails_NoGemfile(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "empty-dir")
	if fw := detectRails(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectSinatra(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "sinatra")
	fw := detectSinatra(dir)
	if fw == nil {
		t.Fatal("got nil, want sinatra framework")
	}
	if fw.Name != "sinatra" {
		t.Errorf("Name = %q, want %q", fw.Name, "sinatra")
	}
	if fw.Port != 4567 {
		t.Errorf("Port = %d, want 4567", fw.Port)
	}
}

func TestDetectSinatra_NoSinatraGem(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "gemfile-no-rails")
	if fw := detectSinatra(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectSinatra_RailsGemfileNotMatched(t *testing.T) {
	// Rails Gemfile doesn't contain "sinatra" — must not false-positive.
	dir := filepath.Join("testdata", "fixtures", "rails")
	if fw := detectSinatra(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}
