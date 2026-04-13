package plan_test

import (
	"testing"

	"github.com/permanu/docksmith/plan"
)

func TestResolveDockerTag(t *testing.T) {
	cases := []struct {
		runtime string
		version string
		want    string
	}{
		{"node", "22", "node:22-alpine"},
		{"node", "", "node:22-alpine"},
		{"python", "3.12", "python:3.12-slim"},
		{"python", "", "python:3.12-slim"},
		{"go", "1.26", "golang:1.26-alpine"},
		{"go", "", "golang:1.26-alpine"},
		{"ruby", "3.3", "ruby:3.3-slim"},
		{"ruby", "", "ruby:3.3-slim"},
		{"php", "8.3", "php:8.3-fpm-alpine"},
		{"php", "", "php:8.3-fpm-alpine"},
		{"php-apache", "8.3", "php:8.3-apache"},
		{"php-apache", "", "php:8.3-apache"},
		{"java", "21", "eclipse-temurin:21-jdk-alpine"},
		{"java", "", "eclipse-temurin:21-jdk-alpine"},
		{"java-jre", "21", "eclipse-temurin:21-jre-alpine"},
		{"java-jre", "", "eclipse-temurin:21-jre-alpine"},
		{"dotnet-sdk", "8.0", "mcr.microsoft.com/dotnet/sdk:8.0"},
		{"dotnet-sdk", "", "mcr.microsoft.com/dotnet/sdk:8.0"},
		{"dotnet-aspnet", "8.0", "mcr.microsoft.com/dotnet/aspnet:8.0"},
		{"dotnet-aspnet", "", "mcr.microsoft.com/dotnet/aspnet:8.0"},
		{"dotnet-runtime", "8.0", "mcr.microsoft.com/dotnet/runtime:8.0"},
		{"dotnet-runtime", "", "mcr.microsoft.com/dotnet/runtime:8.0"},
		{"rust", "1.85", "rust:1.85-alpine"},
		{"rust", "", "rust:1.85-alpine"},
		{"deno", "", "denoland/deno:2.1.4"},
		{"bun", "1", "oven/bun:1"},
		{"bun", "", "oven/bun:1"},
		{"elixir", "1.16", "elixir:1.16-alpine"},
		{"elixir", "", "elixir:1.16-alpine"},
		{"unknown", "1.0", "unknown:1.0"},
		{"unknown", "", "unknown"},
	}

	for _, tc := range cases {
		got := plan.ResolveDockerTag(tc.runtime, tc.version)
		if got != tc.want {
			t.Errorf("ResolveDockerTag(%q, %q) = %q, want %q", tc.runtime, tc.version, got, tc.want)
		}
	}
}
