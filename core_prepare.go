package rellog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type prepareOptions struct {
	Version string
	DryRun  bool
}

type entryFile struct {
	name string
	path string
	e    entry
}

func prepareRelease(opts prepareOptions) error {
	if err := validateReadyReleaseID(opts.Version); err != nil {
		return &exitError{ExitCheckFailed, fmt.Sprintf("invalid release id: %q", opts.Version)}
	}

	checkResults, totalEntries, err := checkRepository()
	if err != nil {
		return err
	}
	if len(checkResults) > 0 {
		return reportPrepareCheckFailure(checkResults)
	}
	if totalEntries == 0 {
		fmt.Fprintf(os.Stderr, "No pending rellog entries found.\n\nAdd a changelog entry:\n  rellog add\n\nIf this release has no changelog-worthy changes, add an explicit empty entry:\n  rellog add-empty\n")
		return &exitError{ExitCheckFailed, ""}
	}

	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
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
		content = fmt.Sprintf("%s %s\n\n%s\n", markdownHeading(releaseHeadingLevel), opts.Version, emptyReleaseMessage)
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

	// Save consumed cache before deleting entries.
	if err := writeConsumedCache(opts.Version, entryFiles); err != nil {
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

type consumedManifest struct {
	SchemaVersion int                     `json:"schemaVersion"`
	ReleaseID     string                  `json:"releaseId"`
	Entries       []consumedManifestEntry `json:"entries"`
}

type consumedManifestEntry struct {
	Filename string `json:"filename"`
}

func writeConsumedCache(releaseID string, entryFiles []entryFile) error {
	entriesDir := consumedEntriesDir(releaseID)
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		return err
	}

	manifest := consumedManifest{
		SchemaVersion: 1,
		ReleaseID:     releaseID,
		Entries:       make([]consumedManifestEntry, 0, len(entryFiles)),
	}
	for _, ef := range entryFiles {
		data, err := os.ReadFile(ef.path)
		if err != nil {
			return err
		}
		dest := filepath.Join(entriesDir, ef.name)
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		manifest.Entries = append(manifest.Entries, consumedManifestEntry{Filename: ef.name})
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestData = append(manifestData, '\n')
	return os.WriteFile(consumedManifestPath(releaseID), manifestData, 0644)
}

// mergeChangelog inserts newContent into existing changelog content.
// If existing starts with a "# Changelog" H1 heading, the new content is
// inserted after that heading so the heading remains at the top.
func mergeChangelog(newContent, existing string) string {
	if existing == "" {
		return "# CHANGELOG\n\n" + newContent
	}
	const h1Header = "# CHANGELOG\n"
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
