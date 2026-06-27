package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type releaseData struct {
	Version string
	Entries []entry
}

type prepareOptions struct {
	Version string
	DryRun  bool
}

func prepareRelease(opts prepareOptions) error {
	// Validate release ID: must not contain path separators or traversal.
	if strings.ContainsRune(opts.Version, '/') || opts.Version == ".." {
		return &exitError{ExitCheckFailed, fmt.Sprintf("invalid release id: %q", opts.Version)}
	}

	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
	}

	type entryFile struct {
		name string
		path string
		e    entry
	}

	var entryFiles []entryFile
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		p := filepath.Join(entriesDir(), f.Name())
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		e, parseErr := parseEntryJSON(data)
		if parseErr != nil {
			return parseErr
		}
		entryFiles = append(entryFiles, entryFile{f.Name(), p, e})
	}

	if len(entryFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No pending rellog entries found.\n\nAdd a changelog entry:\n  rellog add\n\nIf this release has no changelog-worthy changes, add an explicit empty entry:\n  rellog add-empty\n")
		return &exitError{ExitCheckFailed, ""}
	}

	// Validate entry URLs against github-url from config before doing anything.
	cfg, cfgErr := readEntryValidationConfig()
	if cfgErr != nil {
		return cfgErr
	}
	if cfg.githubURL != "" {
		var urlResults []fileCheckResult
		for _, ef := range entryFiles {
			var errs []checkError
			for _, issueURL := range ef.e.Issues {
				for _, msg := range validateStoredIssueURL(issueURL, cfg.githubURL) {
					errs = append(errs, checkError{"error[entry.issues.invalid]", msg + "."})
				}
			}
			for _, prURL := range ef.e.PRs {
				for _, msg := range validateStoredPRURL(prURL, cfg.githubURL) {
					errs = append(errs, checkError{"error[entry.prs.invalid]", msg + "."})
				}
			}
			if len(errs) > 0 {
				urlResults = append(urlResults, fileCheckResult{ef.path, errs})
			}
		}
		if len(urlResults) > 0 {
			return reportPrepareCheckFailure(urlResults)
		}
	}

	// Detect empty/normal conflict.
	var emptyPath string
	hasNormal := false
	for _, ef := range entryFiles {
		if ef.e.Kind == "empty" {
			emptyPath = ef.path
		} else {
			hasNormal = true
		}
	}
	if emptyPath != "" && hasNormal {
		return &exitError{ExitEntryConflict,
			fmt.Sprintf("entry conflict: empty entry %s cannot coexist with normal entries", emptyPath)}
	}

	isEmptyRelease := emptyPath != "" && !hasNormal

	// Collect normal entries for the release note.
	var entries []entry
	for _, ef := range entryFiles {
		if ef.e.Kind != "empty" {
			entries = append(entries, ef.e)
		}
	}

	releaseNotePath := filepath.Join(releaseNotesDir(), opts.Version+".md")
	changelogPath := "CHANGELOG.md"

	var content string
	if isEmptyRelease {
		content = fmt.Sprintf("## %s\n\nNo changelog-worthy changes.\n", opts.Version)
	} else {
		content = renderReleaseNote(opts.Version, entries)
	}

	// Check for existing release note before executing anything.
	if _, statErr := os.Stat(releaseNotePath); statErr == nil {
		return &exitError{ExitCheckFailed, "release-note file already exists: " + releaseNotePath}
	}

	if opts.DryRun {
		fmt.Print(content)
		fmt.Printf("create %s\n", releaseNotePath)
		fmt.Printf("update %s\n", changelogPath)
		for _, ef := range entryFiles {
			fmt.Printf("delete %s\n", ef.path)
		}
		return nil
	}

	// Write release note file.
	if err := os.WriteFile(releaseNotePath, []byte(content), 0644); err != nil {
		return err
	}

	// Update changelog (prepend, preserving any "# Changelog" header).
	existing, _ := os.ReadFile(changelogPath)
	newChangelog := mergeChangelog(content, string(existing))
	if err := os.WriteFile(changelogPath, []byte(newChangelog), 0644); err != nil {
		return err
	}

	// Delete entry files.
	for _, ef := range entryFiles {
		if err := os.Remove(ef.path); err != nil {
			return err
		}
	}

	fmt.Printf("%s release prepared\n", opts.Version)
	return nil
}

// mergeChangelog inserts newContent into existing changelog content.
// If existing starts with a "# Changelog" H1 heading, the new content is
// inserted after that heading so the heading remains at the top.
func mergeChangelog(newContent, existing string) string {
	if existing == "" {
		return newContent
	}
	const h1Header = "# Changelog\n"
	if strings.HasPrefix(existing, h1Header) {
		rest := strings.TrimPrefix(existing, h1Header)
		rest = strings.TrimPrefix(rest, "\n")
		return h1Header + "\n" + newContent + "\n" + rest
	}
	return newContent + "\n" + existing
}

func reportPrepareCheckFailure(results []fileCheckResult) error {
	totalErrs := 0
	for _, r := range results {
		totalErrs += len(r.Errors)
	}
	fmt.Fprintf(os.Stderr, "rellog check: FAILED\n\n%d files\n%d errors\n\n", len(results), totalErrs)
	printCheckDiagnostics(results)
	return &exitError{ExitCheckFailed, ""}
}
