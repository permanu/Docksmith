package docksmith

import "strings"

// planDotnet builds a two-stage BuildPlan for .NET applications.
// Stage "builder" uses the .NET SDK to publish the app.
// Stage "runtime" uses a lean ASP.NET or dotnet-runtime image.
// Workers (dotnet-worker) use dotnet-runtime; all web variants use dotnet-aspnet.
func planDotnet(fw *Framework) (*BuildPlan, error) {
	dotnetVer := fw.DotnetVersion
	if dotnetVer == "" {
		dotnetVer = "8.0"
	}

	sdkImage := ResolveDockerTag("dotnet-sdk", dotnetVer)

	isWorker := fw.Name == "dotnet-worker"

	var runtimeImage string
	if isWorker {
		runtimeImage = ResolveDockerTag("dotnet-runtime", dotnetVer)
	} else {
		runtimeImage = ResolveDockerTag("dotnet-aspnet", dotnetVer)
	}

	// Extract the project name from the StartCommand: "dotnet /app/publish/MyApp.dll" -> "MyApp"
	projectName := extractDotnetProjectName(fw.StartCommand)

	builder := Stage{
		Name: "builder",
		From: sdkImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/src"}},
			{Type: StepCopy, Args: []string{"*.csproj", "./"}},
			{
				Type:       StepRun,
				Args:       []string{"dotnet restore"},
				CacheMount: &CacheMount{Target: "/root/.nuget/packages"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
			{Type: StepRun, Args: []string{"dotnet publish -c Release -o /app/publish"}},
		},
	}

	runtime := Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app/publish", Dst: "."},
				Link:     true,
			},
		},
	}

	if isWorker {
		runtime.Steps = append(runtime.Steps,
			Step{Type: StepEntrypoint, Args: []string{"dotnet", projectName + ".dll"}},
		)
		expose := fw.Port
		if expose <= 0 {
			expose = 8080
		}
		return &BuildPlan{
			Framework:    fw.Name,
			Stages:       []Stage{builder, runtime},
			Expose:       expose,
			Dockerignore: []string{".git", "bin", "obj", "*.log"},
		}, nil
	}

	port := fw.Port
	if port == 0 {
		port = 8080
	}

	runtime.Steps = append(runtime.Steps,
		Step{Type: StepEnv, Args: []string{"ASPNETCORE_URLS", "http://+:8080"}},
		Step{Type: StepExpose, Args: []string{"8080"}},
		Step{Type: StepEntrypoint, Args: []string{"dotnet", projectName + ".dll"}},
	)

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "bin", "obj", "*.log"},
	}, nil
}

// extractDotnetProjectName pulls the project name from a dotnet start command.
// "dotnet /app/publish/MyApp.dll" -> "MyApp"
// Falls back to "app" when the command cannot be parsed.
func extractDotnetProjectName(startCmd string) string {
	if startCmd == "" {
		return "app"
	}
	parts := strings.Fields(startCmd)
	for _, p := range parts {
		if strings.HasSuffix(p, ".dll") {
			base := p[strings.LastIndex(p, "/")+1:]
			return strings.TrimSuffix(base, ".dll")
		}
	}
	return "app"
}
