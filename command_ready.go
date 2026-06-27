package rellog

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func cmdReady() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:          "ready <release-id>",
		Short:        "Check that a release is ready for publishing",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			releaseID := args[0]

			if err := validateReadyReleaseID(releaseID); err != nil {
				return err
			}

			result, err := checkReady(releaseID)
			if err != nil {
				return err
			}

			if jsonOutput {
				out, marshalErr := json.MarshalIndent(result, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Println(string(out))
				if !result.OK {
					return &exitError{ExitReleaseNotReady, ""}
				}
				return nil
			}

			if !result.OK {
				return buildReadyHumanError(result)
			}

			fmt.Printf("%s release ready\n", releaseID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output machine-readable JSON")
	return cmd
}
