package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type addOptions struct {
	Kind    string
	Targets []string
	Body    string
	Issues  []int
	PRs     []int
}

func addEntry(opts addOptions) error {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return &exitError{ExitNotInitialized, "run `rellog init` first"}
	}
	if info, err := os.Stat(entriesDir()); err == nil && !info.IsDir() {
		return &exitError{ExitInvalidStructure, entriesDir() + " is not a directory"}
	}

	cfg, err := readEntryValidationConfig()
	if err != nil {
		return err
	}

	// Validate kind
	if len(cfg.allowedKinds) > 0 && !cfg.allowedKinds[opts.Kind] {
		return &exitError{ExitCheckFailed, fmt.Sprintf("kind %q is not defined in rellog.entries.kinds.", opts.Kind)}
	}

	// Validate targets
	if cfg.targetPolicy != "allow-unknown" {
		for _, target := range opts.Targets {
			if !cfg.knownTargets[target] {
				if cfg.targetPolicy == "warn-unknown" {
					fmt.Fprintf(os.Stderr, "target %q is not defined in rellog.entries.targets.\n", target)
				} else {
					return &exitError{ExitCheckFailed, fmt.Sprintf("target %q is not defined in rellog.entries.targets.", target)}
				}
			}
		}
	}

	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
	}

	// Check for empty entry conflict
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if readErr != nil {
			return readErr
		}
		e, parseErr := parseEntry(data)
		if parseErr == nil && e.Kind == "empty" {
			return &exitError{ExitEntryConflict, "entry conflict: empty entry already exists; normal entry cannot be added"}
		}
	}

	count := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".md") {
			count++
		}
	}

	e := entry{
		Kind:    opts.Kind,
		Targets: opts.Targets,
		Issues:  opts.Issues,
		PRs:     opts.PRs,
		Body:    opts.Body,
	}
	filename := fmt.Sprintf("%04d.md", count+1)
	return os.WriteFile(filepath.Join(entriesDir(), filename), []byte(formatEntry(e)), 0644)
}

func addEmptyEntry() error {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return &exitError{ExitNotInitialized, "run `rellog init` first"}
	}
	if info, err := os.Stat(entriesDir()); err == nil && !info.IsDir() {
		return &exitError{ExitInvalidStructure, entriesDir() + " is not a directory"}
	}

	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
	}

	count := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		count++
		data, readErr := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if readErr != nil {
			return readErr
		}
		e, parseErr := parseEntry(data)
		if parseErr == nil {
			if e.Kind == "empty" {
				// Already have an empty entry; no-op.
				return nil
			}
			// Normal entry exists — conflict.
			return &exitError{ExitEntryConflict, "entry conflict: normal entries already exist; empty entry cannot be added"}
		}
	}

	e := entry{
		Kind: "empty",
		Body: "\nNo changelog-worthy changes.",
	}
	filename := fmt.Sprintf("%04d.md", count+1)
	return os.WriteFile(filepath.Join(entriesDir(), filename), []byte(formatEntry(e)), 0644)
}
