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

// Uninstall removes the pre-commit hook.
func Uninstall(repoRoot string) error {
	return uninstallHook(
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

// UninstallPostRewrite removes the post-rewrite hook.
func UninstallPostRewrite(repoRoot string) error {
	return uninstallHook(
		filepath.Join(repoRoot, ".git", "hooks", "post-rewrite"),
		postRewriteContent,
		postRewriteMarker,
	)
}

const postCommitContent = `#!/bin/sh
if command -v grit >/dev/null 2>&1; then
    grit post-commit
fi
`

const postCommitMarker = "grit post-commit"

// InstallPostCommit writes or appends the post-commit hook.
func InstallPostCommit(repoRoot string) error {
	return installHook(
		filepath.Join(repoRoot, ".git", "hooks", "post-commit"),
		postCommitContent,
		postCommitMarker,
	)
}

// UninstallPostCommit removes the post-commit hook.
func UninstallPostCommit(repoRoot string) error {
	return uninstallHook(
		filepath.Join(repoRoot, ".git", "hooks", "post-commit"),
		postCommitContent,
		postCommitMarker,
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

func uninstallHook(hookPath, content, marker string) error {
	existing, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	text := string(existing)
	if !strings.Contains(text, marker) {
		return nil
	}

	if strings.TrimSpace(text) == strings.TrimSpace(content) {
		return os.Remove(hookPath)
	}

	appendedContent := "\n\n# grít friction logger\n" + content
	if strings.Contains(text, appendedContent) {
		text = strings.Replace(text, appendedContent, "", 1)
	} else if strings.Contains(text, content) {
		text = strings.Replace(text, content, "", 1)
	}
	return os.WriteFile(hookPath, []byte(text), 0755)
}
