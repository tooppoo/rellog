package rellog

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// isValidGitHubRepoURL reports whether u is a canonical GitHub repository URL
// of the form https://github.com/owner/repo (no trailing slash, no query, no fragment).
func isValidGitHubRepoURL(u string) bool {
	if !strings.HasPrefix(u, "https://github.com/") {
		return false
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if strings.HasSuffix(u, "/") {
		return false
	}
	if strings.HasSuffix(u, ".git") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// detectGitHubURL detects the canonical GitHub repository URL from the git origin remote.
// It first tries os.Getwd(), then falls back to the PWD environment variable.
// The PWD fallback handles environments (such as the testscript test framework) where
// os.Chdir is called without updating PWD.
// Returns an error if neither directory is inside a git repository.
func detectGitHubURL() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if u, err := gitRemoteOriginURL(cwd); err == nil {
			return u, nil
		}
	}
	if pwd := os.Getenv("PWD"); pwd != "" {
		if u, err := gitRemoteOriginURL(pwd); err == nil {
			return u, nil
		}
	}
	return "", fmt.Errorf("not a git repository")
}

// gitRemoteOriginURL returns the canonical GitHub HTTPS URL for the origin
// remote of the git repo at dir.
// Returns ("", nil) if dir is a git repository but has no GitHub origin remote.
// Returns ("", error) if dir is not a git repository.
func gitRemoteOriginURL(dir string) (string, error) {
	if err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run(); err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", nil
	}
	raw := strings.TrimSpace(string(out))
	normalized, err := normalizeGitRemoteToHTTPS(raw)
	if err != nil || !isValidGitHubRepoURL(normalized) {
		return "", nil
	}
	return normalized, nil
}

// normalizeGitRemoteToHTTPS converts SSH or HTTPS git remote URLs to canonical
// HTTPS form with no .git suffix.
func normalizeGitRemoteToHTTPS(raw string) (string, error) {
	if after, ok := strings.CutPrefix(raw, "git@github.com:"); ok {
		path := strings.TrimSuffix(after, ".git")
		return "https://github.com/" + path, nil
	}
	if strings.HasPrefix(raw, "https://github.com/") {
		return strings.TrimSuffix(raw, ".git"), nil
	}
	return "", fmt.Errorf("unrecognized remote URL: %s", raw)
}

// isPositiveInteger reports whether s represents a decimal integer > 0.
func isPositiveInteger(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0
}

// validateAndNormalizeIssueRef validates or normalizes a single issue reference.
// Positive integer strings are normalized to full URLs; other strings are validated as URLs.
func validateAndNormalizeIssueRef(ref, githubURL string) (string, []string) {
	if isPositiveInteger(ref) {
		return githubURL + "/issues/" + ref, nil
	}
	errs := validateGitHubRef(ref, "issue", githubURL)
	if len(errs) == 0 {
		return ref, nil
	}
	return "", errs
}

// validateAndNormalizePRRef validates or normalizes a single PR reference.
// Positive integer strings are normalized to full URLs; other strings are validated as URLs.
func validateAndNormalizePRRef(ref, githubURL string) (string, []string) {
	if isPositiveInteger(ref) {
		return githubURL + "/pull/" + ref, nil
	}
	errs := validateGitHubRef(ref, "pr", githubURL)
	if len(errs) == 0 {
		return ref, nil
	}
	return "", errs
}

// validateGitHubRef validates a GitHub issue or PR URL.
// kind must be "issue" or "pr".
func validateGitHubRef(rawURL, kind, githubURL string) []string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return []string{fmt.Sprintf("%s URL is not a valid URL", kindLabel(kind))}
	}

	if u.Host != "github.com" {
		return []string{fmt.Sprintf("%s URL host must be github.com", kindLabel(kind))}
	}

	if u.RawQuery != "" || u.Fragment != "" {
		return []string{fmt.Sprintf("%s URL must be canonical and must not include query or fragment", kindLabel(kind))}
	}

	if strings.HasSuffix(rawURL, "/") {
		return []string{fmt.Sprintf("%s URL must be canonical and must not have a trailing slash", kindLabel(kind))}
	}

	var errs []string

	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 5)
	var urlType string
	if len(parts) >= 3 {
		urlType = parts[2]
	}

	expectedPathType := "issues"
	if kind == "pr" {
		expectedPathType = "pull"
	}

	if urlType != expectedPathType {
		switch kind {
		case "issue":
			if urlType == "pull" {
				errs = append(errs, fmt.Sprintf("pull request URL was passed to --issue; use --pr for %s", rawURL))
			}
		case "pr":
			if urlType == "issues" {
				errs = append(errs, fmt.Sprintf("issue URL was passed to --pr; use --issue for %s", rawURL))
			}
		}
	}

	if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
		repoURL := "https://github.com/" + parts[0] + "/" + parts[1]
		if repoURL != githubURL {
			label := urlTypeLabel(urlType, kind)
			errs = append(errs, fmt.Sprintf(`%s URL repository "%s" does not match github-url "%s"`, label, repoURL, githubURL))
		}
	}

	return errs
}

// validateStoredIssueURL validates an issue URL already stored in an entry.
// Returns error messages if the URL is invalid.
func validateStoredIssueURL(rawURL, githubURL string) []string {
	return validateStoredRef(rawURL, "issue", githubURL)
}

// validateStoredPRURL validates a PR URL already stored in an entry.
// Returns error messages if the URL is invalid.
func validateStoredPRURL(rawURL, githubURL string) []string {
	return validateStoredRef(rawURL, "pr", githubURL)
}

func validateStoredRef(rawURL, kind, githubURL string) []string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return []string{fmt.Sprintf("%s URL is not a valid URL", kindLabel(kind))}
	}

	if u.Host != "github.com" {
		return []string{fmt.Sprintf("%s URL host must be github.com", kindLabel(kind))}
	}

	if u.RawQuery != "" || u.Fragment != "" {
		return []string{fmt.Sprintf("%s URL must be canonical and must not include query or fragment", kindLabel(kind))}
	}

	if strings.HasSuffix(rawURL, "/") {
		return []string{fmt.Sprintf("%s URL must be canonical and must not have a trailing slash", kindLabel(kind))}
	}

	var errs []string

	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 5)
	var urlType string
	if len(parts) >= 3 {
		urlType = parts[2]
	}

	expectedPathType := "issues"
	if kind == "pr" {
		expectedPathType = "pull"
	}

	if urlType != expectedPathType {
		switch kind {
		case "issue":
			if urlType == "pull" {
				errs = append(errs, fmt.Sprintf("pull request URL was used in issues; use prs for %s", rawURL))
			}
		case "pr":
			if urlType == "issues" {
				errs = append(errs, fmt.Sprintf("issue URL was used in prs; use issues for %s", rawURL))
			}
		}
	}

	if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
		repoURL := "https://github.com/" + parts[0] + "/" + parts[1]
		if repoURL != githubURL {
			label := urlTypeLabel(urlType, kind)
			errs = append(errs, fmt.Sprintf(`%s URL repository "%s" does not match github-url "%s"`, label, repoURL, githubURL))
		}
	}

	if len(errs) == 0 && len(parts) >= 4 {
		num := parts[3]
		if !isPositiveInteger(num) {
			errs = append(errs, fmt.Sprintf("%s URL number must be a positive integer", kindLabel(kind)))
		}
	}

	return errs
}

func kindLabel(kind string) string {
	if kind == "pr" {
		return "pull request"
	}
	return kind
}

func urlTypeLabel(urlType, kind string) string {
	switch urlType {
	case "pull":
		return "pull request"
	case "issues":
		return "issue"
	default:
		return kindLabel(kind)
	}
}
