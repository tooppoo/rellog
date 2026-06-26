package rellog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	kdl "github.com/njreid/gokdl2"
	"github.com/spf13/cobra"
)

const appVersion = "0.0.1"

// Exit codes returned by rellog commands.
const (
	ExitNotInitialized   = 1 // rellog has not been initialized (run rellog init first)
	ExitInvalidStructure = 2 // rellog directory structure is invalid (expected directory is a file)
	ExitCheckFailed      = 3 // rellog check found validation errors
	ExitReleaseNotFound  = 4 // required release-note file does not exist
)

type exitError struct {
	Code int
	Msg  string
}

func (e *exitError) Error() string { return e.Msg }

type entry struct {
	Kind                 string
	Targets              []string
	Scope                string
	Issues               []int
	PRs                  []int
	Body                 string
	targetsKeyPresent    bool
	targetsIsScalar      bool
	scopeKeyPresent      bool
	issuesIsScalar       bool
	prsHasNonNumericItem bool
}

type checkError struct {
	Code    string
	Message string
}

type fileCheckResult struct {
	Path   string
	Errors []checkError
}

type releaseData struct {
	Version string  `json:"version"`
	Entries []entry `json:"entries"`
}

const baseDir = ".rellog"

func configFile() string {
	return filepath.Join(baseDir, "config.kdl")
}

func entriesDir() string {
	return filepath.Join(baseDir, "entries")
}

func releaseNotesDir() string {
	return filepath.Join(baseDir, "release-notes")
}

func Main() {
	var showVersion bool

	root := &cobra.Command{
		Use:           "rellog",
		Short:         "Release log management tool",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Printf("rellog v%s\n", appVersion)
				return nil
			}
			return cmd.Help()
		},
	}
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version")

	root.AddCommand(
		cmdInit(),
		cmdAdd(),
		cmdCheck(),
		cmdStatus(),
		cmdPrepare(),
		cmdRequire(),
	)

	if err := root.Execute(); err != nil {
		if ee, ok := errors.AsType[*exitError](err); ok {
			if ee.Msg != "" {
				fmt.Fprintf(os.Stderr, "Error: %s\n", ee.Msg)
			}
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "Initialize rellog directory",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(entriesDir(), 0755); err != nil {
				return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", entriesDir(), err)}
			}
			if err := os.MkdirAll(releaseNotesDir(), 0755); err != nil {
				return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", releaseNotesDir(), err)}
			}
			// Only create config if it doesn't already exist as a regular file (preserve user's config)
			if info, err := os.Stat(configFile()); err == nil && info.Mode().IsRegular() {
				return nil
			}
			if err := os.WriteFile(configFile(), []byte("/- kdl-version 2\n"), 0644); err != nil {
				return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", configFile(), err)}
			}
			return nil
		},
	}
}

func cmdAdd() *cobra.Command {
	var kind, scope, body string
	var targets, issues, prs []string

	cmd := &cobra.Command{
		Use:          "add",
		Short:        "Add a changelog entry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
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
				if strings.HasSuffix(f.Name(), ".md") {
					count++
				}
			}

			e := entry{
				Kind:    kind,
				Targets: targets,
				Scope:   scope,
				Body:    body,
			}
			for _, s := range issues {
				n, _ := strconv.Atoi(s)
				if n != 0 {
					e.Issues = append(e.Issues, n)
				}
			}
			for _, s := range prs {
				n, _ := strconv.Atoi(s)
				if n != 0 {
					e.PRs = append(e.PRs, n)
				}
			}
			filename := fmt.Sprintf("%04d.md", count+1)
			return os.WriteFile(filepath.Join(entriesDir(), filename), []byte(formatEntry(e)), 0644)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Change kind (e.g. changed, fix)")
	cmd.Flags().StringArrayVar(&targets, "target", nil, "Target component (repeatable)")
	cmd.Flags().StringVar(&scope, "scope", "", "Change scope")
	cmd.Flags().StringVar(&body, "body", "", "Change description")
	cmd.Flags().StringArrayVar(&issues, "issue", nil, "Issue number (repeatable)")
	cmd.Flags().StringArrayVar(&prs, "pr", nil, "PR number (repeatable)")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("target")
	_ = cmd.MarkFlagRequired("scope")
	_ = cmd.MarkFlagRequired("body")

	return cmd
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

func checkConfigFile() *fileCheckResult {
	info, statErr := os.Stat(configFile())
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return &fileCheckResult{
				configFile(),
				[]checkError{{"error[config.missing] " + configFile(), "rellog configuration file does not exist."}},
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		msg := "rellog configuration file is not a regular file.\n\n" +
			"Expected a KDL file for config, but found a directory.\n" +
			"Remove the directory, then create a file:\n" +
			"  touch " + configFile()
		return &fileCheckResult{
			configFile(),
			[]checkError{{"error[config.not_file]", msg}},
		}
	}
	data, err := os.ReadFile(configFile())
	if err != nil {
		if os.IsPermission(err) {
			return &fileCheckResult{
				configFile(),
				[]checkError{{"error[config_file.permission]", "permission denied: " + configFile()}},
			}
		}
		return nil
	}
	if _, parseErr := kdl.Parse(strings.NewReader(string(data))); parseErr != nil {
		msg := configFile() + ": " + parseErr.Error() + "\n\n" +
			"Failed to parse rellog configuration file.\n\n" +
			"Fix the KDL syntax error and run `rellog check` again."
		return &fileCheckResult{
			configFile(),
			[]checkError{{"error[config.parse_failed]", msg}},
		}
	}
	return nil
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

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show unreleased entries",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := readEntries()
			if err != nil {
				return err
			}
			fmt.Printf("Unreleased: %d entries\n", len(entries))
			for _, e := range entries {
				fmt.Printf("  [%s] %s\n", e.Kind, e.Body)
			}
			return nil
		},
	}
}

func cmdPrepare() *cobra.Command {
	return &cobra.Command{
		Use:          "prepare <version>",
		Short:        "Prepare a release",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := args[0]
			entries, err := readEntries()
			if err != nil {
				return err
			}

			rel := releaseData{Version: ver, Entries: entries}
			data, err := json.MarshalIndent(rel, "", "  ")
			if err != nil {
				return err
			}

			path := filepath.Join(releaseNotesDir(), ver+".json")
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}

			files, err := os.ReadDir(entriesDir())
			if err != nil {
				return err
			}
			for _, f := range files {
				_ = os.Remove(filepath.Join(entriesDir(), f.Name()))
			}
			return nil
		},
	}
}

func cmdRequire() *cobra.Command {
	requireCmd := &cobra.Command{
		Use:          "require",
		Short:        "Require conditions",
		SilenceUsage: true,
	}

	releaseCmd := &cobra.Command{
		Use:          "release <version>",
		Short:        "Require that a release-note file exists",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := args[0]
			path := filepath.Join(releaseNotesDir(), ver+".json")

			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return &exitError{ExitReleaseNotFound, "release not found: " + ver}
				}
				return err
			}

			var rel releaseData
			if err := json.Unmarshal(data, &rel); err != nil {
				return err
			}

			fmt.Printf("Release %s:\n", rel.Version)
			for _, e := range rel.Entries {
				fmt.Printf("  [%s] %s\n", e.Kind, e.Body)
			}
			return nil
		},
	}

	requireCmd.AddCommand(releaseCmd)
	return requireCmd
}

func formatEntry(e entry) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "kind: %s\n", e.Kind)
	sb.WriteString("targets:\n")
	for _, t := range e.Targets {
		fmt.Fprintf(&sb, "  - %s\n", t)
	}
	fmt.Fprintf(&sb, "scope: %s\n", e.Scope)
	if len(e.Issues) > 0 {
		sb.WriteString("issues:\n")
		for _, i := range e.Issues {
			fmt.Fprintf(&sb, "  - %d\n", i)
		}
	}
	if len(e.PRs) > 0 {
		sb.WriteString("prs:\n")
		for _, p := range e.PRs {
			fmt.Fprintf(&sb, "  - %d\n", p)
		}
	}
	sb.WriteString("---\n")
	sb.WriteString(e.Body)
	sb.WriteString("\n")
	return sb.String()
}

func parseEntry(data []byte) (entry, error) {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return entry{}, fmt.Errorf("invalid frontmatter: missing opening ---")
	}
	rest := s[4:]
	frontmatter, after, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return entry{}, fmt.Errorf("invalid frontmatter: missing closing ---")
	}
	body := strings.TrimRight(after, "\n")

	e := entry{Body: body}
	var currentList string
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.HasPrefix(line, "  - ") {
			item := strings.TrimPrefix(line, "  - ")
			switch currentList {
			case "targets":
				e.Targets = append(e.Targets, item)
			case "issues":
				n, _ := strconv.Atoi(item)
				e.Issues = append(e.Issues, n)
			case "prs":
				n, err := strconv.Atoi(item)
				if err != nil {
					e.prsHasNonNumericItem = true
				} else {
					e.PRs = append(e.PRs, n)
				}
			}
			continue
		}
		currentList = ""
		k, v, hasVal := strings.Cut(line, ": ")
		if hasVal {
			switch k {
			case "kind":
				e.Kind = v
			case "scope":
				e.Scope = v
				e.scopeKeyPresent = true
			case "targets":
				e.targetsKeyPresent = true
				e.targetsIsScalar = true
				_ = v
			case "issues":
				e.issuesIsScalar = true
				_ = v
			}
		} else if strings.HasSuffix(line, ":") {
			currentList = strings.TrimSuffix(line, ":")
			switch currentList {
			case "targets":
				e.targetsKeyPresent = true
			case "scope":
				e.scopeKeyPresent = true
			}
		}
	}
	return e, nil
}

func readEntries() ([]entry, error) {
	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return nil, err
	}

	var entries []entry
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if err != nil {
			return nil, err
		}
		e, err := parseEntry(data)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
