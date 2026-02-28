package project

import (
	"os/exec"
	"strings"
)

// GitRemoteURL returns the origin remote URL for the git repo at dir,
// or empty string if not a git repo or no origin remote.
func GitRemoteURL(dir string) string {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RepoName extracts the repository name from a git remote URL.
// Handles SSH SCP-style (git@host:org/repo), HTTPS, and ssh:// URLs.
// Strips any .git suffix. Returns empty string for unparseable input.
func RepoName(remoteURL string) string {
	if remoteURL == "" {
		return ""
	}

	var path string

	// SSH SCP-style: git@github.com:org/repo.git
	if host, after, ok := strings.Cut(remoteURL, ":"); ok && !strings.Contains(host, "/") && !strings.Contains(remoteURL, "://") {
		path = after
	} else {
		// HTTPS or ssh:// URL — strip scheme and host
		trimmed := remoteURL
		if _, after, ok := strings.Cut(trimmed, "://"); ok {
			trimmed = after
		}
		// Strip host (everything up to first /)
		if _, after, ok := strings.Cut(trimmed, "/"); ok {
			path = after
		}
	}

	if path == "" {
		return ""
	}

	// Strip .git suffix
	path = strings.TrimSuffix(path, ".git")

	return path
}
