package rellog

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type releaseData struct {
	Version string  `json:"version"`
	Entries []entry `json:"entries"`
}

func cmdPrepare() *cobra.Command {
	return &cobra.Command{
		Use:          "prepare <version>",
		Short:        "Prepare a release",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := args[0]
			entries, err := readEntries()
			if err != nil {
				return err
			}

			rel := releaseData{Version: ver, Entries: entries}
			data, err := json.MarshalIndent(rel, "", "  ")
			if err != nil {
				return err
			}

			path := filepath.Join(releaseNotesDir(), ver+".json")
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}

			files, err := os.ReadDir(entriesDir())
			if err != nil {
				return err
			}
			for _, f := range files {
				_ = os.Remove(filepath.Join(entriesDir(), f.Name()))
			}
			return nil
		},
	}
}
