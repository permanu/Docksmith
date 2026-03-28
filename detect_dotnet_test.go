package docksmith

import (
	"testing"
)

const csprojAspNet = `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.OpenApi" Version="8.0.0" />
  </ItemGroup>
</Project>`

const csprojBlazor = `<Project Sdk="Microsoft.NET.Sdk.BlazorWebAssembly">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.Components.WebAssembly" Version="8.0.0" />
  </ItemGroup>
</Project>`

const csprojWorker = `<Project Sdk="Microsoft.NET.Sdk.Worker">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`

// ---- detectDotnetVersion ----

func TestDetectDotnetVersion_Csproj(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "MyApp.csproj", csprojAspNet)
	if got := detectDotnetVersion(dir); got != "8.0" {
		t.Errorf("got %q, want 8.0", got)
	}
}

func TestDetectDotnetVersion_CsprojWithOSMoniker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "App.csproj", `<Project><PropertyGroup><TargetFramework>net7.0-windows</TargetFramework></PropertyGroup></Project>`)
	if got := detectDotnetVersion(dir); got != "7.0" {
		t.Errorf("got %q, want 7.0", got)
	}
}

func TestDetectDotnetVersion_GlobalJson(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "global.json", `{"sdk":{"version":"6.0.400"}}`)
	if got := detectDotnetVersion(dir); got != "6.0" {
		t.Errorf("got %q, want 6.0", got)
	}
}

func TestDetectDotnetVersion_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectDotnetVersion(dir); got != "8.0" {
		t.Errorf("got %q, want 8.0", got)
	}
}

// ---- findCsproj / csprojName ----

func TestFindCsproj_Found(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "MyApp.csproj", "<Project/>")
	if got := findCsproj(dir); got == "" {
		t.Error("got empty, want path")
	}
}

func TestFindCsproj_NotFound(t *testing.T) {
	dir := t.TempDir()
	if got := findCsproj(dir); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestCsprojName(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/projects/MyApp/MyApp.csproj", "MyApp"},
		{"/a/b/WebApi.csproj", "WebApi"},
	}
	for _, c := range cases {
		if got := csprojName(c.path); got != c.want {
			t.Errorf("csprojName(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// ---- detectAspNetCore ----

func TestDetectAspNetCore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "MyWebApp.csproj", csprojAspNet)
	fw := detectAspNetCore(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "aspnet-core" {
		t.Errorf("Name = %q, want aspnet-core", fw.Name)
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
	if fw.DotnetVersion != "8.0" {
		t.Errorf("DotnetVersion = %q, want 8.0", fw.DotnetVersion)
	}
	if fw.StartCommand != "dotnet /app/publish/MyWebApp.dll" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
	if fw.BuildCommand != "dotnet publish -c Release -o /app/publish" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectAspNetCore_ViaProgram(t *testing.T) {
	dir := t.TempDir()
	// .csproj without explicit AspNetCore reference but Program.cs uses WebApplication
	writeFile(t, dir, "Api.csproj", `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup>
</Project>`)
	writeFile(t, dir, "Program.cs", `var builder = WebApplication.CreateBuilder(args);`)
	fw := detectAspNetCore(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "aspnet-core" {
		t.Errorf("Name = %q", fw.Name)
	}
}

func TestDetectAspNetCore_SkipsBlazor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "App.csproj", csprojBlazor)
	if fw := detectAspNetCore(dir); fw != nil {
		t.Errorf("got %q, want nil for Blazor project", fw.Name)
	}
}

func TestDetectAspNetCore_NoCsproj(t *testing.T) {
	dir := t.TempDir()
	if fw := detectAspNetCore(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

// ---- detectBlazor ----

func TestDetectBlazor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "BlazorApp.csproj", csprojBlazor)
	fw := detectBlazor(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "blazor" {
		t.Errorf("Name = %q, want blazor", fw.Name)
	}
	if fw.StartCommand != "dotnet /app/publish/BlazorApp.dll" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectBlazor_NonBlazorCsproj(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "App.csproj", csprojAspNet)
	if fw := detectBlazor(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectBlazor_NoCsproj(t *testing.T) {
	dir := t.TempDir()
	if fw := detectBlazor(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

// ---- detectDotnetWorker ----

func TestDetectDotnetWorker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "WorkerService.csproj", csprojWorker)
	fw := detectDotnetWorker(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "dotnet-worker" {
		t.Errorf("Name = %q, want dotnet-worker", fw.Name)
	}
	if fw.Port != 0 {
		t.Errorf("Port = %d, want 0", fw.Port)
	}
	if fw.StartCommand != "dotnet /app/publish/WorkerService.dll" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
	if fw.DotnetVersion != "8.0" {
		t.Errorf("DotnetVersion = %q, want 8.0", fw.DotnetVersion)
	}
}

func TestDetectDotnetWorker_SkipsAspNet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "App.csproj", csprojAspNet)
	if fw := detectDotnetWorker(dir); fw != nil {
		t.Errorf("got %q, want nil for AspNetCore project", fw.Name)
	}
}

func TestDetectDotnetWorker_NoCsproj(t *testing.T) {
	dir := t.TempDir()
	if fw := detectDotnetWorker(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}
