package rellog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type entry struct {
	Kind    string
	Targets []string
	Issues  []string
	PRs     []string
	Body    string

	// Parsing diagnostics for validation
	targetsPresent  bool
	targetsIsScalar bool
	issuesPresent   bool
	issuesIsScalar  bool
	prsPresent      bool
	prsIsScalar     bool
}

// entryFilename generates a timestamp-based filename for an entry.
func entryFilename(t time.Time) string {
	utc := t.UTC()
	return fmt.Sprintf("%04d%02d%02dT%02d%02d%02d.%09dZ.json",
		utc.Year(), int(utc.Month()), utc.Day(),
		utc.Hour(), utc.Minute(), utc.Second(),
		utc.Nanosecond())
}

type jsonEntryFormat struct {
	Kind    string   `json:"kind"`
	Targets []string `json:"targets"`
	Issues  []string `json:"issues"`
	PRs     []string `json:"prs"`
	Body    string   `json:"body"`
}

// formatEntryJSON serializes an entry to pretty-printed JSON with trailing newline.
func formatEntryJSON(e entry) []byte {
	je := jsonEntryFormat{
		Kind:    e.Kind,
		Targets: e.Targets,
		Issues:  e.Issues,
		PRs:     e.PRs,
		Body:    e.Body,
	}
	if je.Targets == nil {
		je.Targets = []string{}
	}
	if je.Issues == nil {
		je.Issues = []string{}
	}
	if je.PRs == nil {
		je.PRs = []string{}
	}
	data, _ := json.MarshalIndent(je, "", "  ")
	return append(data, '\n')
}

// parseEntryJSON parses a JSON entry file. Returns the entry and any parse error.
// On parse error, returns a zero entry with a non-nil error.
func parseEntryJSON(data []byte) (entry, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return entry{}, fmt.Errorf("invalid JSON entry")
	}

	var e entry

	if kindRaw, ok := raw["kind"]; ok {
		json.Unmarshal(kindRaw, &e.Kind) //nolint // ignore error, Kind stays ""
	}

	if bodyRaw, ok := raw["body"]; ok {
		json.Unmarshal(bodyRaw, &e.Body) //nolint
	}

	if targetsRaw, ok := raw["targets"]; ok {
		e.targetsPresent = true
		if len(targetsRaw) > 0 && targetsRaw[0] == '[' {
			json.Unmarshal(targetsRaw, &e.Targets) //nolint
		} else {
			e.targetsIsScalar = true
		}
	}

	if issuesRaw, ok := raw["issues"]; ok {
		e.issuesPresent = true
		if len(issuesRaw) > 0 && issuesRaw[0] == '[' {
			json.Unmarshal(issuesRaw, &e.Issues) //nolint
		} else {
			e.issuesIsScalar = true
		}
	}

	if prsRaw, ok := raw["prs"]; ok {
		e.prsPresent = true
		if len(prsRaw) > 0 && prsRaw[0] == '[' {
			json.Unmarshal(prsRaw, &e.PRs) //nolint
		} else {
			e.prsIsScalar = true
		}
	}

	return e, nil
}

// renderReleaseNote generates markdown release note content for the given version and entries.
func renderReleaseNote(version string, entries []entry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n", version)

	var kindOrder []string
	kindEntries := map[string][]string{}
	for _, e := range entries {
		if _, seen := kindEntries[e.Kind]; !seen {
			kindOrder = append(kindOrder, e.Kind)
		}
		kindEntries[e.Kind] = append(kindEntries[e.Kind], e.Body)
	}

	for _, kind := range kindOrder {
		fmt.Fprintf(&sb, "\n### %s\n\n", kind)
		for _, body := range kindEntries[kind] {
			fmt.Fprintf(&sb, "- %s\n", body)
		}
	}
	return sb.String()
}

func readEntries() ([]entry, error) {
	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return nil, err
	}

	var entries []entry
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if err != nil {
			return nil, err
		}
		e, err := parseEntryJSON(data)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
