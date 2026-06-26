package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

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
