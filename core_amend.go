package rellog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type amendOptions struct {
	Version string
	DryRun  bool
}

func amendRelease(opts amendOptions) error {
	releaseID := opts.Version
	if err := validateReadyReleaseID(releaseID); err != nil {
		return err
	}

	if _, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			return &exitError{ExitNotInitialized, "run `rellog init` first"}
		}
		return err
	}

	paths, err := readReadyPaths()
	if err != nil {
		if os.IsNotExist(err) {
			return &exitError{ExitNotInitialized, "run `rellog init` first"}
		}
		return err
	}

	releaseNotePath := filepath.Join(paths.releaseNotesDir, releaseID+".md")
	releaseNoteData, err := os.ReadFile(releaseNotePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &exitError{ExitReleaseNotFound, releaseNotFoundMessage(releaseID, releaseNotePath)}
		}
		return err
	}
	releaseNoteContent := string(releaseNoteData)

	if err := checkMarkersBalanced(releaseNoteContent); err != nil {
		return &exitError{ExitCheckFailed, malformedStructureMessage(releaseNotePath, releaseID, err)}
	}

	changelogData, err := os.ReadFile(paths.changelogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &exitError{ExitCheckFailed, changelogMissingMessage(paths.changelogPath, releaseID)}
		}
		return err
	}
	changelogContent := string(changelogData)

	before, section, after, found, err := extractChangelogSection(changelogContent, releaseID)
	if err != nil {
		return &exitError{ExitCheckFailed, malformedStructureMessage(paths.changelogPath, releaseID, err)}
	}
	if !found {
		return &exitError{ExitCheckFailed, changelogMissingSectionMessage(paths.changelogPath, releaseID)}
	}

	entryFiles, err := loadEntryFiles(paths.entriesDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if len(entryFiles) == 0 {
		fmt.Printf("%s release unchanged\n", releaseID)
		return nil
	}

	// Detect empty/normal conflict within the pending entries themselves,
	// same rule prepare applies.
	var pendingEmptyPath string
	pendingHasNormal := false
	for _, ef := range entryFiles {
		if ef.e.Kind == "empty" {
			pendingEmptyPath = ef.path
		} else {
			pendingHasNormal = true
		}
	}
	if pendingEmptyPath != "" && pendingHasNormal {
		return &exitError{ExitEntryConflict, pendingInternalConflictMessage(pendingEmptyPath)}
	}

	cfg, err := readEntryValidationConfig()
	if err != nil {
		return err
	}

	consumedEntries, consumedState, consumedErr := loadConsumedCache(releaseID)
	if consumedState == consumedUnusable {
		fmt.Fprint(os.Stderr, consumedUnusableWarning(releaseID, consumedErr))
	}

	deletePaths := make([]string, 0, len(entryFiles))
	for _, ef := range entryFiles {
		deletePaths = append(deletePaths, ef.path)
	}

	if consumedState == consumedUsable {
		return amendRegenerate(regenerateInput{
			releaseID:          releaseID,
			releaseNotePath:    releaseNotePath,
			releaseNoteContent: releaseNoteContent,
			changelogPath:      paths.changelogPath,
			changelogBefore:    before,
			changelogSection:   section,
			changelogAfter:     after,
			consumedEntries:    consumedEntries,
			pendingEntries:     entryFiles,
			cfg:                cfg,
			dryRun:             opts.DryRun,
			deletePaths:        deletePaths,
		})
	}

	return amendAppend(appendInput{
		releaseID:          releaseID,
		releaseNotePath:    releaseNotePath,
		releaseNoteContent: releaseNoteContent,
		changelogPath:      paths.changelogPath,
		changelogBefore:    before,
		changelogSection:   section,
		changelogAfter:     after,
		pendingEntries:     entryFiles,
		cfg:                cfg,
		dryRun:             opts.DryRun,
		deletePaths:        deletePaths,
	})
}

type regenerateInput struct {
	releaseID          string
	releaseNotePath    string
	releaseNoteContent string
	changelogPath      string
	changelogBefore    string
	changelogSection   string
	changelogAfter     string
	consumedEntries    []entryFile
	pendingEntries     []entryFile
	cfg                entryValidationConfig
	dryRun             bool
	deletePaths        []string
}

func amendRegenerate(in regenerateInput) error {
	baselineIsEmpty := allKindEmpty(in.consumedEntries)

	if err := checkEmptyNormalMerge(in.releaseID, in.releaseNotePath, baselineIsEmpty, in.pendingEntries); err != nil {
		return err
	}

	var consumedOnly []entry
	for _, ef := range in.consumedEntries {
		consumedOnly = append(consumedOnly, ef.e)
	}
	consumedOnlyContent := renderAmendReleaseContent(in.releaseID, consumedOnly, in.cfg)

	if strings.TrimRight(consumedOnlyContent, "\n") != strings.TrimRight(in.releaseNoteContent, "\n") {
		return &exitError{ExitCheckFailed, regenerateMismatchMessage("release note", in.releaseNotePath, in.releaseID)}
	}
	if strings.TrimRight(consumedOnlyContent, "\n") != strings.TrimRight(in.changelogSection, "\n") {
		return &exitError{ExitCheckFailed, regenerateMismatchMessage("changelog release section", in.changelogPath, in.releaseID)}
	}

	combined := append([]entry{}, consumedOnly...)
	for _, ef := range in.pendingEntries {
		if ef.e.Kind != "empty" {
			combined = append(combined, ef.e)
		}
	}
	newContent := renderAmendReleaseContent(in.releaseID, combined, in.cfg)

	if in.dryRun {
		printAmendPreview(newContent, in.releaseNotePath, in.changelogPath, in.deletePaths, newContent != in.releaseNoteContent)
		return nil
	}

	allEntries := append(append([]entryFile{}, in.consumedEntries...), in.pendingEntries...)
	plan, warning, abort := planConsumedCacheUpdate(in.releaseID, allEntries)
	if abort != nil {
		return abort
	}

	if err := writeFileAtomic(in.releaseNotePath, []byte(newContent), 0644); err != nil {
		return err
	}
	newChangelog := spliceSection(in.changelogBefore, newContent, in.changelogAfter)
	if err := writeFileAtomic(in.changelogPath, []byte(newChangelog), 0644); err != nil {
		return err
	}

	commitWarning, commitAbort := commitConsumedCacheUpdate(in.releaseID, plan)
	if commitAbort != nil {
		return commitAbort
	}
	if warning == nil {
		warning = commitWarning
	}

	for _, p := range in.deletePaths {
		if err := os.Remove(p); err != nil {
			return err
		}
	}

	fmt.Printf("%s release amended\n", in.releaseID)
	if warning != nil {
		fmt.Fprintln(os.Stderr, warning)
	}
	return nil
}

type appendInput struct {
	releaseID          string
	releaseNotePath    string
	releaseNoteContent string
	changelogPath      string
	changelogBefore    string
	changelogSection   string
	changelogAfter     string
	pendingEntries     []entryFile
	cfg                entryValidationConfig
	dryRun             bool
	deletePaths        []string
}

func amendAppend(in appendInput) error {
	baselineIsEmpty := isEmptyReleaseContent(in.releaseNoteContent, in.releaseID)

	if err := checkEmptyNormalMerge(in.releaseID, in.releaseNotePath, baselineIsEmpty, in.pendingEntries); err != nil {
		return err
	}

	if baselineIsEmpty {
		// empty + empty merge: content is unchanged, only pending entries are consumed.
		if in.dryRun {
			printAmendPreview(in.releaseNoteContent, in.releaseNotePath, in.changelogPath, in.deletePaths, false)
			return nil
		}
		for _, p := range in.deletePaths {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
		fmt.Printf("%s release amended\n", in.releaseID)
		return nil
	}

	plan := buildKindInsertionPlan(in.pendingEntries, in.cfg)

	newReleaseNoteContent, err := applyKindInsertions(in.releaseNoteContent, plan)
	if err != nil {
		return &exitError{ExitCheckFailed, malformedStructureMessage(in.releaseNotePath, in.releaseID, err)}
	}
	newSectionContent, err := applyKindInsertions(in.changelogSection, plan)
	if err != nil {
		return &exitError{ExitCheckFailed, malformedStructureMessage(in.changelogPath, in.releaseID, err)}
	}

	newReleaseNoteContent = ensureTrailingNewline(newReleaseNoteContent)
	newSectionContent = ensureTrailingNewline(newSectionContent)

	if in.dryRun {
		printAmendPreview(newReleaseNoteContent, in.releaseNotePath, in.changelogPath, in.deletePaths, true)
		return nil
	}

	if err := writeFileAtomic(in.releaseNotePath, []byte(newReleaseNoteContent), 0644); err != nil {
		return err
	}
	newChangelog := spliceSection(in.changelogBefore, newSectionContent, in.changelogAfter)
	if err := writeFileAtomic(in.changelogPath, []byte(newChangelog), 0644); err != nil {
		return err
	}

	// Append mode never writes .rellog/consumed/: it has no reliable original
	// entry set to persist (that is precisely why it fell back to append mode).

	for _, p := range in.deletePaths {
		if err := os.Remove(p); err != nil {
			return err
		}
	}

	fmt.Printf("%s release amended\n", in.releaseID)
	return nil
}

// checkEmptyNormalMerge applies the amend empty/normal merge rules: a single
// release note must not mix empty and normal entries.
func checkEmptyNormalMerge(releaseID, releaseNotePath string, baselineIsEmpty bool, pendingEntries []entryFile) error {
	var pendingEmptyPath string
	var pendingNormalPaths []string
	for _, ef := range pendingEntries {
		if ef.e.Kind == "empty" {
			pendingEmptyPath = ef.path
		} else {
			pendingNormalPaths = append(pendingNormalPaths, ef.path)
		}
	}

	if baselineIsEmpty && len(pendingNormalPaths) > 0 {
		return &exitError{ExitEntryConflict, emptyBaselineConflictMessage(releaseID, releaseNotePath, pendingNormalPaths)}
	}
	if !baselineIsEmpty && pendingEmptyPath != "" {
		return &exitError{ExitEntryConflict, normalBaselineConflictMessage(releaseID, releaseNotePath, pendingEmptyPath)}
	}
	return nil
}

// releaseNotFoundMessage explains that the required release-note file is
// missing and how to create it.
func releaseNotFoundMessage(releaseID, releaseNotePath string) string {
	var sb strings.Builder
	sb.WriteString("release not found: ")
	sb.WriteString(releaseID)
	sb.WriteString("\n\n")
	sb.WriteString("No release-note file exists at:\n  ")
	sb.WriteString(releaseNotePath)
	sb.WriteString("\n\n")
	sb.WriteString("`rellog amend` only adds entries to a release that has already been prepared.\n")
	sb.WriteString("Prepare the release first:\n  rellog prepare ")
	sb.WriteString(releaseID)
	sb.WriteString(" --run")
	return sb.String()
}

// changelogMissingMessage explains that the configured changelog file does
// not exist and how to create it.
func changelogMissingMessage(changelogPath, releaseID string) string {
	var sb strings.Builder
	sb.WriteString("changelog does not exist: ")
	sb.WriteString(changelogPath)
	sb.WriteString("\n\n")
	sb.WriteString(changelogPath)
	sb.WriteString(" is required to amend a release, but it was not found.\n")
	sb.WriteString("If the release note was prepared normally, ")
	sb.WriteString(changelogPath)
	sb.WriteString(" should already exist alongside it.\n")
	sb.WriteString("Restore the file if it was deleted, or run `rellog prepare ")
	sb.WriteString(releaseID)
	sb.WriteString(" --run` again to recreate it.")
	return sb.String()
}

// changelogMissingSectionMessage explains that the changelog does not
// contain the expected release heading for releaseID, and how to fix it.
func changelogMissingSectionMessage(changelogPath, releaseID string) string {
	heading := markdownHeading(releaseHeadingLevel) + " " + releaseID
	var sb strings.Builder
	sb.WriteString("changelog is missing release section for ")
	sb.WriteString(releaseID)
	sb.WriteString(": ")
	sb.WriteString(changelogPath)
	sb.WriteString("\n\n")
	sb.WriteString("Expected to find this heading outside rellog body marker ranges:\n  ")
	sb.WriteString(heading)
	sb.WriteString("\n\n")
	sb.WriteString("but it was not found in ")
	sb.WriteString(changelogPath)
	sb.WriteString(".\n")
	sb.WriteString("If ")
	sb.WriteString(changelogPath)
	sb.WriteString(" was hand-edited, restore the release section for ")
	sb.WriteString(releaseID)
	sb.WriteString(".\n")
	sb.WriteString("Otherwise, run `rellog prepare ")
	sb.WriteString(releaseID)
	sb.WriteString(" --run` first.")
	return sb.String()
}

// malformedStructureMessage explains that a generated Markdown file has a
// malformed rellog body marker range and cannot be safely processed.
func malformedStructureMessage(path, releaseID string, cause error) string {
	var sb strings.Builder
	sb.WriteString("invalid generated Markdown structure in ")
	sb.WriteString(path)
	sb.WriteString(": ")
	sb.WriteString(cause.Error())
	sb.WriteString("\n\n")
	sb.WriteString("rellog cannot safely insert or verify entries while a rellog body marker pair (")
	sb.WriteString(bodyMarkerStart)
	sb.WriteString(" / ")
	sb.WriteString(bodyMarkerEnd)
	sb.WriteString(") is malformed.\n")
	sb.WriteString("Fix the marker pair in ")
	sb.WriteString(path)
	sb.WriteString(", then run `rellog amend ")
	sb.WriteString(releaseID)
	sb.WriteString("` again.")
	return sb.String()
}

// pendingInternalConflictMessage explains that the pending entries directory
// itself already contains both an empty entry and normal entries, and how to
// resolve it.
func pendingInternalConflictMessage(pendingEmptyPath string) string {
	var sb strings.Builder
	sb.WriteString("entry conflict: empty entry ")
	sb.WriteString(pendingEmptyPath)
	sb.WriteString(" cannot coexist with normal entries\n\n")
	sb.WriteString("Pending entries currently include both an empty entry and one or more normal entries.\n")
	sb.WriteString("A release note must not mix empty and normal entries.\n\n")
	sb.WriteString("If normal entries should be included in this release, remove the empty entry:\n  rm ")
	sb.WriteString(pendingEmptyPath)
	sb.WriteString("\n")
	sb.WriteString("Otherwise, if this release truly has no changelog-worthy changes, remove the normal entries instead.")
	return sb.String()
}

// emptyBaselineConflictMessage explains that pending normal entries cannot be
// merged into a release that was already prepared as empty, and how to
// proceed.
func emptyBaselineConflictMessage(releaseID, releaseNotePath string, pendingNormalPaths []string) string {
	var sb strings.Builder
	sb.WriteString("entry conflict: pending normal entries cannot be added to an empty release\n\n")
	sb.WriteString("release: ")
	sb.WriteString(releaseID)
	sb.WriteString("\n")
	sb.WriteString("release note:\n  ")
	sb.WriteString(releaseNotePath)
	sb.WriteString("\n\n")
	sb.WriteString("Pending normal entries:\n")
	for _, p := range pendingNormalPaths {
		sb.WriteString("  ")
		sb.WriteString(p)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(releaseID)
	sb.WriteString(" was already prepared as an empty release (\"")
	sb.WriteString(emptyReleaseMessage)
	sb.WriteString("\"). `rellog amend` does not convert an empty release into a normal one.\n\n")
	sb.WriteString("If these entries belong in ")
	sb.WriteString(releaseID)
	sb.WriteString(", edit ")
	sb.WriteString(releaseNotePath)
	sb.WriteString(" and the matching CHANGELOG.md section by hand to replace the empty-release template with normal content, then run `rellog amend ")
	sb.WriteString(releaseID)
	sb.WriteString("` again.\n")
	sb.WriteString("Otherwise, if these entries belong to a different or future release, remove them from the pending entries directory or move them there before running amend.")
	return sb.String()
}

// normalBaselineConflictMessage explains that a pending empty entry cannot be
// merged into a release that already has normal content, and how to proceed.
func normalBaselineConflictMessage(releaseID, releaseNotePath, pendingEmptyPath string) string {
	var sb strings.Builder
	sb.WriteString("entry conflict: pending empty entry cannot be added to a release with normal entries\n\n")
	sb.WriteString("release: ")
	sb.WriteString(releaseID)
	sb.WriteString("\n")
	sb.WriteString("release note:\n  ")
	sb.WriteString(releaseNotePath)
	sb.WriteString("\n")
	sb.WriteString("pending empty entry:\n  ")
	sb.WriteString(pendingEmptyPath)
	sb.WriteString("\n\n")
	sb.WriteString(releaseID)
	sb.WriteString(" already contains normal (non-empty) content, so an empty entry (created with `rellog add-empty`) cannot be merged into it.\n\n")
	sb.WriteString("If the empty entry was created by mistake, remove it:\n  rm ")
	sb.WriteString(pendingEmptyPath)
	sb.WriteString("\n")
	sb.WriteString("If it was meant for a different or future release, move it out of the pending entries directory before running amend again.")
	return sb.String()
}

// regenerateMismatchMessage explains that a release artifact no longer
// matches the consumed-only rendering, so regenerate mode cannot safely
// proceed, and how to recover.
func regenerateMismatchMessage(what, path, releaseID string) string {
	var sb strings.Builder
	sb.WriteString(what)
	sb.WriteString(" does not match the consumed cache; ")
	sb.WriteString(path)
	sb.WriteString(" was modified after `rellog prepare`\n\n")
	sb.WriteString("release: ")
	sb.WriteString(releaseID)
	sb.WriteString("\n")
	sb.WriteString(what)
	sb.WriteString(":\n  ")
	sb.WriteString(path)
	sb.WriteString("\n\n")
	sb.WriteString("rellog cannot safely regenerate ")
	sb.WriteString(path)
	sb.WriteString(" from `.rellog/consumed/")
	sb.WriteString(releaseID)
	sb.WriteString("/` because its content no longer matches what `rellog prepare` originally wrote.\n\n")
	sb.WriteString("If ")
	sb.WriteString(path)
	sb.WriteString(" was hand-edited intentionally, remove the consumed cache so `amend` falls back to appending onto the current content instead:\n  rm -r .rellog/consumed/")
	sb.WriteString(releaseID)
	sb.WriteString("\n")
	sb.WriteString("Then run `rellog amend ")
	sb.WriteString(releaseID)
	sb.WriteString("` again.\n")
	sb.WriteString("If the edit was unintentional, restore ")
	sb.WriteString(path)
	sb.WriteString(" to its prepared state instead.")
	return sb.String()
}

// consumedUnusableWarning explains, on stderr, why amend fell back to
// append mode and what the underlying problem was, terminated with a
// trailing newline so callers can fmt.Fprint it directly.
func consumedUnusableWarning(releaseID string, cause error) string {
	var sb strings.Builder
	sb.WriteString("warning: consumed cache for ")
	sb.WriteString(releaseID)
	sb.WriteString(" is unusable, falling back to append mode: ")
	sb.WriteString(cause.Error())
	sb.WriteString("\n")
	sb.WriteString("This does not block `amend`, but the release will not be able to use regenerate mode until the cache is fixed.\n")
	sb.WriteString("Investigate or remove the cache directory if it is no longer needed:\n  .rellog/consumed/")
	sb.WriteString(releaseID)
	sb.WriteString("\n")
	return sb.String()
}

func printAmendPreview(content, releaseNotePath, changelogPath string, deletePaths []string, contentChanges bool) {
	fmt.Print(content)
	if contentChanges {
		fmt.Printf("update %s\n", releaseNotePath)
		fmt.Printf("update %s\n", changelogPath)
	}
	for _, p := range deletePaths {
		fmt.Printf("delete %s\n", p)
	}
}

func spliceSection(before, section, after string) string {
	if after == "" {
		return before + section
	}
	return before + section + "\n" + after
}

func ensureTrailingNewline(content string) string {
	return strings.TrimRight(content, "\n") + "\n"
}

// renderAmendReleaseContent renders the full release-note content for entries,
// following the same empty/normal shape rules `prepare` uses: one or more
// "empty"-kind entries with no normal entries renders the fixed empty-release
// template; otherwise entries are rendered normally (empty-kind entries, if
// any, are never mixed in by the time this is called).
// allKindEmpty reports whether entryFiles is non-empty and every entry has
// kind "empty". A release stays "empty" for as long as every entry backing
// it is kind "empty", regardless of how many such entries have accumulated
// across repeated empty+empty amend merges — this must not be narrowed to a
// single-entry check, or a later normal entry could slip past the
// empty/normal conflict guard once the consumed cache holds more than one
// empty entry.
func allKindEmpty(entryFiles []entryFile) bool {
	if len(entryFiles) == 0 {
		return false
	}
	for _, ef := range entryFiles {
		if ef.e.Kind != "empty" {
			return false
		}
	}
	return true
}

func renderAmendReleaseContent(releaseID string, entries []entry, cfg entryValidationConfig) string {
	allEmpty := len(entries) > 0
	var normal []entry
	for _, e := range entries {
		if e.Kind == "empty" {
			continue
		}
		allEmpty = false
		normal = append(normal, e)
	}
	if allEmpty {
		return fmt.Sprintf("%s %s\n\n%s\n", markdownHeading(releaseHeadingLevel), releaseID, emptyReleaseMessage)
	}
	return renderReleaseNote(releaseID, normal, cfg)
}

// isEmptyReleaseContent reports whether content matches the fixed
// empty-release template for releaseID (newline-normalized).
func isEmptyReleaseContent(content, releaseID string) bool {
	expected := fmt.Sprintf("%s %s\n\n%s\n", markdownHeading(releaseHeadingLevel), releaseID, emptyReleaseMessage)
	return strings.TrimRight(content, "\n") == strings.TrimRight(expected, "\n")
}

type consumedState int

const (
	consumedAbsent consumedState = iota
	consumedUnusable
	consumedUsable
)

// loadConsumedCache reads and validates the consumed cache for releaseID.
// It reuses validateConsumedCacheDir (originally written to validate a
// freshly-built temp directory) against the already-committed cache
// directory on disk.
func loadConsumedCache(releaseID string) ([]entryFile, consumedState, error) {
	dir := consumedDir(releaseID)
	if _, statErr := os.Stat(dir); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, consumedAbsent, nil
		}
		return nil, consumedUnusable, statErr
	}

	if err := validateConsumedCacheDir(dir, releaseID); err != nil {
		return nil, consumedUnusable, err
	}

	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, consumedUnusable, err
	}
	var manifest consumedManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, consumedUnusable, err
	}

	entries := make([]entryFile, 0, len(manifest.Entries))
	for _, me := range manifest.Entries {
		p := filepath.Join(dir, "entries", me.Filename)
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, consumedUnusable, err
		}
		e, err := parseEntryJSON(data)
		if err != nil {
			return nil, consumedUnusable, err
		}
		entries = append(entries, entryFile{name: me.Filename, path: p, e: e})
	}
	return entries, consumedUsable, nil
}

// consumedCachePlan holds a successfully built (but not yet committed) temp
// consumed cache directory, to be committed only after the caller's release
// artifacts have been written.
type consumedCachePlan struct {
	tempDir string
	policy  string
}

// planConsumedCacheUpdate builds and validates a temp consumed cache for
// releaseID from entryFiles, mirroring prepareRelease's own build step so the
// same consume.on-fail-create semantics apply. Under the "error" policy,
// build (or destination preflight) failure aborts before any release
// artifact is written.
func planConsumedCacheUpdate(releaseID string, entryFiles []entryFile) (plan consumedCachePlan, warning error, abort error) {
	policy, err := readConsumeOnFailCreate()
	if err != nil {
		return consumedCachePlan{}, nil, err
	}

	builtDir, buildErr := buildConsumedCacheTemp(releaseID, entryFiles)
	if buildErr != nil {
		switch policy {
		case "error":
			return consumedCachePlan{}, nil, buildErr
		case "warn":
			return consumedCachePlan{}, buildErr, nil
		default: // ignore
			return consumedCachePlan{}, nil, nil
		}
	}

	if policy == "error" {
		// Unlike prepare (which only ever creates a fresh consumed cache for a
		// brand-new release id), amend's regenerate mode is always replacing the
		// cache for a release id it already confirmed has usable consumed data,
		// so a pre-existing finalDir is expected, not an obstacle. Only a parent
		// path component being a file (not a directory) is a genuine obstacle.
		finalDir := consumedDir(releaseID)
		if mkErr := os.MkdirAll(filepath.Dir(finalDir), 0755); mkErr != nil {
			_ = os.RemoveAll(builtDir)
			return consumedCachePlan{}, nil, mkErr
		}
	}

	return consumedCachePlan{tempDir: builtDir, policy: policy}, nil, nil
}

// commitConsumedCacheUpdate renames a successfully built temp consumed cache
// into place. Must only be called after the caller's release artifacts have
// been written successfully. If plan.tempDir is empty there is nothing to
// commit (build failed under warn/ignore).
func commitConsumedCacheUpdate(releaseID string, plan consumedCachePlan) (warning error, abort error) {
	if plan.tempDir == "" {
		return nil, nil
	}
	finalDir := consumedDir(releaseID)
	// Rename the old cache aside rather than deleting it outright, and restore
	// it on any failure below. By this point the release note and CHANGELOG.md
	// have already been overwritten with the merged content, so a commit
	// failure that instead left finalDir permanently absent would make the
	// next `amend` invocation fall back to append mode against content it has
	// no record of — silently reinserting entries that are already present.
	// Restoring the old (pre-merge) cache instead makes that next invocation
	// hit the regenerate-mode mismatch check and fail loudly.
	backupDir := ""
	commitErr := func() error {
		if _, statErr := os.Stat(finalDir); statErr == nil {
			backupDir = finalDir + ".amend-bak"
			_ = os.RemoveAll(backupDir) // clear any stale backup from a previous failed attempt
			if err := os.Rename(finalDir, backupDir); err != nil {
				backupDir = ""
				return err
			}
		}
		if err := os.MkdirAll(filepath.Dir(finalDir), 0755); err != nil {
			return err
		}
		return os.Rename(plan.tempDir, finalDir)
	}()
	if commitErr == nil {
		if backupDir != "" {
			_ = os.RemoveAll(backupDir)
		}
		return nil, nil
	}
	if backupDir != "" {
		_ = os.Rename(backupDir, finalDir)
	}
	_ = os.RemoveAll(plan.tempDir)
	switch plan.policy {
	case "error":
		return nil, commitErr
	case "warn":
		return commitErr, nil
	default: // ignore
		return nil, nil
	}
}

// checkMarkersBalanced verifies that rellog body marker comments are
// correctly paired within content.
func checkMarkersBalanced(content string) error {
	inBody := false
	for _, line := range strings.Split(content, "\n") {
		switch strings.TrimRight(line, "\r") {
		case bodyMarkerStart:
			if inBody {
				return errMarkerUnexpectedStart()
			}
			inBody = true
		case bodyMarkerEnd:
			if !inBody {
				return errMarkerUnexpectedEnd()
			}
			inBody = false
		}
	}
	if inBody {
		return errMarkerUnterminated()
	}
	return nil
}

// errMarkerUnexpectedStart, errMarkerUnexpectedEnd, and errMarkerUnterminated
// report the three ways a rellog body marker range can be malformed. They are
// shared by every function that scans for balanced bodyMarkerStart/End pairs
// (checkMarkersBalanced, extractChangelogSection, parseKindSections) so the
// wording stays consistent.
func errMarkerUnexpectedStart() error {
	return fmt.Errorf("unexpected %s before matching %s", bodyMarkerStart, bodyMarkerEnd)
}

func errMarkerUnexpectedEnd() error {
	return fmt.Errorf("unexpected %s without a preceding %s", bodyMarkerEnd, bodyMarkerStart)
}

func errMarkerUnterminated() error {
	return fmt.Errorf("unterminated %s", bodyMarkerStart)
}

// splitLinesWithOffsets splits content into lines the same way
// strings.Split(content, "\n") would, and additionally returns each line's
// starting byte offset within content. offsets[len(lines)] is set to
// len(content), a convenient sentinel for "one past the last line" so
// callers can compute a final section's end without a special case.
func splitLinesWithOffsets(content string) (lines []string, offsets []int) {
	lines = strings.Split(content, "\n")
	offsets = make([]int, len(lines)+1)
	pos := 0
	for i, l := range lines {
		offsets[i] = pos
		pos += len(l) + 1
	}
	offsets[len(lines)] = len(content)
	return lines, offsets
}

// extractChangelogSection locates the release section for releaseID within a
// CHANGELOG.md-shaped document: from its "## <releaseID>" heading (outside
// rellog body marker ranges) up to, but not including, the next top-level
// release heading outside body marker ranges, or end of file. It returns the
// text before the section, the section's own content (matching exactly what
// prepare/amend write for this release; the one blank-line separator before
// a following section is not included), and the remaining text starting at
// the next release heading (empty when this is the last section).
func extractChangelogSection(content, releaseID string) (before, section, after string, found bool, err error) {
	heading := markdownHeading(releaseHeadingLevel) + " " + releaseID
	headingPrefix := markdownHeading(releaseHeadingLevel) + " "

	lines, offsets := splitLinesWithOffsets(content)

	inBody := false
	startLine := -1
	endLine := -1
	for i, l := range lines {
		trimmed := strings.TrimRight(l, "\r")
		switch trimmed {
		case bodyMarkerStart:
			if inBody {
				return "", "", "", false, errMarkerUnexpectedStart()
			}
			inBody = true
			continue
		case bodyMarkerEnd:
			if !inBody {
				return "", "", "", false, errMarkerUnexpectedEnd()
			}
			inBody = false
			continue
		}
		if inBody {
			continue
		}
		if startLine == -1 {
			if trimmed == heading {
				startLine = i
			}
			continue
		}
		if strings.HasPrefix(trimmed, headingPrefix) {
			endLine = i
			break
		}
	}
	if inBody {
		return "", "", "", false, errMarkerUnterminated()
	}
	if startLine == -1 {
		return "", "", "", false, nil
	}
	if endLine == -1 {
		endLine = len(lines)
	}

	startOffset := offsets[startLine]
	endOffset := offsets[endLine]

	before = content[:startOffset]
	raw := content[startOffset:endOffset]
	after = content[endOffset:]
	if endLine < len(lines) {
		raw = strings.TrimSuffix(raw, "\n")
	}
	return before, raw, after, true, nil
}

// kindSection is one "### <title>" section parsed out of a single release
// section's content (as extracted by extractChangelogSection, or a
// release-note file's full content, which is itself exactly one section).
type kindSection struct {
	title      string
	start, end int
}

// parseKindSections finds every level-3 kind-title heading outside rellog
// body marker ranges within content, returning each section's content extent
// (from right after its heading line to the next kind heading or EOF).
func parseKindSections(content string) ([]kindSection, error) {
	kindPrefix := markdownHeading(sectionHeadingLevel) + " "

	lines, offsets := splitLinesWithOffsets(content)

	inBody := false
	var headingLines []int
	var titles []string
	for i, l := range lines {
		trimmed := strings.TrimRight(l, "\r")
		switch trimmed {
		case bodyMarkerStart:
			if inBody {
				return nil, errMarkerUnexpectedStart()
			}
			inBody = true
			continue
		case bodyMarkerEnd:
			if !inBody {
				return nil, errMarkerUnexpectedEnd()
			}
			inBody = false
			continue
		}
		if inBody {
			continue
		}
		if strings.HasPrefix(trimmed, kindPrefix) {
			headingLines = append(headingLines, i)
			titles = append(titles, strings.TrimPrefix(trimmed, kindPrefix))
		}
	}
	if inBody {
		return nil, errMarkerUnterminated()
	}

	sections := make([]kindSection, 0, len(headingLines))
	for idx, lineIdx := range headingLines {
		start := offsets[lineIdx+1]
		end := len(content)
		if idx+1 < len(headingLines) {
			nextHeadingLine := headingLines[idx+1]
			end = offsets[nextHeadingLine]
			// renderKindSection always emits exactly one blank-line separator
			// immediately before a "### " heading (its own leading "\n"). That
			// line belongs to the next heading, not this section's trailing
			// content, so exclude it here — otherwise an insertion at `end`
			// would land after the separator, gluing this section's last entry
			// directly onto the next heading with no blank line between them.
			if nextHeadingLine > 0 && strings.TrimRight(lines[nextHeadingLine-1], "\r") == "" {
				end = offsets[nextHeadingLine-1]
			}
		}
		sections = append(sections, kindSection{title: titles[idx], start: start, end: end})
	}
	return sections, nil
}

// kindInsertion is one entry of an amend append-mode insertion plan: the
// entries destined for the kind section identified by title, in filename
// order.
type kindInsertion struct {
	title   string
	entries []entry
}

// buildKindInsertionPlan groups pendingEntries (already in filename order) by
// their effective kind title, preserving first-seen kind order and per-kind
// filename order.
func buildKindInsertionPlan(pendingEntries []entryFile, cfg entryValidationConfig) []kindInsertion {
	var order []string
	byTitle := map[string][]entry{}
	for _, ef := range pendingEntries {
		if ef.e.Kind == "empty" {
			continue
		}
		title := kindTitle(ef.e.Kind, cfg)
		if _, seen := byTitle[title]; !seen {
			order = append(order, title)
		}
		byTitle[title] = append(byTitle[title], ef.e)
	}
	plan := make([]kindInsertion, 0, len(order))
	for _, title := range order {
		plan = append(plan, kindInsertion{title: title, entries: byTitle[title]})
	}
	return plan
}

// applyKindInsertions splices plan into content: entries destined for a kind
// title that already has a section are inserted at the end of that section's
// content, replicating the same separator renderReleaseNote already produces
// for multiple same-kind entries; entries destined for a title with no
// existing section are appended as brand-new kind sections at the end of
// content, in first-seen order.
func applyKindInsertions(content string, plan []kindInsertion) (string, error) {
	sections, err := parseKindSections(content)
	if err != nil {
		return "", err
	}
	sectionByTitle := map[string]kindSection{}
	for _, s := range sections {
		if _, exists := sectionByTitle[s.title]; !exists {
			sectionByTitle[s.title] = s
		}
	}

	type existingInsert struct {
		pos  int
		text string
	}
	var existingInserts []existingInsert
	var newSections strings.Builder

	for _, ins := range plan {
		if sec, ok := sectionByTitle[ins.title]; ok {
			var sb strings.Builder
			for _, e := range ins.entries {
				sb.WriteString("\n")
				renderEntryBlock(&sb, e)
			}
			existingInserts = append(existingInserts, existingInsert{pos: sec.end, text: sb.String()})
		} else {
			newSections.WriteString(renderKindSection(ins.title, ins.entries))
		}
	}

	sort.Slice(existingInserts, func(i, j int) bool { return existingInserts[i].pos < existingInserts[j].pos })

	var result strings.Builder
	last := 0
	for _, ins := range existingInserts {
		result.WriteString(content[last:ins.pos])
		result.WriteString(ins.text)
		last = ins.pos
	}
	result.WriteString(content[last:])
	result.WriteString(newSections.String())

	return result.String(), nil
}
