package rellog

import (
	"fmt"
	"os"
)

func generateInitConfig(githubURL string) (string, error) {
	if githubURL == "" {
		return "", fmt.Errorf("githubURL must not be empty")
	}
	return fmt.Sprintf(`/- kdl-version 2

rellog config-version=1 {
  github-url "%s"
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
`, githubURL), nil
}

func initRellog() error {
	githubURL, err := detectGitHubURL()
	if err != nil || githubURL == "" {
		return &exitError{ExitNotGitRepo, "rellog init must be run inside a git repository"}
	}
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
	config, err := generateInitConfig(githubURL)
	if err != nil {
		return &exitError{ExitNotGitRepo, err.Error()}
	}
	if err := os.WriteFile(configFile(), []byte(config), 0644); err != nil {
		return &exitError{ExitInvalidStructure, fmt.Sprintf("failed to create %s: %s", configFile(), err)}
	}
	return nil
}
