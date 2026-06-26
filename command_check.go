package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type checkError struct {
	Code    string
	Message string
}

type fileCheckResult struct {
	Path   string
	Errors []checkError
}

func cmdCheck() *cobra.Command {
	return &cobra.Command{
		Use:          "check",
		Short:        "Validate unreleased entries",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
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
				return reportCheckResults(results, totalMd)
			} else if !info.IsDir() {
				msg := ".rellog path is not a directory.\n\n" +
					"Expected a directory for .rellog, but found a file.\n" +
					"Remove or rename the file, then create the directory:\n" +
					"  mkdir .rellog"
				results = append(results, fileCheckResult{
					baseDir,
					[]checkError{{"error[rellog_dir.not_directory]", msg}},
				})
				return reportCheckResults(results, totalMd)
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
				return reportCheckResults(results, totalMd)
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
				return reportCheckResults(results, totalMd)
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
			if r := checkConfigFile(); r != nil {
				results = append(results, *r)
			}

			// Step 6: Process entry files (only if entries dir is accessible)
			if entAccessOk {
				for _, f := range entFiles {
					if !strings.HasSuffix(f.Name(), ".md") {
						continue
					}
					totalMd++
					path := filepath.Join(entriesDir(), f.Name())
					data, err := os.ReadFile(path)
					if err != nil {
						return err
					}

					var errs []checkError
					e, parseErr := parseEntry(data)
					if parseErr != nil {
						msg := strings.TrimPrefix(parseErr.Error(), "invalid frontmatter: ")
						errs = append(errs, checkError{"error[entry.frontmatter.parse_failed]", msg})
					} else {
						if e.Kind == "" {
							errs = append(errs, checkError{"error[entry.kind.missing]", "missing required metadata: kind."})
						}
						switch {
						case e.targetsIsScalar && e.scopeKeyPresent:
							errs = append(errs, checkError{"error[entry.targets.invalid]", "targets must be an array."})
						case e.targetsKeyPresent && !e.targetsIsScalar && len(e.Targets) == 0:
							errs = append(errs, checkError{"error[entry.targets.missing]", "missing required metadata: target."})
						}
						if e.Scope == "" {
							errs = append(errs, checkError{"error[entry.scope.missing]", "missing required metadata: scope."})
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

					if len(errs) > 0 {
						results = append(results, fileCheckResult{path, errs})
					}
				}
			}

			return reportCheckResults(results, totalMd)
		},
	}
}

func reportCheckResults(results []fileCheckResult, totalMd int) error {
	if len(results) == 0 {
		fmt.Printf("rellog check: OK (entries: %d)\n", totalMd)
		return nil
	}

	totalErrs := 0
	for _, r := range results {
		totalErrs += len(r.Errors)
	}
	fmt.Fprintf(os.Stderr, "rellog check: FAILED\n\n%d files\n%d errors\n\n", len(results), totalErrs)
	for i, r := range results {
		fmt.Fprintf(os.Stderr, "%s\n", r.Path)
		for j, ce := range r.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", ce.Code)
			for _, msgLine := range strings.Split(ce.Message, "\n") {
				if msgLine == "" {
					fmt.Fprintln(os.Stderr)
				} else {
					fmt.Fprintf(os.Stderr, "    %s\n", msgLine)
				}
			}
			if i < len(results)-1 || j < len(r.Errors)-1 {
				fmt.Fprintln(os.Stderr)
			}
		}
	}

	return &exitError{ExitCheckFailed, ""}
}
