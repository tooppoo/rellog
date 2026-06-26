package rellog

import (
	"fmt"

	"github.com/spf13/cobra"
)

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show unreleased entries",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := readEntries()
			if err != nil {
				return err
			}
			fmt.Printf("Unreleased: %d entries\n", len(entries))
			for _, e := range entries {
				fmt.Printf("  [%s] %s\n", e.Kind, e.Body)
			}
			return nil
		},
	}
}
