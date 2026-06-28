package rellog

import (
	"fmt"
	"strings"
)

const (
	releaseHeadingLevel         = 2
	sectionHeadingLevel         = 3
	entrySubsectionHeadingLevel = 4
	emptyReleaseMessage         = "No changelog-worthy changes."
	bodyMarkerStart             = "<!-- rellog:body:start -->"
	bodyMarkerEnd               = "<!-- rellog:body:end -->"
	reservedMarkerPrefix        = "<!-- rellog:"
)

func markdownHeading(level int) string {
	return strings.Repeat("#", level)
}

func renderEntryBlock(sb *strings.Builder, e entry) {
	subheading := markdownHeading(entrySubsectionHeadingLevel)
	fmt.Fprintf(sb, "\n%s Details\n\n", subheading)
	sb.WriteString(bodyMarkerStart)
	sb.WriteString("\n")
	sb.WriteString(e.Body)
	sb.WriteString("\n")
	sb.WriteString(bodyMarkerEnd)
	sb.WriteString("\n")

	if len(e.Targets) > 0 {
		fmt.Fprintf(sb, "\n%s Targets\n\n", subheading)
		for _, target := range e.Targets {
			fmt.Fprintf(sb, "- %s\n", target)
		}
	}

	if len(e.Links) > 0 {
		fmt.Fprintf(sb, "\n%s Links\n\n", subheading)
		for _, link := range uniqueStrings(e.Links) {
			fmt.Fprintf(sb, "- %s\n", link)
		}
	}
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
