package rellog

import "path/filepath"

const baseDir = ".rellog"

func configFile() string {
	return filepath.Join(baseDir, "config.kdl")
}

func entriesDir() string {
	return filepath.Join(baseDir, "entries")
}

func releaseNotesDir() string {
	return filepath.Join(baseDir, "release-notes")
}

func consumedDir(releaseID string) string {
	return filepath.Join(baseDir, "consumed", releaseID)
}

