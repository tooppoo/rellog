package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type entry struct {
	Kind                 string
	Targets              []string
	Issues               []int
	PRs                  []int
	Body                 string
	targetsKeyPresent    bool
	targetsIsScalar      bool
	issuesIsScalar       bool
	prsHasNonNumericItem bool
}

func formatEntry(e entry) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "kind: %s\n", e.Kind)
	if len(e.Targets) > 0 {
		sb.WriteString("targets:\n")
		for _, t := range e.Targets {
			fmt.Fprintf(&sb, "  - %s\n", t)
		}
	}
	if len(e.Issues) > 0 {
		sb.WriteString("issues:\n")
		for _, i := range e.Issues {
			fmt.Fprintf(&sb, "  - %d\n", i)
		}
	}
	if len(e.PRs) > 0 {
		sb.WriteString("prs:\n")
		for _, p := range e.PRs {
			fmt.Fprintf(&sb, "  - %d\n", p)
		}
	}
	sb.WriteString("---\n")
	sb.WriteString(e.Body)
	sb.WriteString("\n")
	return sb.String()
}

// renderReleaseNote generates markdown release note content for the given version and entries.
func renderReleaseNote(version string, entries []entry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n", version)

	// Group entries by kind, preserving first-seen order.
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

func parseEntry(data []byte) (entry, error) {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return entry{}, fmt.Errorf("invalid frontmatter: missing opening ---")
	}
	rest := s[4:]
	frontmatter, after, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return entry{}, fmt.Errorf("invalid frontmatter: missing closing ---")
	}
	body := strings.TrimRight(after, "\n")

	e := entry{Body: body}
	var currentList string
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.HasPrefix(line, "  - ") {
			item := strings.TrimPrefix(line, "  - ")
			switch currentList {
			case "targets":
				e.Targets = append(e.Targets, item)
			case "issues":
				n, _ := strconv.Atoi(item)
				e.Issues = append(e.Issues, n)
			case "prs":
				n, err := strconv.Atoi(item)
				if err != nil {
					e.prsHasNonNumericItem = true
				} else {
					e.PRs = append(e.PRs, n)
				}
			}
			continue
		}
		currentList = ""
		k, v, hasVal := strings.Cut(line, ": ")
		if hasVal {
			switch k {
			case "kind":
				e.Kind = v
			case "targets":
				e.targetsKeyPresent = true
				e.targetsIsScalar = true
				_ = v
			case "issues":
				e.issuesIsScalar = true
				_ = v
			}
		} else if strings.HasSuffix(line, ":") {
			currentList = strings.TrimSuffix(line, ":")
			switch currentList {
			case "targets":
				e.targetsKeyPresent = true
			}
		}
	}
	return e, nil
}

func readEntries() ([]entry, error) {
	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return nil, err
	}

	var entries []entry
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(entriesDir(), f.Name()))
		if err != nil {
			return nil, err
		}
		e, err := parseEntry(data)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
