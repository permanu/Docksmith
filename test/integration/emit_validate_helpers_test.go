package integration_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// validInstructions are the standard Dockerfile instructions per the spec.
var validInstructions = map[string]bool{
	"FROM": true, "RUN": true, "COPY": true, "ADD": true,
	"WORKDIR": true, "ENV": true, "ARG": true, "EXPOSE": true,
	"USER": true, "CMD": true, "ENTRYPOINT": true, "HEALTHCHECK": true,
	"LABEL": true, "SHELL": true, "STOPSIGNAL": true, "VOLUME": true,
	"ONBUILD": true,
}

var stageNameRe = regexp.MustCompile(`(?i)^FROM\s+\S+\s+AS\s+(\S+)`)
var copyFromRe = regexp.MustCompile(`COPY\s+--from=(\S+)`)

// dockerfileIssue captures a single validation problem.
type dockerfileIssue struct {
	Line    int
	Kind    string // "syntax", "hardening", "security"
	Message string
}

func (i dockerfileIssue) String() string {
	return fmt.Sprintf("line %d [%s]: %s", i.Line, i.Kind, i.Message)
}

// validateDockerfileSyntax checks that every line is valid Dockerfile syntax.
func validateDockerfileSyntax(dockerfile string) []dockerfileIssue {
	lines := strings.Split(dockerfile, "\n")
	var issues []dockerfileIssue
	stageNames := map[string]bool{}
	continuation := false

	fromCount := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continuation = false
			continue
		}
		if continuation || strings.HasPrefix(trimmed, "&&") || strings.HasPrefix(trimmed, "||") {
			continuation = strings.HasSuffix(trimmed, "\\")
			continue
		}
		instruction := extractInstruction(trimmed)
		if instruction == "" || !validInstructions[instruction] {
			issues = append(issues, dockerfileIssue{
				Line: i + 1, Kind: "syntax",
				Message: fmt.Sprintf("unknown instruction %q", firstWord(trimmed)),
			})
			continue
		}
		if instruction == "FROM" {
			fromCount++
			issues = append(issues, validateFromLine(trimmed, i+1, stageNames)...)
		}
		continuation = strings.HasSuffix(trimmed, "\\")
	}

	if fromCount == 0 {
		issues = append(issues, dockerfileIssue{Kind: "syntax", Message: "no FROM instruction found"})
	}
	issues = append(issues, validateCopyFromRefs(lines, stageNames)...)
	return issues
}

func validateFromLine(line string, lineNum int, stageNames map[string]bool) []dockerfileIssue {
	var issues []dockerfileIssue
	rest := strings.TrimSpace(line[len("FROM"):])
	if rest == "" {
		issues = append(issues, dockerfileIssue{
			Line: lineNum, Kind: "syntax", Message: "FROM without image reference",
		})
	}
	if m := stageNameRe.FindStringSubmatch(line); m != nil {
		name := m[1]
		if stageNames[name] {
			issues = append(issues, dockerfileIssue{
				Line: lineNum, Kind: "syntax",
				Message: fmt.Sprintf("duplicate stage name %q", name),
			})
		}
		stageNames[name] = true
	}
	return issues
}

func validateCopyFromRefs(lines []string, stageNames map[string]bool) []dockerfileIssue {
	var issues []dockerfileIssue
	for i, line := range lines {
		matches := copyFromRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			ref := m[1]
			if isNumericRef(ref) {
				continue
			}
			if !stageNames[ref] {
				issues = append(issues, dockerfileIssue{
					Line: i + 1, Kind: "syntax",
					Message: fmt.Sprintf("COPY --from=%s references undefined stage", ref),
				})
			}
		}
	}
	return issues
}

func isNumericRef(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// extractInstruction returns the uppercase Dockerfile instruction from a line,
// or empty string if it doesn't look like one.
func extractInstruction(line string) string {
	word := firstWord(line)
	upper := strings.ToUpper(word)
	if validInstructions[upper] {
		return upper
	}
	return ""
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexAny(s, " \t"); idx > 0 {
		return s[:idx]
	}
	return s
}

// validateHardening checks production hardening of the final stage.
func validateHardening(dockerfile string) []dockerfileIssue {
	finalStage := extractFinalStage(dockerfile)
	var issues []dockerfileIssue

	if !stageHasInstruction(finalStage, "USER") {
		issues = append(issues, dockerfileIssue{
			Kind: "hardening", Message: "no USER instruction in final stage (running as root)",
		})
	}
	if !stageHasInstruction(finalStage, "HEALTHCHECK") {
		issues = append(issues, dockerfileIssue{
			Kind: "hardening", Message: "no HEALTHCHECK in final stage",
		})
	}
	if !stageHasInstruction(finalStage, "WORKDIR") {
		issues = append(issues, dockerfileIssue{
			Kind: "hardening", Message: "no WORKDIR set in final stage",
		})
	}
	if !stageHasInstruction(finalStage, "CMD") && !stageHasInstruction(finalStage, "ENTRYPOINT") {
		issues = append(issues, dockerfileIssue{
			Kind: "hardening", Message: "no CMD or ENTRYPOINT in final stage",
		})
	}
	return issues
}

// extractFinalStage returns the lines belonging to the last FROM block.
func extractFinalStage(dockerfile string) []string {
	lines := strings.Split(dockerfile, "\n")
	lastFrom := 0
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "FROM ") {
			lastFrom = i
		}
	}
	return lines[lastFrom:]
}

func stageHasInstruction(lines []string, instruction string) bool {
	prefix := instruction + " "
	bracket := instruction + " ["
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, bracket) {
			return true
		}
	}
	return false
}

// secretPatterns catches ENV/ARG lines that leak secrets.
var secretPatterns = []string{
	"password=", "token=", "secret=", "api_key=", "apikey=",
	"private_key=", "access_key=",
}

// secretFilePatterns catches COPY of sensitive files without secret mounts.
var secretFilePatterns = []string{
	".env", ".npmrc", "id_rsa", ".pem", ".key",
}

// validateSecurity checks for secrets in ENV/ARG and dangerous COPY targets.
func validateSecurity(dockerfile string) []dockerfileIssue {
	lines := strings.Split(dockerfile, "\n")
	var issues []dockerfileIssue

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(trimmed, "ENV ") || strings.HasPrefix(trimmed, "ARG ") {
			for _, pat := range secretPatterns {
				if strings.Contains(lower, pat) {
					issues = append(issues, dockerfileIssue{
						Line: i + 1, Kind: "security",
						Message: fmt.Sprintf("potential secret in %s (matched %q)", firstWord(trimmed), pat),
					})
				}
			}
		}

		// COPY of secret files without --mount=type=secret.
		if strings.HasPrefix(trimmed, "COPY ") && !strings.Contains(trimmed, "mount=type=secret") {
			for _, pat := range secretFilePatterns {
				if strings.Contains(trimmed, pat) {
					issues = append(issues, dockerfileIssue{
						Line: i + 1, Kind: "security",
						Message: fmt.Sprintf("COPY of sensitive file pattern %q without secret mount", pat),
					})
				}
			}
		}
	}
	return issues
}

// assertDockerfileValid runs all three validation passes and fails the test
// if any issues are found. Dumps the full Dockerfile on failure.
func assertDockerfileValid(t *testing.T, dockerfile, runtime string, skipHardening map[string]bool) {
	t.Helper()

	syntax := validateDockerfileSyntax(dockerfile)
	for _, issue := range syntax {
		t.Errorf("[%s] %s", runtime, issue)
	}

	hardening := validateHardening(dockerfile)
	for _, issue := range hardening {
		if skipHardening[issue.Message] {
			continue
		}
		t.Errorf("[%s] %s", runtime, issue)
	}

	security := validateSecurity(dockerfile)
	for _, issue := range security {
		t.Errorf("[%s] %s", runtime, issue)
	}

	if t.Failed() {
		t.Logf("Full Dockerfile for %s:\n%s", runtime, dockerfile)
	}
}
