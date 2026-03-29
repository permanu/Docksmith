package detect

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// SecretDef describes a BuildKit secret mount needed for private registry auth.
type SecretDef struct {
	ID     string // secret id passed to --mount=type=secret,id=...
	Target string // target path inside the container
	Src    string // host-side source file hint for docker build --secret
}

// DetectNodeSecrets checks for .npmrc with registry auth lines or
// publishConfig in package.json pointing to a private registry.
func DetectNodeSecrets(dir string) []SecretDef {
	npmrc := filepath.Join(dir, ".npmrc")
	if fileExists(npmrc) && npmrcHasAuth(npmrc) {
		return []SecretDef{{ID: "npm", Target: "/root/.npmrc", Src: ".npmrc"}}
	}
	pkg := filepath.Join(dir, "package.json")
	if fileExists(pkg) && fileContains(pkg, `"publishConfig"`) && fileContains(pkg, `"registry"`) {
		return []SecretDef{{ID: "npm", Target: "/root/.npmrc", Src: ".npmrc"}}
	}
	return nil
}

// DetectPythonSecrets checks for pip.conf, PIP_INDEX_URL env hint, or
// poetry private sources in pyproject.toml.
func DetectPythonSecrets(dir string) []SecretDef {
	pipConf := filepath.Join(dir, "pip.conf")
	if fileExists(pipConf) {
		return []SecretDef{{ID: "pip", Target: "/root/.pip/pip.conf", Src: "pip.conf"}}
	}
	pyproject := filepath.Join(dir, "pyproject.toml")
	if fileExists(pyproject) && fileContains(pyproject, "[tool.poetry.source]") {
		return []SecretDef{{ID: "pip", Target: "/root/.pip/pip.conf", Src: "pip.conf"}}
	}
	return nil
}

// DetectGoSecrets checks for .netrc (private module proxy auth) or GOPRIVATE
// patterns in the environment/go.env.
func DetectGoSecrets(dir string) []SecretDef {
	netrc := filepath.Join(dir, ".netrc")
	if fileExists(netrc) {
		return []SecretDef{{ID: "netrc", Target: "/root/.netrc", Src: ".netrc"}}
	}
	goenv := filepath.Join(dir, "go.env")
	if fileExists(goenv) && fileContains(goenv, "GOPRIVATE") {
		return []SecretDef{{ID: "netrc", Target: "/root/.netrc", Src: ".netrc"}}
	}
	return nil
}

// DetectJavaSecrets checks for Maven settings.xml with <server> blocks
// (contains registry credentials).
func DetectJavaSecrets(dir string) []SecretDef {
	for _, rel := range []string{".mvn/settings.xml", "settings.xml"} {
		p := filepath.Join(dir, rel)
		if fileExists(p) && fileContains(p, "<server>") {
			return []SecretDef{{ID: "maven", Target: "/root/.m2/settings.xml", Src: rel}}
		}
	}
	return nil
}

// DetectRubySecrets checks for Gemfile sources pointing to a non-rubygems host.
func DetectRubySecrets(dir string) []SecretDef {
	gemfile := filepath.Join(dir, "Gemfile")
	if !fileExists(gemfile) {
		return nil
	}
	if gemfileHasPrivateSource(gemfile) {
		return []SecretDef{{ID: "bundle", Target: "/root/.bundle/config", Src: ".bundle/config"}}
	}
	return nil
}

func npmrcHasAuth(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "//registry.") || strings.HasPrefix(line, "//npm.") {
			return true
		}
		if strings.HasPrefix(line, "_authToken") || strings.HasPrefix(line, "_auth") {
			return true
		}
	}
	return false
}

func gemfileHasPrivateSource(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "source") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "rubygems.org") {
			continue
		}
		// A source line pointing elsewhere is a private registry.
		if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
			return true
		}
	}
	return false
}
