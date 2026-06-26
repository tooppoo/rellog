package rellog

import (
	"strconv"

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
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("target")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}
