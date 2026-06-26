package rellog

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type releaseData struct {
	Version string  `json:"version"`
	Entries []entry `json:"entries"`
}

func prepareRelease(version string) error {
	entries, err := readEntries()
	if err != nil {
		return err
	}

	rel := releaseData{Version: version, Entries: entries}
	data, err := json.MarshalIndent(rel, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(releaseNotesDir(), version+".json")
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
}
