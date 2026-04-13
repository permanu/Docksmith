package emit_test

import (
	"testing"

	"github.com/permanu/docksmith/emit"
)

func TestSanitizeDockerfileArg(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"normal string", "normal string"},
		{"has\nnewline", "has newline"},
		{"has\r\nCRLF", "has  CRLF"},
		{"multi\nline\nstring", "multi line string"},
		{"backtick`stripped", "backtickstripped"},
		{"", ""},
	}
	for _, c := range cases {
		if got := emit.SanitizeDockerfileArg(c.input); got != c.want {
			t.Errorf("SanitizeDockerfileArg(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestShellSplit(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"npm start", `"npm", "start"`},
		{"node server.js", `"node", "server.js"`},
		{"server", `"server"`},
		{"node server.js --port 3000", `"node", "server.js", "--port", "3000"`},
		{"", ""},
	}
	for _, c := range cases {
		if got := emit.ShellSplit(c.input); got != c.want {
			t.Errorf("ShellSplit(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestJSONArray(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"npm start", `["npm", "start"]`},
		{"node server.js --port 3000", `["node", "server.js", "--port", "3000"]`},
		{"server", `["server"]`},
		{"", "[]"},
	}
	for _, c := range cases {
		if got := emit.JSONArray(c.input); got != c.want {
			t.Errorf("JSONArray(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestPMCopyLockfiles(t *testing.T) {
	cases := []struct {
		pm      string
		contain string
	}{
		{"npm", "package-lock.json"},
		{"pnpm", "pnpm-lock.yaml"},
		{"yarn", "yarn.lock"},
		{"bun", "bun.lockb"},
		{"unknown", "package-lock.json"},
	}
	for _, c := range cases {
		got := emit.PMCopyLockfiles(c.pm)
		if got == "" {
			t.Errorf("PMCopyLockfiles(%q) returned empty string", c.pm)
			continue
		}
		found := false
		for i := range len(got) - len(c.contain) + 1 {
			if got[i:i+len(c.contain)] == c.contain {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PMCopyLockfiles(%q) = %q, want it to contain %q", c.pm, got, c.contain)
		}
	}
}
