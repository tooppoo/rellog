package rellog

import (
	"github.com/spf13/cobra"
)

func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "Initialize rellog directory",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return initRellog()
		},
	}
}
