package plan

import (
	"github.com/permanu/docksmith/core"
	"strconv"
	"strings"
)

// planJava builds a two-stage BuildPlan for Java applications.
// Stage "builder" compiles with Maven or Gradle.
// Stage "runtime" uses a lean JRE image with the packaged JAR.
func planJava(fw *core.Framework) (*core.BuildPlan, error) {
	javaVer := fw.JavaVersion
	if javaVer == "" {
		javaVer = "21"
	}
	port := fw.Port
	if port == 0 {
		port = 8080
	}

	if strings.Contains(fw.BuildCommand, "gradle") {
		return planJavaGradle(fw, javaVer, port)
	}
	return planJavaMaven(fw, javaVer, port)
}

func planJavaMaven(fw *core.Framework, javaVer string, port int) (*core.BuildPlan, error) {
	builderImage := "maven:3.9-eclipse-temurin-" + javaVer
	runtimeImage := ResolveDockerTag("java-jre", javaVer)

	buildCmd := fw.BuildCommand
	if buildCmd == "" {
		buildCmd = "mvn package -DskipTests"
	}

	builder := core.Stage{
		Name: "builder",
		From: builderImage,
		Steps: []core.Step{
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{Type: core.StepCopy, Args: []string{"pom.xml", "./"}},
			{
				Type:       core.StepRun,
				Args:       []string{"mvn dependency:go-offline -B"},
				CacheMount: &core.CacheMount{Target: "/root/.m2"},
			},
			{Type: core.StepCopy, Args: []string{".", "."}},
			{Type: core.StepRun, Args: []string{buildCmd}},
		},
	}

	runtime := core.Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []core.Step{
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/app/target/*.jar", Dst: "app.jar"},
				Link:     true,
			},
			{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: core.StepCmd, Args: []string{"java", "-jar", "app.jar"}},
		},
	}

	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "java", port)

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "target", "*.log"},
	}, nil
}

func planJavaGradle(fw *core.Framework, javaVer string, port int) (*core.BuildPlan, error) {
	builderImage := "gradle:8-jdk" + javaVer
	runtimeImage := ResolveDockerTag("java-jre", javaVer)

	buildCmd := fw.BuildCommand
	if buildCmd == "" {
		buildCmd = "gradle build -x test"
	}

	builder := core.Stage{
		Name: "builder",
		From: builderImage,
		Steps: []core.Step{
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{Type: core.StepCopy, Args: []string{"build.gradle*", "settings.gradle*", "gradlew*", "./"}},
			{
				Type:       core.StepRun,
				Args:       []string{"gradle dependencies --no-daemon 2>/dev/null || true"},
				CacheMount: &core.CacheMount{Target: "/root/.gradle"},
			},
			{Type: core.StepCopy, Args: []string{".", "."}},
			{Type: core.StepRun, Args: []string{buildCmd}},
		},
	}

	runtime := core.Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []core.Step{
			{Type: core.StepWorkdir, Args: []string{"/app"}},
			{
				Type:     core.StepCopyFrom,
				CopyFrom: &core.CopyFrom{Stage: "builder", Src: "/app/build/libs/*.jar", Dst: "app.jar"},
				Link:     true,
			},
			{Type: core.StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: core.StepCmd, Args: []string{"java", "-jar", "app.jar"}},
		},
	}

	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "java", port)

	return &core.BuildPlan{
		Framework:    fw.Name,
		Stages:       []core.Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "build", ".gradle", "*.log"},
	}, nil
}
