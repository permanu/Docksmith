package docksmith

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectNodeVersion_NvmRC(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, ".nvmrc", "18\n")
	if got := detectNodeVersion(dir); got != "18" {
		t.Errorf("got %q, want %q", got, "18")
	}
}

func TestDetectNodeVersion_NodeVersion(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, ".node-version", "v20.11.0\n")
	if got := detectNodeVersion(dir); got != "20.11.0" {
		t.Errorf("got %q, want %q", got, "20.11.0")
	}
}

func TestDetectNodeVersion_PackageJSONEngines(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "package.json", `{"engines":{"node":">=18.0.0"}}`)
	if got := detectNodeVersion(dir); got != "18" {
		t.Errorf("got %q, want %q", got, "18")
	}
}

func TestDetectNodeVersion_Volta(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "package.json", `{"volta":{"node":"20.11.0"}}`)
	// extractMajorVersion preserves meaningful minor: "20.11.0" -> "20.11"
	if got := detectNodeVersion(dir); got != "20.11" {
		t.Errorf("got %q, want %q", got, "20.11")
	}
}

func TestDetectNodeVersion_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectNodeVersion(dir); got != "22" {
		t.Errorf("got %q, want %q", got, "22")
	}
}

// NvmRC takes priority over package.json.
func TestDetectNodeVersion_NvmrcWins(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, ".nvmrc", "16\n")
	nodeWrite(t, dir, "package.json", `{"engines":{"node":">=18"}}`)
	if got := detectNodeVersion(dir); got != "16" {
		t.Errorf("got %q, want %q", got, "16")
	}
}

func TestDetectPackageManager_PackageManagerField(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "package.json", `{"packageManager":"pnpm@9.1.0"}`)
	if got := detectPackageManager(dir); got != "pnpm" {
		t.Errorf("got %q, want %q", got, "pnpm")
	}
}

func TestDetectPackageManager_BunLockb(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "bun.lockb", "")
	if got := detectPackageManager(dir); got != "bun" {
		t.Errorf("got %q, want %q", got, "bun")
	}
}

func TestDetectPackageManager_PnpmLock(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "pnpm-lock.yaml", "lockfileVersion: '6.0'\n")
	if got := detectPackageManager(dir); got != "pnpm" {
		t.Errorf("got %q, want %q", got, "pnpm")
	}
}

func TestDetectPackageManager_YarnLock(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "yarn.lock", "# yarn lockfile v1\n")
	if got := detectPackageManager(dir); got != "yarn" {
		t.Errorf("got %q, want %q", got, "yarn")
	}
}

func TestDetectPackageManager_PackageLock(t *testing.T) {
	dir := t.TempDir()
	nodeWrite(t, dir, "package-lock.json", `{"lockfileVersion":3}`)
	if got := detectPackageManager(dir); got != "npm" {
		t.Errorf("got %q, want %q", got, "npm")
	}
}

func TestDetectPackageManager_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectPackageManager(dir); got != "npm" {
		t.Errorf("got %q, want %q", got, "npm")
	}
}

func nodeWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
