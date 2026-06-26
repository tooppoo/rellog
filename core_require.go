package rellog

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func requireRelease(version string) (releaseData, error) {
	path := filepath.Join(releaseNotesDir(), version+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return releaseData{}, &exitError{ExitReleaseNotFound, "release not found: " + version}
		}
		return releaseData{}, err
	}

	var rel releaseData
	if err := json.Unmarshal(data, &rel); err != nil {
		return releaseData{}, err
	}
	return rel, nil
}
