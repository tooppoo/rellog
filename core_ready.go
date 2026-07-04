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

type changelogHeadingScan struct {
	FoundOutsideBody bool
	FoundInsideBody  bool
	InvalidStructure string
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
		scan := scanChangelogReleaseHeading(string(changelogData), releaseID)
		if scan.InvalidStructure != "" {
			result.Errors = append(result.Errors, readyCheckError{
				Code:    "changelog_invalid_structure",
				Message: scan.InvalidStructure,
			})
		} else if !scan.FoundOutsideBody {
			message := "Changelog is missing release heading: " + releaseID
			if scan.FoundInsideBody {
				var sb strings.Builder
				sb.WriteString("release is not ready: changelog is missing release heading: ")
				sb.WriteString(releaseID)
				sb.WriteString("\n")
				sb.WriteString("\n")
				sb.WriteString("release: ")
				sb.WriteString(releaseID)
				sb.WriteString("\n")
				sb.WriteString("changelog:\n")
				sb.WriteString("  ")
				sb.WriteString(paths.changelogPath)
				sb.WriteString("\n")
				sb.WriteString("\n")
				sb.WriteString("The changelog does not contain a structural release heading for ")
				sb.WriteString(releaseID)
				sb.WriteString(".\n")
				sb.WriteString("\n")
				sb.WriteString("rellog looked for this heading outside generated body marker comments:\n")
				sb.WriteString("  ")
				sb.WriteString(markdownHeading(releaseHeadingLevel))
				sb.WriteString(" ")
				sb.WriteString(releaseID)
				sb.WriteString("\n")
				sb.WriteString("\n")
				sb.WriteString("A matching heading was found inside a generated entry body:\n")
				sb.WriteString("  " + bodyMarkerStart + "\n")
				sb.WriteString("  ")
				sb.WriteString(markdownHeading(releaseHeadingLevel))
				sb.WriteString(" ")
				sb.WriteString(releaseID)
				sb.WriteString("\n")
				sb.WriteString("  " + bodyMarkerEnd + "\n")
				sb.WriteString("\n")
				sb.WriteString("Headings inside entry bodies are release note content, not release boundaries.\n")
				sb.WriteString("Run `rellog prepare ")
				sb.WriteString(releaseID)
				sb.WriteString(" --run` or add the release section to ")
				sb.WriteString(paths.changelogPath)
				sb.WriteString(" outside the body marker range.")
				message = sb.String()
			}
			result.Errors = append(result.Errors, readyCheckError{
				Code:    "changelog_heading_missing",
				Message: message,
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
		case "changelog_invalid_structure":
			var sb strings.Builder
			sb.WriteString("invalid generated Markdown structure in ")
			sb.WriteString(result.Changelog)
			sb.WriteString("\n")
			sb.WriteString("\n")
			sb.WriteString("release: ")
			sb.WriteString(result.ReleaseID)
			sb.WriteString("\n")
			sb.WriteString("changelog:\n")
			sb.WriteString("  ")
			sb.WriteString(result.Changelog)
			sb.WriteString("\n")
			sb.WriteString("\n")
			sb.WriteString(e.Message)
			return &exitError{ExitReleaseNotReady, sb.String()}
		case "changelog_heading_missing":
			if strings.HasPrefix(e.Message, "release is not ready:") {
				return &exitError{ExitReleaseNotReady, e.Message}
			}
			var sb strings.Builder
			sb.WriteString("release is not ready: changelog is missing release heading\n")
			sb.WriteString("\n")
			sb.WriteString("release: ")
			sb.WriteString(result.ReleaseID)
			sb.WriteString("\n")
			sb.WriteString("expected heading:\n")
			sb.WriteString("  ")
			sb.WriteString(markdownHeading(releaseHeadingLevel))
			sb.WriteString(" ")
			sb.WriteString(result.ReleaseID)
			sb.WriteString("\n")
			sb.WriteString("changelog:\n")
			sb.WriteString("  ")
			sb.WriteString(result.Changelog)
			sb.WriteString("\n")
			sb.WriteString("\n")
			sb.WriteString(e.Message)
			sb.WriteString("\n")
			sb.WriteString("Add the release heading outside generated body marker ranges, then run `rellog ready ")
			sb.WriteString(result.ReleaseID)
			sb.WriteString("` again.")
			return &exitError{ExitReleaseNotReady, sb.String()}
		}
	}

	if len(result.PendingEntries) > 0 {
		var sb strings.Builder
		sb.WriteString("release is not ready: pending entries remain\n")
		sb.WriteString("\n")
		sb.WriteString("release: ")
		sb.WriteString(result.ReleaseID)
		sb.WriteString("\n")
		sb.WriteString("release note:\n")
		sb.WriteString("  ")
		sb.WriteString(result.ReleaseNote)
		sb.WriteString("\n")
		sb.WriteString("\n")
		sb.WriteString("Pending entries:\n")
		for _, p := range result.PendingEntries {
			sb.WriteString("  ")
			sb.WriteString(p)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString("A release note already exists, but pending entries are still present.\n")
		sb.WriteString("Decide what each pending entry means:\n")
		sb.WriteString("\n")
		sb.WriteString("  1. If the entry was created by mistake, remove it.\n")
		sb.WriteString("  2. If the entry should be included in this release, run:\n")
		sb.WriteString("       rellog amend ")
		sb.WriteString(result.ReleaseID)
		sb.WriteString("\n")
		sb.WriteString("     Review the dry-run output, then run:\n")
		sb.WriteString("       rellog amend ")
		sb.WriteString(result.ReleaseID)
		sb.WriteString(" --run\n")
		sb.WriteString("  3. If the entry is for a future release, move it out of the pending entries directory\n")
		sb.WriteString("     and restore it after this release is completed.")
		return &exitError{ExitReleaseNotReady, sb.String()}
	}

	return &exitError{ExitReleaseNotReady, "release is not ready"}
}

func scanChangelogReleaseHeading(content, releaseID string) changelogHeadingScan {
	heading := markdownHeading(releaseHeadingLevel) + " " + releaseID
	var scan changelogHeadingScan
	inBody := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimRight(line, "\r")
		switch trimmed {
		case bodyMarkerStart:
			if inBody {
				scan.InvalidStructure = malformedBodyMarkerMessage(bodyMarkerStart, bodyMarkerEnd, releaseID)
				return scan
			}
			inBody = true
			continue
		case bodyMarkerEnd:
			if !inBody {
				scan.InvalidStructure = malformedBodyMarkerMessage(bodyMarkerEnd, bodyMarkerStart, releaseID)
				return scan
			}
			inBody = false
			continue
		}

		if trimmed == heading {
			if inBody {
				scan.FoundInsideBody = true
			} else {
				scan.FoundOutsideBody = true
			}
		}
	}

	if inBody {
		scan.InvalidStructure = malformedBodyMarkerMessage(bodyMarkerStart, bodyMarkerEnd, releaseID)
	}
	return scan
}

func malformedBodyMarkerMessage(found, missing, releaseID string) string {
	var sb strings.Builder
	if found == bodyMarkerStart && missing == bodyMarkerEnd {
		sb.WriteString("The changelog contains an unterminated rellog body marker range.\n")
	} else {
		sb.WriteString("The changelog contains a malformed rellog body marker range.\n")
	}
	sb.WriteString("\n")
	sb.WriteString("Found:\n")
	sb.WriteString("  ")
	sb.WriteString(found)
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString("Missing matching marker:\n")
	sb.WriteString("  ")
	sb.WriteString(missing)
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString("rellog cannot safely identify release headings while a body marker pair is malformed.\n")
	sb.WriteString("Fix the generated body marker pair in CHANGELOG.md, then run `rellog ready ")
	sb.WriteString(releaseID)
	sb.WriteString("` again.")
	return sb.String()
}
