package detect

import (
	"sort"
	"testing"
)

func TestDetectPythonVersion_PythonVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".python-version", "3.11.4\n")
	if got := detectPythonVersion(dir); got != "3.11" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.11")
	}
}

func TestDetectPythonVersion_RuntimeTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "runtime.txt", "python-3.10.9\n")
	if got := detectPythonVersion(dir); got != "3.10" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.10")
	}
}

func TestDetectPythonVersion_RuntimeTxtNoPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "runtime.txt", "3.9.18")
	if got := detectPythonVersion(dir); got != "3.9" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.9")
	}
}

func TestDetectPythonVersion_PyprojectRequiresPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]
requires-python = ">=3.9,<4"
`)
	if got := detectPythonVersion(dir); got != "3.9" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.9")
	}
}

func TestDetectPythonVersion_PyprojectPythonField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[tool.poetry.dependencies]
python = "^3.11"
`)
	if got := detectPythonVersion(dir); got != "3.11" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.11")
	}
}

func TestDetectPythonVersion_Default(t *testing.T) {
	dir := t.TempDir()
	if got := detectPythonVersion(dir); got != "3.12" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.12")
	}
}

func TestDetectPythonVersion_PythonVersionFileTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".python-version", "3.8.0")
	writeFile(t, dir, "runtime.txt", "python-3.10.0")
	writeFile(t, dir, "pyproject.toml", `requires-python = ">=3.11"`)
	if got := detectPythonVersion(dir); got != "3.8" {
		t.Errorf("detectPythonVersion = %q, want %q", got, "3.8")
	}
}

func TestDetectPythonPM_UV(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")
	if got := detectPythonPM(dir); got != "uv" {
		t.Errorf("detectPythonPM = %q, want %q", got, "uv")
	}
}

func TestDetectPythonPM_Poetry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "poetry.lock", "")
	if got := detectPythonPM(dir); got != "poetry" {
		t.Errorf("detectPythonPM = %q, want %q", got, "poetry")
	}
}

func TestDetectPythonPM_PDM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pdm.lock", "")
	if got := detectPythonPM(dir); got != "pdm" {
		t.Errorf("detectPythonPM = %q, want %q", got, "pdm")
	}
}

func TestDetectPythonPM_Pipfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Pipfile", "[packages]\n")
	if got := detectPythonPM(dir); got != "pipenv" {
		t.Errorf("detectPythonPM = %q, want %q", got, "pipenv")
	}
}

func TestDetectPythonPM_PipfileLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Pipfile.lock", "{}")
	if got := detectPythonPM(dir); got != "pipenv" {
		t.Errorf("detectPythonPM = %q, want %q", got, "pipenv")
	}
}

func TestDetectPythonPM_DefaultPip(t *testing.T) {
	dir := t.TempDir()
	if got := detectPythonPM(dir); got != "pip" {
		t.Errorf("detectPythonPM = %q, want %q", got, "pip")
	}
}

func TestDetectPythonPM_UVBeatsPoetry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "uv.lock", "")
	writeFile(t, dir, "poetry.lock", "")
	if got := detectPythonPM(dir); got != "uv" {
		t.Errorf("detectPythonPM = %q, want %q", got, "uv")
	}
}

func TestPythonInstallCmd(t *testing.T) {
	tests := []struct {
		pm   string
		want string
	}{
		{"uv", "pip install uv && uv sync --frozen"},
		{"poetry", "pip install poetry && poetry install --no-interaction --no-ansi"},
		{"pdm", "pip install pdm && pdm install --no-self"},
		{"pipenv", "pip install pipenv && pipenv install --deploy --system"},
		{"pip", "pip install --no-cache-dir -r requirements.txt"},
		{"", "pip install --no-cache-dir -r requirements.txt"},
	}
	for _, tt := range tests {
		if got := pythonInstallCmd(tt.pm); got != tt.want {
			t.Errorf("pythonInstallCmd(%q) = %q, want %q", tt.pm, got, tt.want)
		}
	}
}

func TestDetectPythonSystemDeps_FromRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "psycopg2>=2.9\nDjango==4.2\n")
	got := detectPythonSystemDeps(dir)
	if len(got) != 1 || got[0] != "libpq-dev" {
		t.Errorf("detectPythonSystemDeps = %v, want [libpq-dev]", got)
	}
}

func TestDetectPythonSystemDeps_Pillow(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "Pillow==10.0.0\n")
	got := detectPythonSystemDeps(dir)
	sort.Strings(got)
	want := []string{"libjpeg-dev", "zlib1g-dev"}
	if len(got) != len(want) {
		t.Fatalf("detectPythonSystemDeps = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestDetectPythonSystemDeps_PsycopgBinaryNoLibs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "psycopg2-binary==2.9\n")
	got := detectPythonSystemDeps(dir)
	if len(got) != 0 {
		t.Errorf("detectPythonSystemDeps = %v, want empty (psycopg2-binary ships its own libs)", got)
	}
}

func TestDetectPythonSystemDeps_NoDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "requests==2.31\nflask==3.0\n")
	got := detectPythonSystemDeps(dir)
	if got != nil {
		t.Errorf("detectPythonSystemDeps = %v, want nil", got)
	}
}

func TestDetectPythonSystemDeps_FromPyproject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]
dependencies = ["lxml>=4.0"]
`)
	got := detectPythonSystemDeps(dir)
	sort.Strings(got)
	want := []string{"libxml2-dev", "libxslt1-dev"}
	if len(got) != len(want) {
		t.Fatalf("detectPythonSystemDeps = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestDetectPythonSystemDeps_FromPipfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Pipfile", `[packages]
cryptography = "*"
requests = ">=2.0"
`)
	got := detectPythonSystemDeps(dir)
	sort.Strings(got)
	want := []string{"libffi-dev", "libssl-dev"}
	if len(got) != len(want) {
		t.Fatalf("detectPythonSystemDeps = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	content := `
# comment
Django>=4.0
requests==2.31.0
-r base.txt
psycopg2[binary]>=2.9
numpy ; python_version < "3.9"
`
	got := parseRequirementsTxt(content)
	want := map[string]bool{"Django": true, "requests": true, "psycopg2": true, "numpy": true}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected package %q", p)
		}
		delete(want, p)
	}
	for p := range want {
		t.Errorf("missing package %q", p)
	}
}

func TestParsePyprojectDeps(t *testing.T) {
	content := `
[project]
dependencies = [
  "fastapi>=0.100",
  "uvicorn[standard]>=0.20",
  "sqlalchemy",
]
`
	got := parsePyprojectDeps(content)
	names := make(map[string]bool)
	for _, p := range got {
		names[p] = true
	}
	for _, want := range []string{"fastapi", "uvicorn", "sqlalchemy"} {
		if !names[want] {
			t.Errorf("missing %q in parsed deps", want)
		}
	}
}

func TestParsePipfileDeps(t *testing.T) {
	content := `
[[source]]
url = "https://pypi.org/simple"

[packages]
flask = "*"
sqlalchemy = ">=1.4"
requests = {version = ">=2.0"}

[dev-packages]
pytest = "*"
`
	got := parsePipfileDeps(content)
	names := make(map[string]bool)
	for _, p := range got {
		names[p] = true
	}
	for _, want := range []string{"flask", "sqlalchemy", "requests"} {
		if !names[want] {
			t.Errorf("missing %q in Pipfile packages", want)
		}
	}
	if names["pytest"] {
		t.Error("dev-package pytest should not appear in parsed packages")
	}
}
