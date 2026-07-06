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

// validateEntrySchema checks that e conforms to the structural entry schema
// (unknown fields, required fields present, correct types, non-empty body, valid
// links). It intentionally omits config-based whitelist checks (allowed kinds,
// known targets) so that it can be used on entries that have already passed the
// full check suite, such as consumed cache entries.
func validateEntrySchema(e entry) []checkError {
	var errs []checkError

	for _, field := range e.unknownFields {
		errs = append(errs, checkError{
			"error[entry.unknown_field]",
			fmt.Sprintf("unknown entry field: %s.\n\nRemove unknown fields from the entry JSON.", field),
		})
	}

	if e.Kind == "empty" {
		// For empty entries, targets, links and body must be present with correct types
		// and empty values.
		switch {
		case !e.targetsPresent:
			errs = append(errs, checkError{"error[entry.targets.missing]", "missing required metadata: targets."})
		case e.targetsIsScalar:
			errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must be an array."})
		case e.targetsTypeError:
			errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must contain only strings."})
		}
		switch {
		case !e.linksPresent:
			errs = append(errs, checkError{"error[entry.links.missing]", "missing required metadata: links."})
		case e.linksIsScalar:
			errs = append(errs, checkError{"error[entry.links.invalid]", "links must be an array."})
		case e.linksTypeError:
			errs = append(errs, checkError{"error[entry.links.invalid]", "links must contain only strings."})
		}
		if !e.bodyPresent {
			errs = append(errs, checkError{"error[entry.body.missing]", "missing required metadata: body."})
		} else if e.bodyTypeError {
			errs = append(errs, checkError{"error[entry.body.invalid]", "body must be a string."})
		}
		errs = append(errs, checkEmptyEntryFields(e)...)
		return errs
	}

	if e.Kind == "" {
		errs = append(errs, checkError{"error[entry.kind.missing]", "missing required metadata: kind."})
	}

	switch {
	case !e.targetsPresent:
		errs = append(errs, checkError{"error[entry.targets.missing]", "missing required metadata: targets."})
	case e.targetsIsScalar:
		errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must be an array."})
	case e.targetsTypeError:
		errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must contain only strings."})
	case len(e.Targets) == 0:
		errs = append(errs, checkError{"error[entry.targets.empty]", "entry must declare at least one target."})
	}

	if !e.linksPresent {
		errs = append(errs, checkError{"error[entry.links.missing]", "missing required metadata: links."})
	} else if e.linksIsScalar {
		errs = append(errs, checkError{"error[entry.links.invalid]", "links must be an array."})
	} else if e.linksTypeError {
		errs = append(errs, checkError{"error[entry.links.invalid]", "links must contain only strings."})
	} else {
		for _, link := range e.Links {
			for _, msg := range validateLink(link) {
				errs = append(errs, checkError{"error[entry.links.invalid]", msg + "."})
			}
		}
	}

	if !e.bodyPresent {
		errs = append(errs, checkError{"error[entry.body.missing]", "missing required metadata: body."})
	} else if e.bodyTypeError {
		errs = append(errs, checkError{"error[entry.body.invalid]", "body must be a string."})
	} else if e.Body == "" {
		errs = append(errs, checkError{"error[entry.body.empty]", "entry body must not be empty."})
	} else if strings.Contains(e.Body, reservedMarkerPrefix) {
		errs = append(errs, checkError{
			"error[entry.body.reserved_marker]",
			"entry body must not contain rellog reserved marker comments.\n\nRemove comments beginning with `<!-- rellog:` from the entry body.",
		})
	}

	return errs
}

func checkEmptyEntryFields(e entry) []checkError {
	var errs []checkError
	if len(e.Targets) > 0 {
		errs = append(errs, checkError{"error[entry.empty.targets.invalid]", "empty entry targets must be an empty array."})
	}
	if len(e.Links) > 0 {
		errs = append(errs, checkError{"error[entry.empty.links.invalid]", "empty entry links must be an empty array."})
	}
	return errs
}

func checkRepository() ([]fileCheckResult, int, error) {
	var results []fileCheckResult
	totalEntries := 0

	// Step 0: Check .rellog directory itself
	if info, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			results = append(results, fileCheckResult{
				baseDir,
				[]checkError{{"error[rellog.not_initialized]", "run `rellog init` first"}},
			})
		}
		return results, totalEntries, nil
	} else if !info.IsDir() {
		msg := ".rellog path is not a directory.\n\n" +
			"Expected a directory for .rellog, but found a file.\n" +
			"Remove or rename the file, then create the directory:\n" +
			"  mkdir .rellog"
		results = append(results, fileCheckResult{
			baseDir,
			[]checkError{{"error[rellog_dir.not_directory]", msg}},
		})
		return results, totalEntries, nil
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
		return results, totalEntries, nil
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
		return results, totalEntries, nil
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
		type parsedFile struct {
			path        string
			e           entry
			parseOK     bool
			unsupported bool
		}
		var parsed []parsedFile
		hasEmpty := false
		hasNormal := false

		for _, f := range entFiles {
			p := filepath.Join(entriesDir(), f.Name())
			if !strings.HasSuffix(f.Name(), ".json") {
				parsed = append(parsed, parsedFile{path: p, unsupported: true})
				continue
			}
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				return nil, 0, readErr
			}
			e, parseErr := parseEntryJSON(data)
			ok := parseErr == nil
			if ok {
				if e.Kind == "empty" {
					hasEmpty = true
				} else if e.Kind != "" {
					hasNormal = true
				}
			}
			parsed = append(parsed, parsedFile{path: p, e: e, parseOK: ok})
		}
		hasConflict := hasEmpty && hasNormal

		for _, pf := range parsed {
			if pf.unsupported {
				results = append(results, fileCheckResult{pf.path, []checkError{
					{"error[entry.file.unsupported]", "pending entry files must use the .json extension."},
				}})
				continue
			}

			totalEntries++
			var errs []checkError

			if !pf.parseOK {
				errs = append(errs, checkError{"error[entry.json.parse_failed]", "invalid JSON entry."})
			} else {
				e := pf.e
				for _, field := range e.unknownFields {
					errs = append(errs, checkError{
						"error[entry.unknown_field]",
						fmt.Sprintf("unknown entry field: %s.\n\nRemove unknown fields from the entry JSON.", field),
					})
				}
				if e.Kind == "empty" {
					emptyFieldErrs := checkEmptyEntryFields(e)
					if len(emptyFieldErrs) > 0 {
						errs = append(errs, emptyFieldErrs...)
					} else if hasConflict {
						errs = append(errs, checkError{
							"error[entry.conflict.empty_and_normal]",
							"entry conflict: empty entry cannot coexist with normal entries.",
						})
					}
				} else {
					if e.Kind == "" {
						errs = append(errs, checkError{"error[entry.kind.missing]", "missing required metadata: kind."})
					} else if configOK && len(entryConfig.allowedKinds) > 0 && !entryConfig.allowedKinds[e.Kind] {
						errs = append(errs, checkError{
							"error[entry.kind.unknown]",
							fmt.Sprintf("kind %q is not defined in rellog.entries.kinds.", e.Kind),
						})
					}

					switch {
					case !e.targetsPresent:
						errs = append(errs, checkError{"error[entry.targets.missing]", "missing required metadata: targets."})
					case e.targetsIsScalar:
						errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must be an array."})
					case e.targetsTypeError:
						errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must contain only strings."})
					case len(e.Targets) == 0:
						errs = append(errs, checkError{"error[entry.targets.empty]", "entry must declare at least one target."})
					default:
						if configOK {
							for _, target := range e.Targets {
								if entryConfig.knownTargets[target] {
									continue
								}
								errs = append(errs, checkError{
									"error[entry.targets.unknown]",
									fmt.Sprintf("target %q is not defined in rellog.entries.targets.", target),
								})
							}
						}
					}

					if !e.linksPresent {
						errs = append(errs, checkError{"error[entry.links.missing]", "missing required metadata: links."})
					} else if e.linksIsScalar {
						errs = append(errs, checkError{"error[entry.links.invalid]", "links must be an array."})
					} else if e.linksTypeError {
						errs = append(errs, checkError{"error[entry.links.invalid]", "links must contain only strings."})
					} else {
						for _, link := range e.Links {
							for _, msg := range validateLink(link) {
								errs = append(errs, checkError{"error[entry.links.invalid]", msg + "."})
							}
						}
					}

					if !e.bodyPresent {
						errs = append(errs, checkError{"error[entry.body.missing]", "missing required metadata: body."})
					} else if e.bodyTypeError {
						errs = append(errs, checkError{"error[entry.body.invalid]", "body must be a string."})
					} else if e.Body == "" {
						errs = append(errs, checkError{"error[entry.body.empty]", "entry body must not be empty."})
					} else if strings.Contains(e.Body, reservedMarkerPrefix) {
						errs = append(errs, checkError{
							"error[entry.body.reserved_marker]",
							"entry body must not contain rellog reserved marker comments.\n\nRemove comments beginning with `<!-- rellog:` from the entry body.",
						})
					}
				}
			}

			if len(errs) > 0 {
				results = append(results, fileCheckResult{pf.path, errs})
			}
		}
	}

	return results, totalEntries, nil
}
