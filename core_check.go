package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type checkError struct {
	Code    string
	Message string
}

type fileCheckResult struct {
	Path   string
	Errors []checkError
}

func checkRepository() ([]fileCheckResult, int, error) {
	var results []fileCheckResult
	totalMd := 0

	// Step 0: Check .rellog directory itself
	if info, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			results = append(results, fileCheckResult{
				baseDir,
				[]checkError{{"error[rellog.not_initialized]", "run `rellog init` first"}},
			})
		}
		return results, totalMd, nil
	} else if !info.IsDir() {
		msg := ".rellog path is not a directory.\n\n" +
			"Expected a directory for .rellog, but found a file.\n" +
			"Remove or rename the file, then create the directory:\n" +
			"  mkdir .rellog"
		results = append(results, fileCheckResult{
			baseDir,
			[]checkError{{"error[rellog_dir.not_directory]", msg}},
		})
		return results, totalMd, nil
	}

	// Step 1: Structural check — release-notes must be a directory if it exists
	rnInfo, rnStatErr := os.Stat(releaseNotesDir())
	if rnStatErr == nil && !rnInfo.IsDir() {
		msg := "release-notes path is not a directory.\n\n" +
			"Expected a directory for release-notes, but found a file.\n" +
			"Remove or rename the file, then create the directory:\n" +
			"  mkdir -p " + releaseNotesDir()
		results = append(results, fileCheckResult{
			releaseNotesDir(),
			[]checkError{{"error[release_notes_dir.not_directory]", msg}},
		})
		return results, totalMd, nil
	}

	// Step 2: Structural check — entries must be a directory if it exists
	entInfo, entStatErr := os.Stat(entriesDir())
	if entStatErr == nil && !entInfo.IsDir() {
		msg := "Pending entry path is not a directory.\n\n" +
			"Expected a directory for pending changelog entries, but found a file.\n" +
			"Remove or rename the file, then create the directory:\n" +
			"  mkdir -p " + entriesDir()
		results = append(results, fileCheckResult{
			entriesDir(),
			[]checkError{{"error[entry_dir.not_directory]", msg}},
		})
		return results, totalMd, nil
	}

	// Step 3: Check release-notes existence and accessibility
	if rnStatErr == nil {
		if _, readErr := os.ReadDir(releaseNotesDir()); readErr != nil && os.IsPermission(readErr) {
			results = append(results, fileCheckResult{
				releaseNotesDir(),
				[]checkError{{"error[release_notes.permission]", "permission denied: " + releaseNotesDir()}},
			})
		}
	} else if os.IsNotExist(rnStatErr) {
		results = append(results, fileCheckResult{
			releaseNotesDir(),
			[]checkError{{"error[release_notes_dir.missing]", "missing required directory: " + releaseNotesDir()}},
		})
	}

	// Step 4: Check entries existence and accessibility
	var entFiles []os.DirEntry
	entAccessOk := false
	if entStatErr == nil {
		files, readErr := os.ReadDir(entriesDir())
		if readErr != nil {
			if os.IsPermission(readErr) {
				results = append(results, fileCheckResult{
					entriesDir(),
					[]checkError{{"error[entries_file.permission]", "permission denied: " + entriesDir()}},
				})
			}
		} else {
			entAccessOk = true
			entFiles = files
		}
	} else if os.IsNotExist(entStatErr) {
		results = append(results, fileCheckResult{
			entriesDir(),
			[]checkError{{"error[entries_dir.missing]", "missing required directory: " + entriesDir()}},
		})
	}

	// Step 5: Check config file
	var entryConfig entryValidationConfig
	configOK := true
	if r := checkConfigFile(); r != nil {
		results = append(results, *r)
		configOK = false
	}
	if configOK {
		var err error
		entryConfig, err = readEntryValidationConfig()
		if err != nil {
			return nil, 0, err
		}
	}

	// Step 6: Process entry files (only if entries dir is accessible)
	if entAccessOk {
		// First pass: detect empty/normal conflict.
		type parsedFile struct {
			path string
			e    entry
			ok   bool
		}
		var parsed []parsedFile
		hasEmpty := false
		hasNormal := false
		for _, f := range entFiles {
			if !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			p := filepath.Join(entriesDir(), f.Name())
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				return nil, 0, readErr
			}
			e, parseErr := parseEntry(data)
			if parseErr == nil {
				if e.Kind == "empty" {
					hasEmpty = true
				} else if e.Kind != "" {
					hasNormal = true
				}
			}
			parsed = append(parsed, parsedFile{p, e, parseErr == nil})
		}
		hasConflict := hasEmpty && hasNormal

		// Second pass: validate each file.
		for _, pf := range parsed {
			totalMd++
			var errs []checkError
			if !pf.ok {
				data, _ := os.ReadFile(pf.path)
				_, parseErr := parseEntry(data)
				msg := strings.TrimPrefix(parseErr.Error(), "invalid frontmatter: ")
				errs = append(errs, checkError{"error[entry.frontmatter.parse_failed]", msg})
			} else {
				e := pf.e
				if hasConflict && e.Kind == "empty" {
					errs = append(errs, checkError{
						"error[entry.conflict.empty_and_normal]",
						"entry conflict: empty entry cannot coexist with normal entries.",
					})
				} else {
					if e.Kind == "" {
						errs = append(errs, checkError{"error[entry.kind.missing]", "missing required metadata: kind."})
					} else if configOK && e.Kind != "empty" && !entryConfig.allowedKinds[e.Kind] {
						errs = append(errs, checkError{
							"error[entry.kind.unknown]",
							fmt.Sprintf("kind %q is not defined in rellog.entries.kinds.", e.Kind),
						})
					}
					targetsValidForLookup := true
					switch {
					case e.targetsIsScalar:
						errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must be an array."})
						targetsValidForLookup = false
					case e.targetsKeyPresent && !e.targetsIsScalar && len(e.Targets) == 0:
						errs = append(errs, checkError{"error[entry.targets.missing]", "missing required metadata: target."})
						targetsValidForLookup = false
					}
					if configOK && targetsValidForLookup && e.targetsKeyPresent && entryConfig.targetPolicy != "allow-unknown" {
						for _, target := range e.Targets {
							if entryConfig.knownTargets[target] {
								continue
							}
							code := "error[entry.targets.unknown]"
							if entryConfig.targetPolicy == "warn-unknown" {
								code = "warning[entry.targets.unknown]"
							}
							errs = append(errs, checkError{
								code,
								fmt.Sprintf("target %q is not defined in rellog.entries.targets.", target),
							})
						}
					}
					if e.issuesIsScalar {
						errs = append(errs, checkError{"error[entry.issues.invalid]", "issues must be an array."})
					}
					if e.prsHasNonNumericItem {
						errs = append(errs, checkError{"error[entry.prs.invalid]", "prs item must be a number."})
					}
					if e.Body == "" {
						errs = append(errs, checkError{"error[entry.body.empty]", "entry body must not be empty."})
					}
				}
			}

			if len(errs) > 0 {
				results = append(results, fileCheckResult{pf.path, errs})
			}
		}
	}

	return results, totalMd, nil
}
