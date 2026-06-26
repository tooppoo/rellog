package rellog

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func cmdAdd() *cobra.Command {
	var kind, body string
	var targets, issues, prs []string

	cmd := &cobra.Command{
		Use:          "add",
		Short:        "Add a changelog entry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Interactive mode when no flags are provided.
			if !cmd.Flags().Changed("kind") && !cmd.Flags().Changed("target") && !cmd.Flags().Changed("body") {
				return addEntryInteractive(cmd.InOrStdin())
			}

			var issueNumbers, prNumbers []int
			for _, s := range issues {
				n, _ := strconv.Atoi(s)
				if n != 0 {
					issueNumbers = append(issueNumbers, n)
				}
			}
			for _, s := range prs {
				n, _ := strconv.Atoi(s)
				if n != 0 {
					prNumbers = append(prNumbers, n)
				}
			}

			return addEntry(addOptions{
				Kind:    kind,
				Targets: targets,
				Body:    body,
				Issues:  issueNumbers,
				PRs:     prNumbers,
			})
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Change kind (e.g. changed, fix)")
	cmd.Flags().StringArrayVar(&targets, "target", nil, "Target component (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "Change description")
	cmd.Flags().StringArrayVar(&issues, "issue", nil, "Issue number (repeatable)")
	cmd.Flags().StringArrayVar(&prs, "pr", nil, "PR number (repeatable)")

	return cmd
}

func addEntryInteractive(r io.Reader) error {
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
		Kind:    kind,
		Targets: splitTokens(targetsLine),
		Body:    body,
		Issues:  parseNumberTokens(issuesLine),
		PRs:     parseNumberTokens(prsLine),
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

// parseNumberTokens splits on whitespace/commas and parses integers.
func parseNumberTokens(s string) []int {
	var nums []int
	for _, t := range splitTokens(s) {
		n, err := strconv.Atoi(t)
		if err == nil && n != 0 {
			nums = append(nums, n)
		}
	}
	return nums
}
