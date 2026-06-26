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
		if !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		p := filepath.Join(entriesDir(), f.Name())
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		e, parseErr := parseEntry(data)
		if parseErr != nil {
			return parseErr
		}
		entryFiles = append(entryFiles, entryFile{f.Name(), p, e})
	}

	// Detect empty/normal conflict before doing anything.
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
