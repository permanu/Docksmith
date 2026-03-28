package detect

import (
	"testing"
)

// ---- detectQuarkus ----

func TestDetectQuarkus_Maven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomQuarkus)
	fw := detectQuarkus(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "quarkus" {
		t.Errorf("Name = %q", fw.Name)
	}
	if fw.StartCommand != "java -jar target/quarkus-app/quarkus-run.jar" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
	if fw.JavaVersion != "21" {
		t.Errorf("JavaVersion = %q, want 21", fw.JavaVersion)
	}
}

func TestDetectQuarkus_Gradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", buildGradleQuarkus)
	fw := detectQuarkus(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.StartCommand != "java -jar build/quarkus-app/quarkus-run.jar" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectQuarkus_NoMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomGeneric)
	if fw := detectQuarkus(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

// ---- detectMicronaut ----

func TestDetectMicronaut_Maven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomMicronaut)
	fw := detectMicronaut(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "micronaut" {
		t.Errorf("Name = %q", fw.Name)
	}
	if fw.StartCommand != "java -jar target/*.jar" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectMicronaut_Gradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", buildGradleMicronaut)
	fw := detectMicronaut(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.StartCommand != "java -jar build/libs/*.jar" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectMicronaut_NoMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomGeneric)
	if fw := detectMicronaut(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

// ---- detectMavenGeneric ----

func TestDetectMavenGeneric(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomGeneric)
	fw := detectMavenGeneric(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "maven" {
		t.Errorf("Name = %q, want maven", fw.Name)
	}
	if fw.BuildCommand != "mvn package -DskipTests" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectMavenGeneric_SkipsKnownFrameworks(t *testing.T) {
	for _, pom := range []string{pomSpringBoot, pomQuarkus, pomMicronaut} {
		dir := t.TempDir()
		writeFile(t, dir, "pom.xml", pom)
		if fw := detectMavenGeneric(dir); fw != nil {
			t.Errorf("got %q, want nil for known framework", fw.Name)
		}
	}
}

func TestDetectMavenGeneric_NoPom(t *testing.T) {
	dir := t.TempDir()
	if fw := detectMavenGeneric(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

// ---- detectGradleGeneric ----

func TestDetectGradleGeneric(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", buildGradleGeneric)
	fw := detectGradleGeneric(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "gradle" {
		t.Errorf("Name = %q, want gradle", fw.Name)
	}
	if fw.BuildCommand != "gradle build -x test" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
}

func TestDetectGradleGeneric_Kts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", `plugins { id("java") }`)
	fw := detectGradleGeneric(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "gradle" {
		t.Errorf("Name = %q, want gradle", fw.Name)
	}
}

func TestDetectGradleGeneric_SkipsKnownFrameworks(t *testing.T) {
	for _, content := range []string{buildGradleSpringBoot, buildGradleQuarkus, buildGradleMicronaut} {
		dir := t.TempDir()
		writeFile(t, dir, "build.gradle", content)
		if fw := detectGradleGeneric(dir); fw != nil {
			t.Errorf("got %q, want nil for known framework", fw.Name)
		}
	}
}

func TestDetectGradleGeneric_NoBuildFile(t *testing.T) {
	dir := t.TempDir()
	if fw := detectGradleGeneric(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}
