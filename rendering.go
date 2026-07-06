package rellog

import (
	"fmt"
	"strings"
)

const (
	releaseHeadingLevel       = 2
	sectionHeadingLevel       = 3
	targetSectionHeadingLevel = 4
	emptyReleaseMessage       = "No changelog-worthy changes."
	bodyMarkerStart           = "<!-- rellog:body:start -->"
	bodyMarkerEnd             = "<!-- rellog:body:end -->"
	entryMarkerStart          = "<!-- rellog:entry:start -->"
	entryMarkerEnd            = "<!-- rellog:entry:end -->"
	refsLabel                 = "Refs:"
	reservedMarkerPrefix      = "<!-- rellog:"
)

func markdownHeading(level int) string {
	return strings.Repeat("#", level)
}

// renderEntryBlock renders one entry, wrapped in entry markers. The body is
// wrapped in body markers; links, when present, render under the plain
// "Refs:" label (not a Markdown heading). The leading newline doubles as the
// blank-line separator after a target heading or a preceding entry block.
func renderEntryBlock(sb *strings.Builder, e entry) {
	sb.WriteString("\n")
	sb.WriteString(entryMarkerStart)
	sb.WriteString("\n")
	sb.WriteString(bodyMarkerStart)
	sb.WriteString("\n")
	sb.WriteString(e.Body)
	sb.WriteString("\n")
	sb.WriteString(bodyMarkerEnd)
	sb.WriteString("\n")

	if len(e.Links) > 0 {
		sb.WriteString("\n")
		sb.WriteString(refsLabel)
		sb.WriteString("\n")
		for _, link := range uniqueStrings(e.Links) {
			fmt.Fprintf(sb, "- %s\n", link)
		}
	}

	sb.WriteString(entryMarkerEnd)
	sb.WriteString("\n")
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
