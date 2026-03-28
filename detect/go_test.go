package detect

import (
	"path/filepath"
	"testing"
)

func TestDetectGoVersion_FromGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
	if got := detectGoVersion(dir); got != "1.26" {
		t.Errorf("detectGoVersion = %q, want %q", got, "1.26")
	}
}

func TestDetectGoVersion_FromGoVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".go-version", "1.23.4\n")
	if got := detectGoVersion(dir); got != "1.23" {
		t.Errorf("detectGoVersion = %q, want %q", got, "1.23")
	}
}

func TestDetectGoVersion_GoModTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, dir, ".go-version", "1.19.0\n")
	if got := detectGoVersion(dir); got != "1.21" {
		t.Errorf("detectGoVersion = %q, want %q", got, "1.21")
	}
}

func TestDetectGoVersion_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectGoVersion(dir); got != "1.25" {
		t.Errorf("detectGoVersion = %q, want %q", got, "1.25")
	}
}

func TestFindGoMainPackage_RootMainGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	// detectGoStd checks root main.go before calling findGoMainPackage,
	// so findGoMainPackage should return "" for pure root case.
	if got := findGoMainPackage(dir); got != "" {
		t.Errorf("findGoMainPackage = %q, want empty (root main.go is handled upstream)", got)
	}
}

func TestFindGoMainPackage_CmdSubdir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "cmd/server/main.go", "package main\n\nfunc main() {}\n")
	if got := findGoMainPackage(dir); got != "./cmd/server" {
		t.Errorf("findGoMainPackage = %q, want %q", got, "./cmd/server")
	}
}

func TestFindGoMainPackage_NoMainPackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	if got := findGoMainPackage(dir); got != "" {
		t.Errorf("findGoMainPackage = %q, want empty string", got)
	}
}

func TestDetectGoGin(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-gin")
	fw := detectGoGin(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "go-gin" {
		t.Errorf("Name = %q, want %q", fw.Name, "go-gin")
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
	if fw.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
	if fw.BuildCommand != "go build -o app ." {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
	if fw.StartCommand != "./app" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectGoGin_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	if fw := detectGoGin(dir); fw != nil {
		t.Errorf("got %q, want nil without go.mod", fw.Name)
	}
}

func TestDetectGoGin_NoGinDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	if fw := detectGoGin(dir); fw != nil {
		t.Errorf("got %q, want nil without gin dep", fw.Name)
	}
}

func TestDetectGoEcho(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-echo")
	fw := detectGoEcho(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "go-echo" {
		t.Errorf("Name = %q, want %q", fw.Name, "go-echo")
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
	if fw.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
}

func TestDetectGoEcho_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	if fw := detectGoEcho(dir); fw != nil {
		t.Errorf("got %q, want nil without go.mod", fw.Name)
	}
}

func TestDetectGoEcho_NoEchoDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	if fw := detectGoEcho(dir); fw != nil {
		t.Errorf("got %q, want nil without echo dep", fw.Name)
	}
}

func TestDetectGoFiber(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-fiber")
	fw := detectGoFiber(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "go-fiber" {
		t.Errorf("Name = %q, want %q", fw.Name, "go-fiber")
	}
	// Fiber defaults to 3000.
	if fw.Port != 3000 {
		t.Errorf("Port = %d, want 3000", fw.Port)
	}
	if fw.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
}

func TestDetectGoFiber_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	if fw := detectGoFiber(dir); fw != nil {
		t.Errorf("got %q, want nil without go.mod", fw.Name)
	}
}

func TestDetectGoFiber_NoFiberDep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	if fw := detectGoFiber(dir); fw != nil {
		t.Errorf("got %q, want nil without fiber dep", fw.Name)
	}
}

func TestDetectGoStd_RootMainGo(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-std-root")
	fw := detectGoStd(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "go" {
		t.Errorf("Name = %q, want %q", fw.Name, "go")
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
	if fw.BuildCommand != "go build -o app ." {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectGoStd_CmdSubdir(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-std-cmd")
	fw := detectGoStd(dir)
	if fw == nil {
		t.Fatal("got nil, want framework")
	}
	if fw.Name != "go" {
		t.Errorf("Name = %q, want %q", fw.Name, "go")
	}
	if fw.BuildCommand != "go build -o app ./cmd/server" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectGoStd_NoGoMod(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures", "go-no-gomod")
	if fw := detectGoStd(dir); fw != nil {
		t.Errorf("got %q, want nil without go.mod", fw.Name)
	}
}

func TestDetectGoStd_GoModNoMain(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, dir, "handler.go", "package app\n\nfunc Handle() {}\n")
	if fw := detectGoStd(dir); fw != nil {
		t.Errorf("got %q, want nil when no main package found", fw.Name)
	}
}
