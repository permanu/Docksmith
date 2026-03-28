package detect

import (
	"testing"
)

// pom.xml fragments shared across Java test files.
const pomSpringBoot = `<project>
  <properties><java.version>17</java.version></properties>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.0</version>
  </parent>
</project>`

const pomQuarkus = `<project>
  <properties><java.version>21</java.version></properties>
  <dependencies>
    <dependency><groupId>io.quarkus</groupId><artifactId>quarkus-core</artifactId></dependency>
  </dependencies>
</project>`

const pomMicronaut = `<project>
  <properties><maven.compiler.target>17</maven.compiler.target></properties>
  <dependencies>
    <dependency><groupId>io.micronaut</groupId></dependency>
  </dependencies>
</project>`

const pomGeneric = `<project>
  <groupId>com.example</groupId>
  <artifactId>myapp</artifactId>
</project>`

const buildGradleSpringBoot = `plugins {
  id 'org.springframework.boot' version '3.2.0'
}
// spring-boot configuration
sourceCompatibility = 17`

const buildGradleQuarkus = `plugins {
  id 'io.quarkus'
}
// quarkus native build
sourceCompatibility = '21'`

const buildGradleMicronaut = `plugins {
  id 'io.micronaut.application' version '4.0.0'
}
// micronaut runtime config
sourceCompatibility = 17`

const buildGradleGeneric = `plugins {
  id 'java'
}
sourceCompatibility = 11`

// ---- detectJavaVersion ----

func TestDetectJavaVersion_PomJavaVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomSpringBoot)
	if got := detectJavaVersion(dir); got != "17" {
		t.Errorf("got %q, want 17", got)
	}
}

func TestDetectJavaVersion_PomMavenCompilerTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomMicronaut)
	if got := detectJavaVersion(dir); got != "17" {
		t.Errorf("got %q, want 17", got)
	}
}

func TestDetectJavaVersion_GradleSourceCompatibility(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", buildGradleGeneric)
	if got := detectJavaVersion(dir); got != "11" {
		t.Errorf("got %q, want 11", got)
	}
}

func TestDetectJavaVersion_GradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", `kotlin { jvmTarget = "21" }`)
	if got := detectJavaVersion(dir); got != "21" {
		t.Errorf("got %q, want 21", got)
	}
}

func TestDetectJavaVersion_JavaVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".java-version", "17.0.5\n")
	if got := detectJavaVersion(dir); got != "17" {
		t.Errorf("got %q, want 17", got)
	}
}

func TestDetectJavaVersion_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectJavaVersion(dir); got != "21" {
		t.Errorf("got %q, want 21", got)
	}
}

// ---- detectSpringBoot ----

func TestDetectSpringBoot_Maven(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomSpringBoot)
	fw := detectSpringBoot(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "spring-boot" {
		t.Errorf("Name = %q, want spring-boot", fw.Name)
	}
	if fw.BuildCommand != "mvn package -DskipTests" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
	if fw.StartCommand != "java -jar target/*.jar" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
	if fw.Port != 8080 {
		t.Errorf("Port = %d, want 8080", fw.Port)
	}
	if fw.JavaVersion != "17" {
		t.Errorf("JavaVersion = %q, want 17", fw.JavaVersion)
	}
}

func TestDetectSpringBoot_Gradle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle", buildGradleSpringBoot)
	fw := detectSpringBoot(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.BuildCommand != "gradle build -x test" {
		t.Errorf("BuildCommand = %q", fw.BuildCommand)
	}
	if fw.StartCommand != "java -jar build/libs/*.jar" {
		t.Errorf("StartCommand = %q", fw.StartCommand)
	}
}

func TestDetectSpringBoot_GradleKts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "build.gradle.kts", "// spring-boot project\nid(\"org.springframework.boot\") version \"3.2.0\"")
	fw := detectSpringBoot(dir)
	if fw == nil {
		t.Fatal("got nil")
	}
	if fw.Name != "spring-boot" {
		t.Errorf("Name = %q", fw.Name)
	}
}

func TestDetectSpringBoot_NoMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", pomGeneric)
	if fw := detectSpringBoot(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}

func TestDetectSpringBoot_AbsentBuildFiles(t *testing.T) {
	dir := t.TempDir()
	if fw := detectSpringBoot(dir); fw != nil {
		t.Errorf("got %q, want nil", fw.Name)
	}
}
