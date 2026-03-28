package docksmith

import (
	"strconv"
	"strings"
)

// planJava builds a two-stage BuildPlan for Java applications.
// Stage "builder" compiles with Maven or Gradle.
// Stage "runtime" uses a lean JRE image with the packaged JAR.
func planJava(fw *Framework) (*BuildPlan, error) {
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

func planJavaMaven(fw *Framework, javaVer string, port int) (*BuildPlan, error) {
	builderImage := "maven:3.9-eclipse-temurin-" + javaVer
	runtimeImage := ResolveDockerTag("java-jre", javaVer)

	buildCmd := fw.BuildCommand
	if buildCmd == "" {
		buildCmd = "mvn package -DskipTests"
	}

	builder := Stage{
		Name: "builder",
		From: builderImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/app"}},
			{Type: StepCopy, Args: []string{"pom.xml", "./"}},
			{
				Type:       StepRun,
				Args:       []string{"mvn dependency:go-offline -B"},
				CacheMount: &CacheMount{Target: "/root/.m2"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
			{Type: StepRun, Args: []string{buildCmd}},
		},
	}

	runtime := Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app/target/*.jar", Dst: "app.jar"},
				Link:     true,
			},
			{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: StepCmd, Args: []string{"java", "-jar", "app.jar"}},
		},
	}

	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "java", port)

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "target", "*.log"},
	}, nil
}

func planJavaGradle(fw *Framework, javaVer string, port int) (*BuildPlan, error) {
	builderImage := "gradle:8-jdk" + javaVer
	runtimeImage := ResolveDockerTag("java-jre", javaVer)

	buildCmd := fw.BuildCommand
	if buildCmd == "" {
		buildCmd = "gradle build -x test"
	}

	builder := Stage{
		Name: "builder",
		From: builderImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/app"}},
			{Type: StepCopy, Args: []string{"build.gradle*", "settings.gradle*", "gradlew*", "./"}},
			{
				Type:       StepRun,
				Args:       []string{"gradle dependencies --no-daemon 2>/dev/null || true"},
				CacheMount: &CacheMount{Target: "/root/.gradle"},
			},
			{Type: StepCopy, Args: []string{".", "."}},
			{Type: StepRun, Args: []string{buildCmd}},
		},
	}

	runtime := Stage{
		Name: "runtime",
		From: runtimeImage,
		Steps: []Step{
			{Type: StepWorkdir, Args: []string{"/app"}},
			{
				Type:     StepCopyFrom,
				CopyFrom: &CopyFrom{Stage: "builder", Src: "/app/build/libs/*.jar", Dst: "app.jar"},
				Link:     true,
			},
			{Type: StepExpose, Args: []string{strconv.Itoa(port)}},
			{Type: StepCmd, Args: []string{"java", "-jar", "app.jar"}},
		},
	}

	addNonRootUser(&runtime, "")
	addHealthcheck(&runtime, "java", port)

	return &BuildPlan{
		Framework:    fw.Name,
		Stages:       []Stage{builder, runtime},
		Expose:       port,
		Dockerignore: []string{".git", "build", ".gradle", "*.log"},
	}, nil
}
