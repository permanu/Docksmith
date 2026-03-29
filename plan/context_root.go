package plan

import (
	"path"
	"strings"

	"github.com/permanu/docksmith/core"
)

// applyContextRoot rewrites COPY paths so they reference files within
// appSubdir instead of the build context root. This is the key adjustment
// for monorepo builds where the Docker build context is the repo root but
// the app lives in a subdirectory.
//
// Concretely: `COPY . .` becomes `COPY ./apps/frontend .`, and lockfile
// copies like `COPY package.json ./` become `COPY ./apps/frontend/package.json ./`.
func applyContextRoot(plan *core.BuildPlan, appSubdir string) {
	if appSubdir == "" {
		return
	}
	// Normalize: no leading slash, no trailing slash, forward slashes.
	appSubdir = strings.TrimPrefix(appSubdir, "/")
	appSubdir = strings.TrimSuffix(appSubdir, "/")

	for i := range plan.Stages {
		for j := range plan.Stages[i].Steps {
			step := &plan.Stages[i].Steps[j]
			if step.Type != core.StepCopy {
				continue
			}
			rewriteCopyArgs(step, appSubdir)
		}
	}
}

// rewriteCopyArgs adjusts a COPY step's source args to be relative to the
// context root. The last element in Args is the destination; everything
// before it is a source path.
//
// Rewrite rules:
//   - "." or "./" (whole-app copy) -> "./{appSubdir}"
//   - "package.json" (specific file) -> "./{appSubdir}/package.json"
//   - Glob patterns like "*.lock*" -> "./{appSubdir}/*.lock*"
//   - Paths already prefixed with "./" that aren't the destination are prefixed
//   - Absolute paths (inside the container from a prior stage) are left alone
func rewriteCopyArgs(step *core.Step, appSubdir string) {
	args := step.Args
	if len(args) < 2 {
		return
	}
	// Last arg is the destination — leave it unchanged.
	for i := 0; i < len(args)-1; i++ {
		src := args[i]
		// Absolute container paths (e.g. /app/dist) are not context-relative.
		if strings.HasPrefix(src, "/") {
			continue
		}
		args[i] = prefixWithSubdir(src, appSubdir)
	}
}

func prefixWithSubdir(src, appSubdir string) string {
	// "." -> "./apps/frontend"
	if src == "." || src == "./" {
		return "./" + appSubdir
	}
	// Already has explicit relative prefix: "./foo" -> "./apps/frontend/foo"
	if strings.HasPrefix(src, "./") {
		inner := strings.TrimPrefix(src, "./")
		return "./" + path.Join(appSubdir, inner)
	}
	// Bare filename or glob: "package.json" -> "./apps/frontend/package.json"
	return "./" + path.Join(appSubdir, src)
}
