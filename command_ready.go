package rellog

import (
	"fmt"

	"github.com/spf13/cobra"
)

func cmdReady() *cobra.Command {
	return &cobra.Command{
		Use:          "ready <release-id>",
		Short:        "Check that a release-note file exists for the given release id",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rel, err := requireRelease(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Release %s:\n", rel.Version)
			for _, e := range rel.Entries {
				fmt.Printf("  [%s] %s\n", e.Kind, e.Body)
			}
			return nil
		},
	}
}
