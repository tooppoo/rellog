package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kdl "github.com/njreid/gokdl2"
)

type readyPaths struct {
	changelogPath   string
	entriesDir      string
	releaseNotesDir string
}

type readyCheckError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type readyResult struct {
	OK             bool              `json:"ok"`
	ReleaseID      string            `json:"releaseId"`
	ReleaseNote    string            `json:"releaseNote"`
	Changelog      string            `json:"changelog"`
	PendingEntries []string          `json:"pendingEntries"`
	Errors         []readyCheckError `json:"errors"`
}

// validateReadyReleaseID rejects release IDs containing dot-only path segments
// (. or ..) or empty segments from leading/trailing slashes.
// Path separators are otherwise allowed, e.g. "cli/v1.0.0" is valid.
func validateReadyReleaseID(id string) error {
	if id == "" {
		return &exitError{ExitInvalidArgument, fmt.Sprintf("invalid release id: %q", id)}
	}
	for _, seg := range strings.Split(id, "/") {
		if seg == "." || seg == ".." || seg == "" {
			return &exitError{ExitInvalidArgument, fmt.Sprintf("invalid release id: %q", id)}
		}
	}
	return nil
}

func readReadyPaths() (readyPaths, error) {
	data, err := os.ReadFile(configFile())
	if err != nil {
		return readyPaths{}, err
	}
	doc, err := kdl.Parse(strings.NewReader(string(data)))
	if err != nil {
		return readyPaths{}, &exitError{ExitCheckFailed, "failed to parse config: " + err.Error()}
	}

	paths := readyPaths{
		changelogPath:   "CHANGELOG.md",
		entriesDir:      entriesDir(),
		releaseNotesDir: releaseNotesDir(),
	}

	for _, n := range doc.Nodes {
		if nodeName(n) != "rellog" {
			continue
		}
		for _, child := range n.Children {
			if nodeName(child) != "paths" {
				continue
			}
			for _, pathNode := range child.Children {
				if len(pathNode.Arguments) == 0 {
					continue
				}
				val := pathNode.Arguments[0].ValueString()
				switch nodeName(pathNode) {
				case "changelog":
					paths.changelogPath = val
				case "entries":
					paths.entriesDir = val
				case "release-notes":
					paths.releaseNotesDir = val
				}
			}
		}
		break
	}

	return paths, nil
}

func checkReady(releaseID string) (readyResult, error) {
	if _, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			return readyResult{}, &exitError{ExitNotInitialized, "run `rellog init` first"}
		}
		return readyResult{}, err
	}

	paths, err := readReadyPaths()
	if err != nil {
		if os.IsNotExist(err) {
			return readyResult{}, &exitError{ExitNotInitialized, "run `rellog init` first"}
		}
		return readyResult{}, err
	}

	releaseNotePath := filepath.Join(paths.releaseNotesDir, releaseID+".md")

	result := readyResult{
		ReleaseID:      releaseID,
		ReleaseNote:    releaseNotePath,
		Changelog:      paths.changelogPath,
		PendingEntries: []string{},
		Errors:         []readyCheckError{},
	}

	if _, err := os.Stat(releaseNotePath); err != nil {
		if os.IsNotExist(err) {
			return result, &exitError{ExitReleaseNotFound, "release not found: " + releaseID}
		}
		return result, err
	}

	entryFiles, err := os.ReadDir(paths.entriesDir)
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}
	if err == nil {
		for _, f := range entryFiles {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
				result.PendingEntries = append(result.PendingEntries, filepath.Join(paths.entriesDir, f.Name()))
			}
		}
	}

	changelogData, readErr := os.ReadFile(paths.changelogPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			result.Errors = append(result.Errors, readyCheckError{
				Code:    "changelog_missing",
				Message: "Changelog file does not exist.",
			})
		} else {
			return result, readErr
		}
	} else {
		heading := "## " + releaseID
		found := false
		for _, line := range strings.Split(string(changelogData), "\n") {
			if strings.TrimRight(line, "\r") == heading {
				found = true
				break
			}
		}
		if !found {
			result.Errors = append(result.Errors, readyCheckError{
				Code:    "changelog_heading_missing",
				Message: "Changelog is missing release heading: " + releaseID,
			})
		}
	}

	if len(result.PendingEntries) > 0 {
		result.Errors = append(result.Errors, readyCheckError{
			Code:    "pending_entries_present",
			Message: "Pending entries remain after the release note was prepared.",
		})
	}

	result.OK = len(result.Errors) == 0
	return result, nil
}

func buildReadyHumanError(result readyResult) error {
	for _, e := range result.Errors {
		switch e.Code {
		case "changelog_missing":
			return &exitError{ExitReleaseNotReady, "release is not ready: changelog is missing"}
		case "changelog_heading_missing":
			return &exitError{ExitReleaseNotReady, "release is not ready: changelog is missing release heading: " + result.ReleaseID}
		}
	}

	if len(result.PendingEntries) > 0 {
		var sb strings.Builder
		sb.WriteString("release is not ready: pending entries remain\n")
		sb.WriteString("\n")
		sb.WriteString("release: " + result.ReleaseID + "\n")
		sb.WriteString("release note:\n")
		sb.WriteString("  " + result.ReleaseNote + "\n")
		sb.WriteString("\n")
		sb.WriteString("Pending entries:\n")
		for _, p := range result.PendingEntries {
			sb.WriteString("  " + p + "\n")
		}
		sb.WriteString("\n")
		sb.WriteString("A release note already exists, but pending entries are still present.\n")
		sb.WriteString("Decide what each pending entry means:\n")
		sb.WriteString("\n")
		sb.WriteString("  1. If the entry was created by mistake, remove it.\n")
		sb.WriteString("  2. If the entry should be included in this release, run:\n")
		sb.WriteString("       rellog replace " + result.ReleaseID + "\n")
		sb.WriteString("     Review the dry-run output, then run:\n")
		sb.WriteString("       rellog replace " + result.ReleaseID + " --run\n")
		sb.WriteString("  3. If the entry is for a future release, move it out of the pending entries directory\n")
		sb.WriteString("     and restore it after this release is completed.")
		return &exitError{ExitReleaseNotReady, sb.String()}
	}

	return &exitError{ExitReleaseNotReady, "release is not ready"}
}
