package rellog

import (
	"os"
	"path/filepath"
)

func requireRelease(version string) (releaseData, error) {
	path := filepath.Join(releaseNotesDir(), version+".md")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return releaseData{}, &exitError{ExitReleaseNotFound, "release not found: " + version}
		}
		return releaseData{}, err
	}
	return releaseData{Version: version}, nil
}
