package detect

import (
	"github.com/permanu/docksmith/core"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	RegisterDetector("aspnet-core", detectAspNetCore)
	RegisterDetector("blazor", detectBlazor)
	RegisterDetector("dotnet-worker", detectDotnetWorker)
}

func detectDotnetVersion(dir string) string {
	csproj := findCsproj(dir)
	if csproj != "" {
		if data, err := os.ReadFile(csproj); err == nil {
			re := regexp.MustCompile(`<TargetFramework>net(\d+\.\d+)(?:-[a-z]+\d*)?</TargetFramework>`)
			if m := re.FindSubmatch(data); len(m) > 1 {
				return string(m[1])
			}
		}
	}
	globalJSON := filepath.Join(dir, "global.json")
	if fileExists(globalJSON) {
		if data, err := os.ReadFile(globalJSON); err == nil {
			re := regexp.MustCompile(`"version"\s*:\s*"(\d+)\.(\d+)`)
			if m := re.FindSubmatch(data); len(m) > 2 {
				return string(m[1]) + "." + string(m[2])
			}
		}
	}
	return "8.0"
}

func findCsproj(dir string) string {
	matches, err := filepath.Glob(filepath.Join(dir, "*.csproj"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func csprojName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".csproj")
}

func dotnetWebFramework(dir, csproj, name string) *core.Framework {
	dotnetVer := detectDotnetVersion(dir)
	projectName := csprojName(csproj)
	return &core.Framework{
		Name:          name,
		BuildCommand:  "dotnet publish -c Release -o /app/publish",
		StartCommand:  fmt.Sprintf("dotnet /app/publish/%s.dll", projectName),
		Port:          8080,
		DotnetVersion: dotnetVer,
	}
}

func detectAspNetCore(dir string) *core.Framework {
	csproj := findCsproj(dir)
	if csproj == "" {
		return nil
	}
	content, err := os.ReadFile(csproj)
	if err != nil {
		return nil
	}
	s := string(content)
	// Blazor WASM check must come first — it also references Microsoft.AspNetCore.
	if strings.Contains(s, "Microsoft.AspNetCore.Components.WebAssembly") {
		return nil
	}
	if strings.Contains(s, "Microsoft.AspNetCore") {
		return dotnetWebFramework(dir, csproj, "aspnet-core")
	}
	if hasFile(dir, "Program.cs") && fileContains(filepath.Join(dir, "Program.cs"), "WebApplication") {
		return dotnetWebFramework(dir, csproj, "aspnet-core")
	}
	return nil
}

func detectBlazor(dir string) *core.Framework {
	csproj := findCsproj(dir)
	if csproj == "" {
		return nil
	}
	if fileContains(csproj, "Microsoft.AspNetCore.Components.WebAssembly") {
		return dotnetWebFramework(dir, csproj, "blazor")
	}
	return nil
}

func detectDotnetWorker(dir string) *core.Framework {
	csproj := findCsproj(dir)
	if csproj == "" {
		return nil
	}
	if fileContains(csproj, "Microsoft.AspNetCore") {
		return nil
	}
	dotnetVer := detectDotnetVersion(dir)
	projectName := csprojName(csproj)
	return &core.Framework{
		Name:          "dotnet-worker",
		BuildCommand:  "dotnet publish -c Release -o /app/publish",
		StartCommand:  fmt.Sprintf("dotnet /app/publish/%s.dll", projectName),
		Port:          0,
		DotnetVersion: dotnetVer,
	}
}
