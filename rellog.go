package rellog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const appVersion = "0.0.1"

type entry struct {
	Kind   string
	Target string
	Scope  string
	Issue  int
	PR     int
	Body   string
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
				return err
			}
			if err := os.MkdirAll(releaseNotesDir(), 0755); err != nil {
				return err
			}
			return os.WriteFile(configFile(), []byte("// rellog config\n"), 0644)
		},
	}
}

func cmdAdd() *cobra.Command {
	var kind, target, scope, body string
	var issue, pr int

	cmd := &cobra.Command{
		Use:          "add",
		Short:        "Add a changelog entry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			e := entry{Kind: kind, Target: target, Scope: scope, Issue: issue, PR: pr, Body: body}
			filename := fmt.Sprintf("%04d.md", count+1)
			return os.WriteFile(filepath.Join(entriesDir(), filename), []byte(formatEntry(e)), 0644)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Change kind (e.g. changed, fix)")
	cmd.Flags().StringVar(&target, "target", "", "Target component")
	cmd.Flags().StringVar(&scope, "scope", "", "Change scope")
	cmd.Flags().StringVar(&body, "body", "", "Change description")
	cmd.Flags().IntVar(&issue, "issue", 0, "Issue number")
	cmd.Flags().IntVar(&pr, "pr", 0, "PR number")
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
			entries, err := readEntries()
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.Kind == "" {
					return fmt.Errorf("entry missing kind")
				}
				if e.Body == "" {
					return fmt.Errorf("entry missing body")
				}
			}
			return nil
		},
	}
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
		Short:        "Show release if it exists",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := args[0]
			path := filepath.Join(releaseNotesDir(), ver+".json")

			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
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
	fmt.Fprintf(&sb, "target: %s\n", e.Target)
	fmt.Fprintf(&sb, "scope: %s\n", e.Scope)
	if e.Issue > 0 {
		fmt.Fprintf(&sb, "issue: %d\n", e.Issue)
	}
	if e.PR > 0 {
		fmt.Fprintf(&sb, "pr: %d\n", e.PR)
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
	for line := range strings.SplitSeq(frontmatter, "\n") {
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch k {
		case "kind":
			e.Kind = v
		case "target":
			e.Target = v
		case "scope":
			e.Scope = v
		case "issue":
			n, _ := strconv.Atoi(v)
			e.Issue = n
		case "pr":
			n, _ := strconv.Atoi(v)
			e.PR = n
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
