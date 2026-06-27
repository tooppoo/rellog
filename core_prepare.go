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

	// Collect normal entries for the release note.
	var entries []entry
	for _, ef := range entryFiles {
		if ef.e.Kind != "empty" {
			entries = append(entries, ef.e)
		}
	}

	releaseNotePath := filepath.Join(releaseNotesDir(), opts.Version+".md")
	changelogPath := "CHANGELOG.md"
	content := renderReleaseNote(opts.Version, entries)

	if opts.DryRun {
		fmt.Print(content)
		fmt.Printf("create %s\n", releaseNotePath)
		fmt.Printf("append %s\n", changelogPath)
		for _, ef := range entryFiles {
			fmt.Printf("delete %s\n", ef.path)
		}
		return nil
	}

	// Write release note file.
	if err := os.WriteFile(releaseNotePath, []byte(content), 0644); err != nil {
		return err
	}

	// Prepend to changelog (or create it).
	existing, _ := os.ReadFile(changelogPath)
	var newContent string
	if len(existing) > 0 {
		newContent = content + "\n" + string(existing)
	} else {
		newContent = content
	}
	if err := os.WriteFile(changelogPath, []byte(newContent), 0644); err != nil {
		return err
	}

	// Delete entry files.
	for _, ef := range entryFiles {
		if err := os.Remove(ef.path); err != nil {
			return err
		}
	}
	return nil
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
