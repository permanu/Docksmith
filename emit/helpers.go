package emit

import (
	"fmt"
	"strings"
)

// SanitizeDockerfileArg strips newlines and carriage returns to prevent injection
// of additional Dockerfile instructions when user-supplied strings are interpolated.
func SanitizeDockerfileArg(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "$(", "")
	return s
}

// ShellSplit quotes each word for CMD array elements: "npm start" => "npm", "start".
func ShellSplit(cmd string) string {
	parts := strings.Fields(cmd)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return strings.Join(quoted, ", ")
}

// JSONArray converts a command to Dockerfile exec form: "npm start" => ["npm", "start"].
func JSONArray(cmd string) string {
	parts := strings.Fields(cmd)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// PMCopyLockfiles returns the COPY instruction for a package manager's manifest and lockfile.
func PMCopyLockfiles(pm string) string {
	switch pm {
	case "pnpm":
		return "COPY package.json pnpm-lock.yaml* ./\n"
	case "yarn":
		return "COPY package.json yarn.lock* ./\n"
	case "bun":
		return "COPY package.json bun.lockb* bun.lock* ./\n"
	default:
		return "COPY package.json package-lock.json* ./\n"
	}
}

// SanitizeArgs applies SanitizeDockerfileArg to each element.
func SanitizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = SanitizeDockerfileArg(a)
	}
	return out
}
