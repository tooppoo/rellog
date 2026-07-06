package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type addOptions struct {
	Kind          string
	Targets       []string
	Body          string
	Links         []string
	DebugDatetime string
}

// checkEntryStorePreconditions verifies the entry store is usable: rellog is
// initialized and the entries path is a directory. The entries directory is
// not tracked by git when empty, so a fresh checkout may be missing it even
// though rellog was initialized; in that case it is created transparently.
func checkEntryStorePreconditions() error {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return &exitError{ExitNotInitialized, "run `rellog init` first"}
	}
	info, err := os.Stat(entriesDir())
	switch {
	case os.IsNotExist(err):
		return os.MkdirAll(entriesDir(), 0755)
	case err != nil:
		return err
	case !info.IsDir():
		return &exitError{ExitInvalidStructure, entriesDir() + " is not a directory"}
	}
	return nil
}

// checkNoEmptyEntryConflict fails when an empty entry already exists, in
// which case a normal entry can never be added.
func checkNoEmptyEntryConflict() error {
	entries, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
	}
	for _, f := range entries {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if readErr != nil {
			return readErr
		}
		e, parseErr := parseEntryJSON(data)
		if parseErr == nil && e.Kind == "empty" {
			return &exitError{ExitEntryConflict, "entry conflict: empty entry already exists; normal entry cannot be added"}
		}
	}
	return nil
}

func addEntry(opts addOptions) error {
	if err := checkEntryStorePreconditions(); err != nil {
		return err
	}

	cfg, err := readEntryValidationConfig()
	if err != nil {
		return err
	}

	// Validate kind
	if len(cfg.allowedKinds) > 0 && !cfg.allowedKinds[opts.Kind] {
		return &exitError{ExitCheckFailed, fmt.Sprintf("kind %q is not defined in rellog.entries.kinds.", opts.Kind)}
	}

	// Validate targets: strict structural vocabulary. Targets become
	// release-note headings, so every entry needs at least one declared target.
	if len(opts.Targets) == 0 {
		return &exitError{ExitCheckFailed, "entry must declare at least one target."}
	}
	for _, target := range opts.Targets {
		if !cfg.knownTargets[target] {
			return &exitError{ExitCheckFailed, fmt.Sprintf("target %q is not defined in rellog.entries.targets.", target)}
		}
	}

	var linkErrs []string
	for _, link := range opts.Links {
		linkErrs = append(linkErrs, validateLink(link)...)
	}
	if len(linkErrs) > 0 {
		return &exitError{ExitCheckFailed, strings.Join(linkErrs, "\n")}
	}

	if err := checkNoEmptyEntryConflict(); err != nil {
		return err
	}

	e := entry{
		Kind:    opts.Kind,
		Targets: opts.Targets,
		Links:   opts.Links,
		Body:    opts.Body,
	}
	filename := resolveEntryFilename(opts.DebugDatetime)
	return os.WriteFile(filepath.Join(entriesDir(), filename), formatEntryJSON(e), 0644)
}

func addEmptyEntry(debugDatetime string) error {
	if err := checkEntryStorePreconditions(); err != nil {
		return err
	}

	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if readErr != nil {
			return readErr
		}
		e, parseErr := parseEntryJSON(data)
		if parseErr == nil {
			if e.Kind == "empty" {
				return nil
			}
			return &exitError{ExitEntryConflict, "entry conflict: normal entries already exist; empty entry cannot be added"}
		}
	}

	e := entry{
		Kind:    "empty",
		Targets: []string{},
		Links:   []string{},
		Body:    emptyReleaseMessage,
	}
	filename := resolveEntryFilename(debugDatetime)
	return os.WriteFile(filepath.Join(entriesDir(), filename), formatEntryJSON(e), 0644)
}

func resolveEntryFilename(debugDatetime string) string {
	if debugDatetime != "" {
		return debugDatetime + ".json"
	}
	return entryFilename(time.Now())
}
