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
	Issues        []string
	PRs           []string
	DebugDatetime string
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

	// Normalize and validate issue/PR references
	if len(opts.Issues) > 0 || len(opts.PRs) > 0 {
		if cfg.githubURL == "" {
			return &exitError{ExitCheckFailed, "github-url is required in " + configFile()}
		}
	}

	var normalizedIssues []string
	var normalizedPRs []string
	var refErrs []string

	for _, ref := range opts.Issues {
		normalized, errs := validateAndNormalizeIssueRef(ref, cfg.githubURL)
		if len(errs) > 0 {
			refErrs = append(refErrs, errs...)
		} else {
			normalizedIssues = append(normalizedIssues, normalized)
		}
	}
	for _, ref := range opts.PRs {
		normalized, errs := validateAndNormalizePRRef(ref, cfg.githubURL)
		if len(errs) > 0 {
			refErrs = append(refErrs, errs...)
		} else {
			normalizedPRs = append(normalizedPRs, normalized)
		}
	}

	if len(refErrs) > 0 {
		return &exitError{ExitCheckFailed, strings.Join(refErrs, "\n")}
	}

	// Check for empty entry conflict
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

	e := entry{
		Kind:    opts.Kind,
		Targets: opts.Targets,
		Issues:  normalizedIssues,
		PRs:     normalizedPRs,
		Body:    opts.Body,
	}
	filename := resolveEntryFilename(opts.DebugDatetime)
	return os.WriteFile(filepath.Join(entriesDir(), filename), formatEntryJSON(e), 0644)
}

func addEmptyEntry(debugDatetime string) error {
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
		Kind: "empty",
		Body: "No changelog-worthy changes.",
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
