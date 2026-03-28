package docksmith

import (
	"fmt"
	"strings"
)

// sanitizeDockerfileArg strips newlines and carriage returns to prevent injection
// of additional Dockerfile instructions when user-supplied strings are interpolated.
func sanitizeDockerfileArg(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// shellSplit quotes each word of cmd and joins with ", " for use inside a CMD array literal.
func shellSplit(cmd string) string {
	parts := strings.Fields(cmd)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return strings.Join(quoted, ", ")
}

// jsonArray converts cmd into a JSON array string suitable for Dockerfile CMD.
func jsonArray(cmd string) string {
	parts := strings.Fields(cmd)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// pmCopyLockfiles returns the COPY instruction for a package manager's manifest and lockfile.
func pmCopyLockfiles(pm string) string {
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
