// SPDX-License-Identifier: MPL-2.0

package detect

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMonorepoFixtureResolvesNamedServiceNotRoot(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "monorepo")

	got := goBuildPath(dir)
	if got != "./cmd/web" {
		t.Fatalf("goBuildPath = %q, want ./cmd/web named by the module, not the guessed root", got)
	}

	fw := detectGoStd(dir)
	if fw == nil {
		t.Fatal("detectGoStd = nil, want the named web service")
	}
	if fw.BuildCommand != "go build -o app ./cmd/web" {
		t.Fatalf("BuildCommand = %q, want the service the fixture names", fw.BuildCommand)
	}
	if strings.HasSuffix(fw.BuildCommand, " .") {
		t.Fatalf("BuildCommand = %q uses the guessed root", fw.BuildCommand)
	}
}
