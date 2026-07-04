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

// unionValues returns seed (already deduped, in order) followed by every
// not-yet-seen value extract(e) yields while walking entries in order. It is
// used to build the single, kind-section-wide Targets/Links list: entries
// contribute values in filename order, and within one entry, in that entry's
// own list order.
func unionValues(seed []string, entries []entry, extract func(entry) []string) []string {
	seen := make(map[string]bool, len(seed))
	out := append([]string{}, seed...)
	for _, v := range seed {
		seen[v] = true
	}
	for _, e := range entries {
		for _, v := range extract(e) {
			if seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// renderKindSectionInner renders the content of one kind section (everything
// after its "### <title>" heading line): a single "#### Details" holding one
// marker-delimited body block per entry, followed by an optional single
// "#### Targets" and an optional single "#### Links".
func renderKindSectionInner(bodies, targets, links []string) string {
	var sb strings.Builder
	subheading := markdownHeading(entrySubsectionHeadingLevel)

	fmt.Fprintf(&sb, "\n%s Details\n\n", subheading)
	for i, body := range bodies {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(bodyMarkerStart)
		sb.WriteString("\n")
		sb.WriteString(body)
		sb.WriteString("\n")
		sb.WriteString(bodyMarkerEnd)
		sb.WriteString("\n")
	}

	if len(targets) > 0 {
		fmt.Fprintf(&sb, "\n%s Targets\n\n", subheading)
		for _, target := range targets {
			fmt.Fprintf(&sb, "- %s\n", target)
		}
	}

	if len(links) > 0 {
		fmt.Fprintf(&sb, "\n%s Links\n\n", subheading)
		for _, link := range links {
			fmt.Fprintf(&sb, "- %s\n", link)
		}
	}

	return sb.String()
}
