package cli

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/levifig/loaf/internal/project"
)

// checkoutToplevel returns the physical toplevel of the Git worktree that
// contains dir.
func checkoutToplevel(dir string) (string, bool) {
	toplevel, err := commandOutput(dir, "git", "rev-parse", "--show-toplevel")
	toplevel = strings.TrimSpace(toplevel)
	if err != nil || toplevel == "" {
		return "", false
	}
	if real, err := filepath.EvalSymlinks(toplevel); err == nil {
		toplevel = real
	}
	return toplevel, true
}

// prTargetCheckout returns the first of the command's and the session's
// checkouts that tracks the gh pr create --repo target as a remote.
func prTargetCheckout(target, root, processRoot string) (string, bool) {
	host, path := githubTargetKey(target)
	if path == "" {
		return root, false
	}
	for _, candidate := range []string{root, processRoot} {
		if candidate != "" && checkoutTracksRepository(candidate, host, path) {
			return candidate, true
		}
	}
	return root, false
}

// githubTargetKey splits gh's [HOST/]OWNER/REPO or a repository URL into a
// host and a lowercase /owner/repo path. A bare OWNER/REPO carries no host:
// gh resolves it against configuration the hook does not read, so it matches
// that repository on any host.
func githubTargetKey(target string) (host, path string) {
	if strings.Contains(target, "://") || strings.HasPrefix(target, "git@") {
		return githubRemoteKey(target)
	}
	parts := strings.Split(strings.TrimSuffix(target, ".git"), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", ""
		}
	}
	switch len(parts) {
	case 2:
		return "", strings.ToLower("/" + parts[0] + "/" + parts[1])
	case 3:
		return githubCanonicalHost(parts[0]), strings.ToLower("/" + parts[1] + "/" + parts[2])
	}
	return "", ""
}

// githubRemoteKey normalizes a hosted remote URL like githubTargetKey. Only
// scheme URLs and scp-style host:path remotes name a host; filesystem paths
// and file:// remotes never match a hosted target.
func githubRemoteKey(remote string) (host, path string) {
	scheme, _, hasScheme := strings.Cut(remote, "://")
	switch {
	case hasScheme:
		if strings.EqualFold(scheme, "file") {
			return "", ""
		}
	case !scpLikeRemoteRE.MatchString(remote):
		return "", ""
	}
	key, err := project.NormalizeRemoteURL(remote)
	if err != nil || key == "" {
		return "", ""
	}
	host, rest, found := strings.Cut(key, "/")
	if !found {
		return "", ""
	}
	return githubCanonicalHost(host), strings.ToLower("/" + rest)
}

// GitHub serves SSH on port 443 as ssh.github.com; gh treats it as github.com.
func githubCanonicalHost(host string) string {
	host = strings.ToLower(host)
	if host == "ssh.github.com" {
		return "github.com"
	}
	return host
}

// Git reads host:path as scp-style only when the colon precedes any slash.
var scpLikeRemoteRE = regexp.MustCompile(`^([^@/:]+@)?[^/:]+:`)

func checkoutTracksRepository(root, host, path string) bool {
	output, err := commandOutput(root, "git", "remote", "-v")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		remoteHost, remotePath := githubRemoteKey(fields[1])
		if remotePath == path && (host == "" || remoteHost == host) {
			return true
		}
	}
	return false
}
