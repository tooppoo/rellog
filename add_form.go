package rellog

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type addFormField int

const (
	fieldKind addFormField = iota
	fieldTargets
	fieldBody
	fieldLinks
	fieldCount
)

var addFormFieldLabels = [fieldCount]string{
	fieldKind:    "kind",
	fieldTargets: "targets",
	fieldBody:    "body",
	fieldLinks:   "links",
}

// addFormModel is the Bubble Tea model for the TTY-only rich add form.
// It only collects values; validation and file writing stay in addEntry.
type addFormModel struct {
	kind    textinput.Model
	targets textinput.Model
	body    textarea.Model
	links   textinput.Model

	kindCandidates   []string
	targetCandidates []string
	targetPolicy     string

	focus     addFormField
	listOpen  bool
	listIndex int
	helpOpen  bool

	submitted bool
	cancelled bool
}

func newAddFormModel(cfg entryValidationConfig) addFormModel {
	newInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = placeholder
		return ti
	}

	body := textarea.New()
	body.Placeholder = "Change description"
	body.ShowLineNumbers = false
	body.SetWidth(60)
	body.SetHeight(4)

	m := addFormModel{
		kind:             newInput("e.g. changed"),
		targets:          newInput("space or comma separated"),
		body:             body,
		links:            newInput("absolute http(s) URLs, space or comma separated"),
		kindCandidates:   slices.Sorted(maps.Keys(cfg.allowedKinds)),
		targetCandidates: slices.Sorted(maps.Keys(cfg.knownTargets)),
		targetPolicy:     cfg.targetPolicy,
	}
	m.kind.Focus()
	return m
}

func (m addFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m addFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		// Swallow pastes while the candidate list is open, matching how
		// typed keys are swallowed below.
		if _, isPaste := msg.(tea.PasteMsg); isPaste && m.listOpen {
			return m, nil
		}
		return m.updateFocused(msg)
	}

	switch keyMsg.String() {
	case "ctrl+c":
		m.cancelled = true
		return m, tea.Quit
	case "ctrl+d", "ctrl+enter":
		m.submitted = true
		return m, tea.Quit
	case "ctrl+g":
		m.helpOpen = !m.helpOpen
		return m, nil
	case "ctrl+l":
		if len(m.candidatesForFocus()) > 0 {
			m.listOpen = !m.listOpen
			m.listIndex = 0
		}
		return m, nil
	case "esc":
		if m.helpOpen {
			m.helpOpen = false
			return m, nil
		}
		m.listOpen = false
		return m, nil
	case "tab":
		if m.listOpen {
			m.listIndex = (m.listIndex + 1) % len(m.candidatesForFocus())
			return m, nil
		}
		return m, m.setFocus((m.focus + 1) % fieldCount)
	case "shift+tab":
		return m, m.setFocus((m.focus + fieldCount - 1) % fieldCount)
	case "enter":
		if m.listOpen {
			m.adoptCandidate()
			return m, nil
		}
		// Fall through to the focused widget: inserts a newline in the
		// body textarea, no-op on single-line inputs.
	case "alt+1", "ctrl+1":
		return m, m.setFocus(fieldKind)
	case "alt+2", "ctrl+2":
		return m, m.setFocus(fieldTargets)
	case "alt+3", "ctrl+3":
		return m, m.setFocus(fieldBody)
	case "alt+4", "ctrl+4":
		return m, m.setFocus(fieldLinks)
	}

	if m.listOpen {
		return m, nil
	}
	return m.updateFocused(msg)
}

// setFocus moves focus to f, closing any open candidate list.
func (m *addFormModel) setFocus(f addFormField) tea.Cmd {
	m.focus = f
	m.listOpen = false
	m.kind.Blur()
	m.targets.Blur()
	m.body.Blur()
	m.links.Blur()
	switch f {
	case fieldKind:
		return m.kind.Focus()
	case fieldTargets:
		return m.targets.Focus()
	case fieldBody:
		return m.body.Focus()
	default:
		return m.links.Focus()
	}
}

// candidatesForFocus returns the candidate list for the focused field.
// Only kind and targets have candidates.
func (m addFormModel) candidatesForFocus() []string {
	switch m.focus {
	case fieldKind:
		return m.kindCandidates
	case fieldTargets:
		return m.targetCandidates
	default:
		return nil
	}
}

// adoptCandidate applies the highlighted candidate to the focused field:
// kind is replaced outright, targets get the candidate appended as a token.
func (m *addFormModel) adoptCandidate() {
	candidates := m.candidatesForFocus()
	if len(candidates) == 0 {
		return
	}
	candidate := candidates[m.listIndex]
	switch m.focus {
	case fieldKind:
		m.kind.SetValue(candidate)
		m.kind.CursorEnd()
	case fieldTargets:
		existing := strings.TrimSpace(m.targets.Value())
		if existing == "" {
			m.targets.SetValue(candidate)
		} else {
			m.targets.SetValue(existing + " " + candidate)
		}
		m.targets.CursorEnd()
	}
	m.listOpen = false
}

func (m addFormModel) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case fieldKind:
		m.kind, cmd = m.kind.Update(msg)
	case fieldTargets:
		m.targets, cmd = m.targets.Update(msg)
	case fieldBody:
		m.body, cmd = m.body.Update(msg)
	default:
		m.links, cmd = m.links.Update(msg)
	}
	return m, cmd
}

// result converts the collected form values into addOptions. The bool is
// true only when the user submitted the form (not on cancel).
func (m addFormModel) result(debugDatetime string) (addOptions, bool) {
	if !m.submitted {
		return addOptions{}, false
	}
	return addOptions{
		Kind:          strings.TrimSpace(m.kind.Value()),
		Targets:       splitTokens(m.targets.Value()),
		Body:          strings.TrimRight(m.body.Value(), "\n"),
		Links:         splitTokens(m.links.Value()),
		DebugDatetime: debugDatetime,
	}, true
}

type addFormStyles struct {
	title     lipgloss.Style
	focus     lipgloss.Style
	hint      lipgloss.Style
	candidate lipgloss.Style
}

// addFormStylesOnce builds the lipgloss styles lazily: creating styles makes
// lipgloss probe the terminal (writing an OSC background-color query to
// stdout), which must only ever happen while the TUI form is running, never
// as a side effect of importing this package in unrelated commands.
var addFormStylesOnce = sync.OnceValue(func() addFormStyles {
	return addFormStyles{
		title:     lipgloss.NewStyle().Bold(true),
		focus:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
		hint:      lipgloss.NewStyle().Faint(true),
		candidate: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
	}
})

// View renders the form for Bubble Tea. The string assembly lives in
// viewString so tests can assert on plain text.
func (m addFormModel) View() tea.View {
	return tea.NewView(m.viewString())
}

func (m addFormModel) viewString() string {
	styles := addFormStylesOnce()

	var sb strings.Builder
	sb.WriteString(styles.title.Render("rellog add — new changelog entry"))
	sb.WriteString("\n\n")

	label := func(f addFormField) string {
		text := fmt.Sprintf("[%d] %-7s", int(f)+1, addFormFieldLabels[f])
		if f == m.focus {
			return styles.focus.Render("▸ " + text)
		}
		return "  " + text
	}

	sb.WriteString(label(fieldKind) + " " + m.kind.View() + "\n")
	sb.WriteString(label(fieldTargets) + " " + m.targets.View() + "\n")
	sb.WriteString(label(fieldBody) + "\n")
	sb.WriteString(m.body.View() + "\n")
	sb.WriteString(label(fieldLinks) + " " + m.links.View() + "\n")

	if m.listOpen {
		sb.WriteString("\n" + styles.hint.Render("candidates (tab: next, enter: select, esc: close)") + "\n")
		for i, c := range m.candidatesForFocus() {
			if i == m.listIndex {
				sb.WriteString(styles.candidate.Render("▸ "+c) + "\n")
			} else {
				sb.WriteString("  " + c + "\n")
			}
		}
		if m.focus == fieldTargets {
			sb.WriteString(styles.hint.Render("target-policy: "+m.targetPolicy) + "\n")
		}
	}

	if m.helpOpen {
		sb.WriteString("\n" + styles.hint.Render(strings.Join([]string{
			"tab / shift+tab   next / previous field (tab cycles an open candidate list)",
			"enter             select candidate; newline while editing body",
			"ctrl+l            toggle candidate list (kind / targets)",
			"ctrl+g            toggle this help",
			"alt+1 … alt+4     jump to field (ctrl+1 … ctrl+4 in capable terminals)",
			"ctrl+d            submit (ctrl+enter in capable terminals)",
			"ctrl+c            cancel without saving",
		}, "\n")) + "\n")
	}

	sb.WriteString("\n" + styles.hint.Render("tab: next field • ctrl+l: candidates • ctrl+g: help • ctrl+d: submit • ctrl+c: cancel") + "\n")
	return sb.String()
}
