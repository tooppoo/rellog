package rellog

import "github.com/spf13/cobra"

func cmdPrepare() *cobra.Command {
	var run bool

	cmd := &cobra.Command{
		Use:          "prepare <version>",
		Short:        "Prepare a release",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return prepareRelease(prepareOptions{
				Version: args[0],
				DryRun:  !run,
			})
		},
	}

	cmd.Flags().BoolVar(&run, "run", false, "Execute the release preparation (default is dry-run)")
	return cmd
}
