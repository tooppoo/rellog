package rellog

import "github.com/spf13/cobra"

func cmdAmend() *cobra.Command {
	var run bool

	cmd := &cobra.Command{
		Use:          "amend <release-id>",
		Short:        "Add pending entries to an already-prepared release",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return amendRelease(amendOptions{
				Version: args[0],
				DryRun:  !run,
			})
		},
	}

	cmd.Flags().BoolVar(&run, "run", false, "Execute the amendment (default is dry-run)")
	return cmd
}
