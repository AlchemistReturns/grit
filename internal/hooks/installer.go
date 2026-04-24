package hooks

import (
	"os"
	"path/filepath"
	"strings"
)

const preCommitContent = `#!/bin/sh
if command -v grit >/dev/null 2>&1; then
    grit commit
fi
`

const preCommitMarker = "grit commit"

// Install writes or appends the pre-commit hook.
func Install(repoRoot string) error {
	return installHook(
		filepath.Join(repoRoot, ".git", "hooks", "pre-commit"),
		preCommitContent,
		preCommitMarker,
	)
}

const postRewriteContent = `#!/bin/sh
# grit post-mortem: detect reverts committed via git commit --amend
if command -v grit >/dev/null 2>&1; then
    grit revert --check
fi
`

const postRewriteMarker = "grit revert --check"

// InstallPostRewrite writes or appends the post-rewrite hook.
func InstallPostRewrite(repoRoot string) error {
	return installHook(
		filepath.Join(repoRoot, ".git", "hooks", "post-rewrite"),
		postRewriteContent,
		postRewriteMarker,
	)
}

func installHook(hookPath, content, marker string) error {
	existing, err := os.ReadFile(hookPath)
	if err == nil {
		if strings.Contains(string(existing), marker) {
			return nil // already installed
		}
		// Append to existing hook
		appended := strings.TrimRight(string(existing), "\n") + "\n\n" +
			"# grít friction logger\n" +
			content
		return os.WriteFile(hookPath, []byte(appended), 0755)
	}
	return os.WriteFile(hookPath, []byte(content), 0755)
}
