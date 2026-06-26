package rellog

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const appVersion = "0.0.1"

func Main() {
	var showVersion bool

	root := &cobra.Command{
		Use:           "rellog",
		Short:         "Release log management tool",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Printf("rellog v%s\n", appVersion)
				return nil
			}
			return cmd.Help()
		},
	}
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version")

	root.AddCommand(
		cmdInit(),
		cmdAdd(),
		cmdAddEmpty(),
		cmdCheck(),
		cmdStatus(),
		cmdPrepare(),
		cmdRequire(),
	)

	if err := root.Execute(); err != nil {
		if ee, ok := errors.AsType[*exitError](err); ok {
			if ee.Msg != "" {
				fmt.Fprintf(os.Stderr, "Error: %s\n", ee.Msg)
			}
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
