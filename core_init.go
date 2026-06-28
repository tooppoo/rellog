package rellog

import (
	"fmt"
	"os"
)

func generateInitConfig() string {
	return `/- kdl-version 2

rellog config-version=1 {
  paths {
    changelog "CHANGELOG.md"
    entries ".rellog/entries"
    release-notes ".rellog/release-notes"
  }

  entries {
    target-policy "allow-unknown"

    kinds {
      kind "added"
      kind "changed"
      kind "fixed"
    }
  }
}
`
}

func initRellog() error {
	if err := os.MkdirAll(entriesDir(), 0755); err != nil {
		return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", entriesDir(), err)}
	}
	if err := os.MkdirAll(releaseNotesDir(), 0755); err != nil {
		return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", releaseNotesDir(), err)}
	}
	// Only create config if it doesn't already exist as a regular file (preserve user's config)
	if info, err := os.Stat(configFile()); err == nil && info.Mode().IsRegular() {
		return nil
	}
	config := generateInitConfig()
	if err := os.WriteFile(configFile(), []byte(config), 0644); err != nil {
		return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", configFile(), err)}
	}
	return nil
}
