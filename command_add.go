package rellog

import (
	"bufio"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func cmdAdd() *cobra.Command {
	var kind, body, debugDatetime string
	var targets, issues, prs []string

	cmd := &cobra.Command{
		Use:          "add",
		Short:        "Add a changelog entry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Interactive mode when no flags are provided (except --debug-datetime).
			if !cmd.Flags().Changed("kind") && !cmd.Flags().Changed("target") && !cmd.Flags().Changed("body") {
				return addEntryInteractive(cmd.InOrStdin(), debugDatetime)
			}

			return addEntry(addOptions{
				Kind:          kind,
				Targets:       targets,
				Body:          body,
				Issues:        filterEmpty(issues),
				PRs:           filterEmpty(prs),
				DebugDatetime: debugDatetime,
			})
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Change kind (e.g. changed, fix)")
	cmd.Flags().StringArrayVar(&targets, "target", nil, "Target component (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "Change description")
	cmd.Flags().StringArrayVar(&issues, "issue", nil, "Issue number or URL (repeatable)")
	cmd.Flags().StringArrayVar(&prs, "pr", nil, "PR number or URL (repeatable)")
	cmd.Flags().StringVar(&debugDatetime, "debug-datetime", "", "Override entry timestamp for testing")

	return cmd
}

func addEntryInteractive(r io.Reader, debugDatetime string) error {
	scanner := bufio.NewScanner(r)

	readLine := func() string {
		if scanner.Scan() {
			return strings.TrimRight(scanner.Text(), "\r")
		}
		return ""
	}

	kind := readLine()
	targetsLine := readLine()
	body := readLine()
	issuesLine := readLine()
	prsLine := readLine()

	return addEntry(addOptions{
		Kind:          kind,
		Targets:       splitTokens(targetsLine),
		Body:          body,
		Issues:        splitTokens(issuesLine),
		PRs:           splitTokens(prsLine),
		DebugDatetime: debugDatetime,
	})
}

// splitTokens splits a string on whitespace and commas, returning non-empty tokens.
func splitTokens(s string) []string {
	s = strings.ReplaceAll(s, ",", " ")
	var result []string
	for _, t := range strings.Fields(s) {
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

// filterEmpty removes empty strings from a slice.
func filterEmpty(ss []string) []string {
	var result []string
	for _, s := range ss {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
