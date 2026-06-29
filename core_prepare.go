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

	// Read consume.on-fail-create policy.
	consumePolicy, err := readConsumeOnFailCreate()
	if err != nil {
		return err
	}

	// Build the consumed cache in a temp directory under .rellog/consumed/ so that:
	//   - a partial build never lands at the final path
	//   - the cache is committed (renamed) only after all required artifact writes succeed
	// tempCacheDir is non-empty only when a successfully-built temp dir is pending commit;
	// the defer guarantees cleanup on any early return.
	var tempCacheDir string
	defer func() {
		if tempCacheDir != "" {
			_ = os.RemoveAll(tempCacheDir)
		}
	}()

	var consumeErr error
	if builtDir, buildErr := buildConsumedCacheTemp(opts.Version, entryFiles); buildErr != nil {
		switch consumePolicy {
		case "error":
			return buildErr
		case "warn":
			consumeErr = buildErr
		case "ignore":
			// suppress
		}
	} else {
		// Build succeeded.  For "error" policy, preflight the final path now so we can
		// fail cleanly before writing any release artifacts.  For warn/ignore the stale
		// dir is removed at commit time (after artifacts) so that a failed artifact write
		// cannot leave us with no consumed cache at all.
		finalDir := consumedDir(opts.Version)
		if consumePolicy == "error" {
			// Preflight: detect any filesystem obstacle before writing release artifacts.
			// (1) finalDir itself already exists.
			if _, statErr := os.Stat(finalDir); statErr == nil {
				_ = os.RemoveAll(builtDir)
				return fmt.Errorf("%s already exists", finalDir)
			}
			// (2) A path component of the parent is a file, not a directory (e.g.
			//     release id "cli/v1.0.0" where .rellog/consumed/cli is a file).
			//     Attempt to create the parent now so that the failure is caught here
			//     rather than in the commit closure after artifacts are written.
			if mkErr := os.MkdirAll(filepath.Dir(finalDir), 0755); mkErr != nil {
				_ = os.RemoveAll(builtDir)
				return mkErr
			}
		}
		tempCacheDir = builtDir
	}

	// Write release note atomically.
	if err := writeFileAtomic(releaseNotePath, []byte(content), 0644); err != nil {
		return err
	}

	// Update changelog atomically (prepend, preserving any "# Changelog" header).
	existing, _ := os.ReadFile(changelogPath)
	newChangelog := mergeChangelog(content, string(existing))
	if err := writeFileAtomic(changelogPath, []byte(newChangelog), 0644); err != nil {
		return err
	}

	// Commit consumed cache: rename temp dir to its final path only after required
	// artifact writes have succeeded. Commit failure is also subject to on-fail-create
	// so that warn/ignore callers still get a successful prepare.
	if tempCacheDir != "" {
		finalDir := consumedDir(opts.Version)
		commitErr := func() error {
			// Remove a stale consumed cache dir if one exists.  This is deferred to
			// commit time so that an artifact write failure never leaves us without any
			// consumed cache (the old one is still intact until this point).
			if _, statErr := os.Stat(finalDir); statErr == nil {
				if removeErr := os.RemoveAll(finalDir); removeErr != nil {
					return removeErr
				}
			}
			// Release IDs may contain path separators (e.g. "cli/v1.0.0"), so
			// the parent of finalDir might not exist yet.
			if err := os.MkdirAll(filepath.Dir(finalDir), 0755); err != nil {
				return err
			}
			return os.Rename(tempCacheDir, finalDir)
		}()
		if commitErr != nil {
			switch consumePolicy {
			case "error":
				return commitErr
			case "warn":
				if consumeErr == nil {
					consumeErr = commitErr
				}
			case "ignore":
				// suppress
			}
			// tempCacheDir is still set so the defer removes the uncommitted temp dir.
		} else {
			tempCacheDir = "" // committed — prevent defer from removing the final dir
		}
	}

	// Delete entry files only after required release artifact writes have succeeded.
	for _, ef := range entryFiles {
		if err := os.Remove(ef.path); err != nil {
			return err
		}
	}

	fmt.Printf("%s release prepared\n", opts.Version)
	if consumeErr != nil {
		fmt.Fprintln(os.Stderr, consumeErr)
	}
	return nil
}

// buildConsumedCacheTemp builds the consumed cache for releaseID into a temporary
// directory under .rellog/consumed/ and validates its internal consistency (manifest
// schema decoded from disk, cross-file missing/orphan/duplicate checks, entry JSON
// schema for each copied file). On success it returns the temp dir path; the caller
// must either rename it to the final location (commit) or let the deferred RemoveAll
// discard it. On failure the temp dir is removed internally before returning.
func buildConsumedCacheTemp(releaseID string, entryFiles []entryFile) (string, error) {
	consumedBase := filepath.Join(baseDir, "consumed")
	if err := os.MkdirAll(consumedBase, 0755); err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp(consumedBase, ".rellog-tmp-*")
	if err != nil {
		return "", err
	}

	entriesSubDir := filepath.Join(tempDir, "entries")
	if err := os.Mkdir(entriesSubDir, 0755); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	manifest := consumedManifest{
		SchemaVersion: 1,
		ReleaseID:     releaseID,
		Entries:       make([]consumedManifestEntry, 0, len(entryFiles)),
	}

	for _, ef := range entryFiles {
		data, err := os.ReadFile(ef.path)
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return "", err
		}
		if err := os.WriteFile(filepath.Join(entriesSubDir, ef.name), data, 0644); err != nil {
			_ = os.RemoveAll(tempDir)
			return "", err
		}
		manifest.Entries = append(manifest.Entries, consumedManifestEntry{Filename: ef.name})
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}
	manifestData = append(manifestData, '\n')

	if err := os.WriteFile(filepath.Join(tempDir, "manifest.json"), manifestData, 0644); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	if err := validateConsumedCacheDir(tempDir, releaseID); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	return tempDir, nil
}

// validateConsumedCacheDir validates the consumed cache in dir by decoding
// manifest.json from disk (not from the in-memory struct) and checking:
//   - manifest schema version and release ID
//   - no duplicate filenames in the manifest
//   - every manifest entry has a corresponding file in entries/ (missing check)
//   - every file in entries/ is listed in the manifest (orphan check)
//   - every entry file parses successfully as valid entry JSON
func validateConsumedCacheDir(dir, releaseID string) error {
	// Decode manifest.json from disk to catch any serialisation issues.
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return fmt.Errorf("consumed cache: cannot read manifest: %w", err)
	}
	var manifest consumedManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("consumed cache: invalid manifest JSON: %w", err)
	}

	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("consumed cache: unexpected schema version %d", manifest.SchemaVersion)
	}
	if manifest.ReleaseID != releaseID {
		return fmt.Errorf("consumed cache: release ID mismatch: got %q, want %q", manifest.ReleaseID, releaseID)
	}

	// Build manifest filename set; detect duplicates.
	manifestSet := make(map[string]bool, len(manifest.Entries))
	for _, e := range manifest.Entries {
		if manifestSet[e.Filename] {
			return fmt.Errorf("consumed cache: duplicate filename in manifest: %q", e.Filename)
		}
		manifestSet[e.Filename] = true
	}

	// Build on-disk filename set.
	dirEntries, err := os.ReadDir(filepath.Join(dir, "entries"))
	if err != nil {
		return fmt.Errorf("consumed cache: cannot read entries dir: %w", err)
	}
	filesSet := make(map[string]bool, len(dirEntries))
	for _, de := range dirEntries {
		filesSet[de.Name()] = true
	}

	// Missing: listed in manifest but no file on disk.
	for name := range manifestSet {
		if !filesSet[name] {
			return fmt.Errorf("consumed cache: manifest entry missing file: %q", name)
		}
	}
	// Orphan: file on disk but not listed in manifest.
	for name := range filesSet {
		if !manifestSet[name] {
			return fmt.Errorf("consumed cache: orphan entry file not in manifest: %q", name)
		}
	}

	// Validate each entry file copied into the temp dir using the full structural
	// schema check (not just parseEntryJSON, which only catches invalid JSON and
	// records field-level issues as diagnostics without returning errors).
	for name := range manifestSet {
		entryData, err := os.ReadFile(filepath.Join(dir, "entries", name))
		if err != nil {
			return fmt.Errorf("consumed cache: cannot read entry %q: %w", name, err)
		}
		e, parseErr := parseEntryJSON(entryData)
		if parseErr != nil {
			return fmt.Errorf("consumed cache: invalid entry JSON %q: %w", name, parseErr)
		}
		if schemaErrs := validateEntrySchema(e); len(schemaErrs) > 0 {
			return fmt.Errorf("consumed cache: entry %q schema error: %s", name, schemaErrs[0].Message)
		}
	}

	return nil
}

// writeFileAtomic writes data to dst via a temp file + rename so that no
// partial content is ever visible at the destination path.
func writeFileAtomic(dst string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".rellog-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return err
	}
	ok = true
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
