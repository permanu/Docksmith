package plan

import (
	"cmp"
	"fmt"
)

func ResolveDockerTag(runtime, version string) string {
	switch runtime {
	case "node":
		return fmt.Sprintf("node:%s-alpine", cmp.Or(version, "22"))
	case "python":
		return fmt.Sprintf("python:%s-slim", cmp.Or(version, "3.12"))
	case "go":
		return fmt.Sprintf("golang:%s-alpine", cmp.Or(version, "1.26"))
	case "ruby":
		return fmt.Sprintf("ruby:%s-slim", cmp.Or(version, "3.3"))
	case "php":
		return fmt.Sprintf("php:%s-fpm-alpine", cmp.Or(version, "8.3"))
	case "php-apache":
		return fmt.Sprintf("php:%s-apache", cmp.Or(version, "8.3"))
	case "java":
		return fmt.Sprintf("eclipse-temurin:%s-jdk-alpine", cmp.Or(version, "21"))
	case "java-jre":
		return fmt.Sprintf("eclipse-temurin:%s-jre-alpine", cmp.Or(version, "21"))
	case "dotnet-sdk":
		return fmt.Sprintf("mcr.microsoft.com/dotnet/sdk:%s", cmp.Or(version, "8.0"))
	case "dotnet-aspnet":
		return fmt.Sprintf("mcr.microsoft.com/dotnet/aspnet:%s", cmp.Or(version, "8.0"))
	case "dotnet-runtime":
		return fmt.Sprintf("mcr.microsoft.com/dotnet/runtime:%s", cmp.Or(version, "8.0"))
	case "rust":
		return fmt.Sprintf("rust:%s-alpine", cmp.Or(version, "1.85"))
	case "deno":
		return fmt.Sprintf("denoland/deno:%s", cmp.Or(version, "latest"))
	case "bun":
		// oven/bun has no -alpine variants.
		return fmt.Sprintf("oven/bun:%s", cmp.Or(version, "1"))
	case "elixir":
		return fmt.Sprintf("elixir:%s-alpine", cmp.Or(version, "1.16"))
	default:
		if version != "" {
			return fmt.Sprintf("%s:%s", runtime, version)
		}
		return runtime
	}
}
