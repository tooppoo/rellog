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
	Links   []string
	Body    string

	// Parsing diagnostics for validation
	targetsPresent   bool
	targetsIsScalar  bool
	targetsTypeError bool
	linksPresent     bool
	linksIsScalar    bool
	linksTypeError   bool
	bodyPresent      bool
	bodyTypeError    bool
	unknownFields    []string
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
	Links   []string `json:"links"`
	Body    string   `json:"body"`
}

// formatEntryJSON serializes an entry to pretty-printed JSON with trailing newline.
func formatEntryJSON(e entry) []byte {
	je := jsonEntryFormat{
		Kind:    e.Kind,
		Targets: e.Targets,
		Links:   e.Links,
		Body:    e.Body,
	}
	if je.Targets == nil {
		je.Targets = []string{}
	}
	if je.Links == nil {
		je.Links = []string{}
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
	for key := range raw {
		switch key {
		case "kind", "targets", "links", "body":
		default:
			e.unknownFields = append(e.unknownFields, key)
		}
	}

	if kindRaw, ok := raw["kind"]; ok {
		json.Unmarshal(kindRaw, &e.Kind) //nolint // ignore error, Kind stays ""
	}

	if bodyRaw, ok := raw["body"]; ok {
		e.bodyPresent = true
		// json.Unmarshal silently no-ops when unmarshalling null into a non-pointer
		// string, returning nil error while Body stays "". Guard with a token-type
		// check so that null (and any other non-string value) sets bodyTypeError.
		if len(bodyRaw) > 0 && bodyRaw[0] == '"' {
			if err := json.Unmarshal(bodyRaw, &e.Body); err != nil {
				e.bodyTypeError = true
			}
		} else {
			e.bodyTypeError = true
		}
	}

	if targetsRaw, ok := raw["targets"]; ok {
		e.targetsPresent = true
		if len(targetsRaw) > 0 && targetsRaw[0] == '[' {
			if err := json.Unmarshal(targetsRaw, &e.Targets); err != nil {
				e.targetsTypeError = true
			}
		} else {
			e.targetsIsScalar = true
		}
	}

	if linksRaw, ok := raw["links"]; ok {
		e.linksPresent = true
		if len(linksRaw) > 0 && linksRaw[0] == '[' {
			if err := json.Unmarshal(linksRaw, &e.Links); err != nil {
				e.linksTypeError = true
			}
		} else {
			e.linksIsScalar = true
		}
	}

	return e, nil
}

// renderReleaseNote generates markdown release note content for the given version and entries.
// Entries are grouped under their effective kind title (see kindTitle), in first-seen order.
func renderReleaseNote(version string, entries []entry, cfg entryValidationConfig) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s\n", markdownHeading(releaseHeadingLevel), version)

	var titleOrder []string
	titleEntries := map[string][]entry{}
	for _, e := range entries {
		title := kindTitle(e.Kind, cfg)
		if _, seen := titleEntries[title]; !seen {
			titleOrder = append(titleOrder, title)
		}
		titleEntries[title] = append(titleEntries[title], e)
	}

	for _, title := range titleOrder {
		sb.WriteString(renderKindSection(title, titleEntries[title]))
	}
	return sb.String()
}

// kindTitle returns the effective title for kindID: the configured title if
// present, otherwise the kind id itself (see docs/configuration.md "kind.title").
func kindTitle(kindID string, cfg entryValidationConfig) string {
	if title, ok := cfg.kindTitles[kindID]; ok {
		return title
	}
	return kindID
}

// renderKindSection renders one "### <title>" section with its entries, in the
// same shape renderReleaseNote produces for a single kind. It is also used by
// `amend` to append a brand-new kind section into an existing document, so the
// output must stay byte-identical to what a full regenerate would produce.
func renderKindSection(title string, entries []entry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n%s %s\n", markdownHeading(sectionHeadingLevel), title)
	for i, e := range entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		renderEntryBlock(&sb, e)
	}
	return sb.String()
}

// entryFile pairs a parsed entry with its filename and full path.
type entryFile struct {
	name string
	path string
	e    entry
}

// loadEntryFiles reads and parses every *.json entry file in dir, in
// directory-listing order (which matches filename order for the generated
// timestamp filenames). Non-.json files are ignored.
func loadEntryFiles(dir string) ([]entryFile, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var entryFiles []entryFile
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		p := filepath.Join(dir, f.Name())
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil, readErr
		}
		e, parseErr := parseEntryJSON(data)
		if parseErr != nil {
			return nil, parseErr
		}
		entryFiles = append(entryFiles, entryFile{f.Name(), p, e})
	}
	return entryFiles, nil
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
