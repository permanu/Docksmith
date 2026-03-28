package detect

import (
	"github.com/permanu/docksmith/core"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	RegisterDetector("spring-boot", detectSpringBoot)
	RegisterDetector("quarkus", detectQuarkus)
	RegisterDetector("micronaut", detectMicronaut)
	RegisterDetector("maven", detectMavenGeneric)
	RegisterDetector("gradle", detectGradleGeneric)
}

func detectJavaVersion(dir string) string {
	pomPath := filepath.Join(dir, "pom.xml")
	if fileExists(pomPath) {
		if data, err := os.ReadFile(pomPath); err == nil {
			s := string(data)
			re1 := regexp.MustCompile(`<java\.version>(\d+)</java\.version>`)
			if m := re1.FindStringSubmatch(s); len(m) > 1 {
				return m[1]
			}
			re2 := regexp.MustCompile(`<maven\.compiler\.target>(\d+)</maven\.compiler\.target>`)
			if m := re2.FindStringSubmatch(s); len(m) > 1 {
				return m[1]
			}
		}
	}
	for _, name := range []string{"build.gradle", "build.gradle.kts"} {
		gpath := filepath.Join(dir, name)
		if !fileExists(gpath) {
			continue
		}
		if data, err := os.ReadFile(gpath); err == nil {
			s := string(data)
			re1 := regexp.MustCompile(`sourceCompatibility\s*=\s*(?:JavaVersion\.VERSION_)?['"]*(\d+)`)
			if m := re1.FindStringSubmatch(s); len(m) > 1 {
				return m[1]
			}
			re2 := regexp.MustCompile(`jvmTarget\s*=\s*['"]+(\d+)`)
			if m := re2.FindStringSubmatch(s); len(m) > 1 {
				return m[1]
			}
		}
	}
	jvPath := filepath.Join(dir, ".java-version")
	if fileExists(jvPath) {
		if data, err := os.ReadFile(jvPath); err == nil {
			ver := strings.TrimSpace(string(data))
			if parts := strings.SplitN(ver, ".", 2); len(parts) > 0 && parts[0] != "" {
				return parts[0]
			}
		}
	}
	return "21"
}

func detectSpringBoot(dir string) *core.Framework {
	jv := detectJavaVersion(dir)
	if hasFile(dir, "pom.xml") && fileContains(filepath.Join(dir, "pom.xml"), "spring-boot") {
		return &core.Framework{
			Name:         "spring-boot",
			BuildCommand: "mvn package -DskipTests",
			StartCommand: "java -jar target/*.jar",
			Port:         8080,
			JavaVersion:  jv,
		}
	}
	for _, f := range []string{"build.gradle", "build.gradle.kts"} {
		if hasFile(dir, f) && fileContains(filepath.Join(dir, f), "spring-boot") {
			return &core.Framework{
				Name:         "spring-boot",
				BuildCommand: "gradle build -x test",
				StartCommand: "java -jar build/libs/*.jar",
				Port:         8080,
				JavaVersion:  jv,
			}
		}
	}
	return nil
}

func detectQuarkus(dir string) *core.Framework {
	jv := detectJavaVersion(dir)
	if hasFile(dir, "pom.xml") && fileContains(filepath.Join(dir, "pom.xml"), "quarkus") {
		return &core.Framework{
			Name:         "quarkus",
			BuildCommand: "mvn package -DskipTests",
			StartCommand: "java -jar target/quarkus-app/quarkus-run.jar",
			Port:         8080,
			JavaVersion:  jv,
		}
	}
	for _, f := range []string{"build.gradle", "build.gradle.kts"} {
		if hasFile(dir, f) && fileContains(filepath.Join(dir, f), "quarkus") {
			return &core.Framework{
				Name:         "quarkus",
				BuildCommand: "gradle build -x test",
				StartCommand: "java -jar build/quarkus-app/quarkus-run.jar",
				Port:         8080,
				JavaVersion:  jv,
			}
		}
	}
	return nil
}

func detectMicronaut(dir string) *core.Framework {
	jv := detectJavaVersion(dir)
	if hasFile(dir, "pom.xml") && fileContains(filepath.Join(dir, "pom.xml"), "micronaut") {
		return &core.Framework{
			Name:         "micronaut",
			BuildCommand: "mvn package -DskipTests",
			StartCommand: "java -jar target/*.jar",
			Port:         8080,
			JavaVersion:  jv,
		}
	}
	for _, f := range []string{"build.gradle", "build.gradle.kts"} {
		if hasFile(dir, f) && fileContains(filepath.Join(dir, f), "micronaut") {
			return &core.Framework{
				Name:         "micronaut",
				BuildCommand: "gradle build -x test",
				StartCommand: "java -jar build/libs/*.jar",
				Port:         8080,
				JavaVersion:  jv,
			}
		}
	}
	return nil
}

func detectMavenGeneric(dir string) *core.Framework {
	if !hasFile(dir, "pom.xml") {
		return nil
	}
	pom := filepath.Join(dir, "pom.xml")
	if fileContains(pom, "spring-boot") || fileContains(pom, "quarkus") || fileContains(pom, "micronaut") {
		return nil
	}
	return &core.Framework{
		Name:         "maven",
		BuildCommand: "mvn package -DskipTests",
		StartCommand: "java -jar target/*.jar",
		Port:         8080,
		JavaVersion:  detectJavaVersion(dir),
	}
}

func detectGradleGeneric(dir string) *core.Framework {
	gradleFile := ""
	if hasFile(dir, "build.gradle") {
		gradleFile = filepath.Join(dir, "build.gradle")
	} else if hasFile(dir, "build.gradle.kts") {
		gradleFile = filepath.Join(dir, "build.gradle.kts")
	}
	if gradleFile == "" {
		return nil
	}
	if fileContains(gradleFile, "spring-boot") || fileContains(gradleFile, "quarkus") || fileContains(gradleFile, "micronaut") {
		return nil
	}
	return &core.Framework{
		Name:         "gradle",
		BuildCommand: "gradle build -x test",
		StartCommand: "java -jar build/libs/*.jar",
		Port:         8080,
		JavaVersion:  detectJavaVersion(dir),
	}
}
