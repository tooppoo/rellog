package rellog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func cmdRequire() *cobra.Command {
	requireCmd := &cobra.Command{
		Use:          "require",
		Short:        "Require conditions",
		SilenceUsage: true,
	}

	releaseCmd := &cobra.Command{
		Use:          "release <version>",
		Short:        "Require that a release-note file exists",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := args[0]
			path := filepath.Join(releaseNotesDir(), ver+".json")

			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return &exitError{ExitReleaseNotFound, "release not found: " + ver}
				}
				return err
			}

			var rel releaseData
			if err := json.Unmarshal(data, &rel); err != nil {
				return err
			}

			fmt.Printf("Release %s:\n", rel.Version)
			for _, e := range rel.Entries {
				fmt.Printf("  [%s] %s\n", e.Kind, e.Body)
			}
			return nil
		},
	}

	requireCmd.AddCommand(releaseCmd)
	return requireCmd
}
