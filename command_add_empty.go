package rellog

import "github.com/spf13/cobra"

func cmdAddEmpty() *cobra.Command {
	var debugDatetime string

	cmd := &cobra.Command{
		Use:          "add-empty",
		Short:        "Add an empty changelog entry (no changelog-worthy changes)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addEmptyEntry(debugDatetime)
		},
	}

	cmd.Flags().StringVar(&debugDatetime, "debug-datetime", "", "Override entry timestamp for testing")
	return cmd
}
