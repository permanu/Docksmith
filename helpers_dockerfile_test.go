package docksmith

import (
	"testing"
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
		{"backtick`preserved", "backtick`preserved"},
		{"", ""},
	}
	for _, c := range cases {
		if got := sanitizeDockerfileArg(c.input); got != c.want {
			t.Errorf("sanitizeDockerfileArg(%q) = %q, want %q", c.input, got, c.want)
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
		if got := shellSplit(c.input); got != c.want {
			t.Errorf("shellSplit(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestJsonArray(t *testing.T) {
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
		if got := jsonArray(c.input); got != c.want {
			t.Errorf("jsonArray(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestPmCopyLockfiles(t *testing.T) {
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
		got := pmCopyLockfiles(c.pm)
		if got == "" {
			t.Errorf("pmCopyLockfiles(%q) returned empty string", c.pm)
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
			t.Errorf("pmCopyLockfiles(%q) = %q, want it to contain %q", c.pm, got, c.contain)
		}
	}
}
