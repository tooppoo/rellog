package rellog

import "github.com/spf13/cobra"

func cmdAddEmpty() *cobra.Command {
	return &cobra.Command{
		Use:          "add-empty",
		Short:        "Add an empty changelog entry (no changelog-worthy changes)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addEmptyEntry()
		},
	}
}
