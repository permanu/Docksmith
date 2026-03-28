package docksmith

import (
	"strings"
	"testing"
)

func springBootFramework() *Framework {
	return &Framework{
		Name:         "spring-boot",
		BuildCommand: "mvn package -DskipTests",
		StartCommand: "java -jar target/*.jar",
		Port:         8080,
		JavaVersion:  "21",
	}
}

func gradleFramework() *Framework {
	return &Framework{
		Name:         "gradle",
		BuildCommand: "gradle build -x test",
		StartCommand: "java -jar build/libs/*.jar",
		Port:         8080,
		JavaVersion:  "21",
	}
}

func mustPlanJava(t *testing.T, fw *Framework) *BuildPlan {
	t.Helper()
	plan, err := planJava(fw)
	if err != nil {
		t.Fatalf("planJava: %v", err)
	}
	return plan
}

// --- Maven ---

func TestPlanJava_Maven_TwoStages(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanJava_Maven_BuilderImage(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	if !strings.HasPrefix(plan.Stages[0].From, "maven:") {
		t.Errorf("maven builder from: got %q, want maven:... prefix", plan.Stages[0].From)
	}
}

func TestPlanJava_Maven_RuntimeImage(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	want := ResolveDockerTag("java-jre", "21")
	if plan.Stages[1].From != want {
		t.Errorf("runtime from: got %q, want %q", plan.Stages[1].From, want)
	}
}

func TestPlanJava_Maven_BuilderRunsMvn(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun {
			for _, arg := range step.Args {
				if strings.Contains(arg, "mvn") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("maven builder should run mvn")
	}
}

func TestPlanJava_Maven_BuilderHasCacheMount(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil && step.CacheMount.Target == "/root/.m2" {
			found = true
		}
	}
	if !found {
		t.Error("maven builder should have cache mount at /root/.m2")
	}
}

func TestPlanJava_Maven_RuntimeCopiesJAR(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCopyFrom && step.CopyFrom != nil {
			if strings.Contains(step.CopyFrom.Src, "target") {
				found = true
			}
		}
	}
	if !found {
		t.Error("maven runtime should copy JAR from target/")
	}
}

func TestPlanJava_Maven_RuntimeRunsJar(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCmd {
			for _, arg := range step.Args {
				if arg == "java" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("runtime should have CMD [java, -jar, app.jar]")
	}
}

func TestPlanJava_Maven_ValidatesOK(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

// --- Gradle ---

func TestPlanJava_Gradle_TwoStages(t *testing.T) {
	plan := mustPlanJava(t, gradleFramework())
	if len(plan.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(plan.Stages))
	}
}

func TestPlanJava_Gradle_BuilderImage(t *testing.T) {
	plan := mustPlanJava(t, gradleFramework())
	if !strings.HasPrefix(plan.Stages[0].From, "gradle:") {
		t.Errorf("gradle builder from: got %q, want gradle:... prefix", plan.Stages[0].From)
	}
}

func TestPlanJava_Gradle_RuntimeCopiesFromBuildLibs(t *testing.T) {
	plan := mustPlanJava(t, gradleFramework())
	runtime := plan.Stages[1]
	found := false
	for _, step := range runtime.Steps {
		if step.Type == StepCopyFrom && step.CopyFrom != nil {
			if strings.Contains(step.CopyFrom.Src, "build/libs") {
				found = true
			}
		}
	}
	if !found {
		t.Error("gradle runtime should copy JAR from build/libs/")
	}
}

func TestPlanJava_Gradle_CacheMount(t *testing.T) {
	plan := mustPlanJava(t, gradleFramework())
	builder := plan.Stages[0]
	found := false
	for _, step := range builder.Steps {
		if step.Type == StepRun && step.CacheMount != nil && step.CacheMount.Target == "/root/.gradle" {
			found = true
		}
	}
	if !found {
		t.Error("gradle builder should have cache mount at /root/.gradle")
	}
}

func TestPlanJava_Gradle_ValidatesOK(t *testing.T) {
	plan := mustPlanJava(t, gradleFramework())
	if err := plan.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestPlanJava_DefaultJavaVersion(t *testing.T) {
	fw := &Framework{Name: "maven", BuildCommand: "mvn package -DskipTests", Port: 8080}
	plan := mustPlanJava(t, fw)
	if !strings.Contains(plan.Stages[0].From, "21") {
		t.Errorf("expected default java 21 in builder image, got %q", plan.Stages[0].From)
	}
}

func TestPlanJava_ExposedPort(t *testing.T) {
	plan := mustPlanJava(t, springBootFramework())
	if plan.Expose != 8080 {
		t.Errorf("expose: got %d, want 8080", plan.Expose)
	}
}
