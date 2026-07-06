package rellog

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testFormConfig() entryValidationConfig {
	return entryValidationConfig{
		allowedKinds: map[string]bool{"added": true, "changed": true, "fixed": true},
		kindTitles:   map[string]string{},
		knownTargets: map[string]bool{"api": true, "cli": true},
		targetOrder:  []string{"api", "cli"},
		targetTitles: map[string]string{},
	}
}

func press(t *testing.T, m addFormModel, msg tea.Msg) addFormModel {
	t.Helper()
	next, _ := m.Update(msg)
	fm, ok := next.(addFormModel)
	if !ok {
		t.Fatalf("Update returned %T, want addFormModel", next)
	}
	return fm
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+d":
		return tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	case "ctrl+g":
		return tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}
	case "ctrl+l":
		return tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}
	default:
		panic("unknown key: " + s)
	}
}

func altDigit(d rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: d, Mod: tea.ModAlt}
}

func typeText(t *testing.T, m addFormModel, s string) addFormModel {
	t.Helper()
	for _, r := range s {
		m = press(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestAddFormFieldNavigation(t *testing.T) {
	m := newAddFormModel(testFormConfig())
	if m.focus != fieldKind {
		t.Fatalf("initial focus = %v, want fieldKind", m.focus)
	}

	// Tab cycles forward and wraps around.
	order := []addFormField{fieldTargets, fieldBody, fieldLinks, fieldKind}
	for _, want := range order {
		m = press(t, m, key("tab"))
		if m.focus != want {
			t.Fatalf("after tab, focus = %v, want %v", m.focus, want)
		}
	}

	// Shift+Tab cycles backward and wraps around.
	back := []addFormField{fieldLinks, fieldBody, fieldTargets, fieldKind}
	for _, want := range back {
		m = press(t, m, key("shift+tab"))
		if m.focus != want {
			t.Fatalf("after shift+tab, focus = %v, want %v", m.focus, want)
		}
	}
}

func TestAddFormAltNumberFieldJumps(t *testing.T) {
	tests := []struct {
		digit rune
		want  addFormField
	}{
		{'1', fieldKind},
		{'2', fieldTargets},
		{'3', fieldBody},
		{'4', fieldLinks},
	}
	for _, tt := range tests {
		m := newAddFormModel(testFormConfig())
		m = press(t, m, altDigit(tt.digit))
		if m.focus != tt.want {
			t.Errorf("alt+%c: focus = %v, want %v", tt.digit, m.focus, tt.want)
		}
	}
}

func TestAddFormContextSensitiveTab(t *testing.T) {
	m := newAddFormModel(testFormConfig())

	// Open the kind candidate list: tab cycles candidates, not fields.
	m = press(t, m, key("ctrl+l"))
	if !m.listOpen {
		t.Fatal("ctrl+l should open the candidate list on kind")
	}
	if m.listIndex != 0 {
		t.Fatalf("listIndex = %d, want 0", m.listIndex)
	}

	m = press(t, m, key("tab"))
	if m.focus != fieldKind {
		t.Fatalf("tab with open list moved focus to %v, want to stay on fieldKind", m.focus)
	}
	if m.listIndex != 1 {
		t.Fatalf("tab with open list: listIndex = %d, want 1", m.listIndex)
	}

	// Cycling wraps around: candidates are [added changed fixed].
	m = press(t, m, key("tab"))
	m = press(t, m, key("tab"))
	if m.listIndex != 0 {
		t.Fatalf("tab cycling should wrap: listIndex = %d, want 0", m.listIndex)
	}

	// Closing the list restores field navigation.
	m = press(t, m, key("esc"))
	if m.listOpen {
		t.Fatal("esc should close the candidate list")
	}
	m = press(t, m, key("tab"))
	if m.focus != fieldTargets {
		t.Fatalf("tab after closing list: focus = %v, want fieldTargets", m.focus)
	}
}

func TestAddFormHelpAndListToggles(t *testing.T) {
	m := newAddFormModel(testFormConfig())

	// ctrl+g toggles help.
	m = press(t, m, key("ctrl+g"))
	if !m.helpOpen {
		t.Fatal("ctrl+g should open help")
	}
	m = press(t, m, key("ctrl+g"))
	if m.helpOpen {
		t.Fatal("ctrl+g should close help")
	}

	// ctrl+l toggles the candidate list on fields with candidates.
	m = press(t, m, key("ctrl+l"))
	if !m.listOpen {
		t.Fatal("ctrl+l should open the list on kind")
	}
	m = press(t, m, key("ctrl+l"))
	if m.listOpen {
		t.Fatal("ctrl+l should close the list")
	}

	// ctrl+l is a no-op on fields without candidates (body, links).
	m = press(t, m, altDigit('3'))
	m = press(t, m, key("ctrl+l"))
	if m.listOpen {
		t.Fatal("ctrl+l on body should not open a list")
	}
	m = press(t, m, altDigit('4'))
	m = press(t, m, key("ctrl+l"))
	if m.listOpen {
		t.Fatal("ctrl+l on links should not open a list")
	}

	// esc closes help before the list.
	m = press(t, m, altDigit('1'))
	m = press(t, m, key("ctrl+l"))
	m = press(t, m, key("ctrl+g"))
	m = press(t, m, key("esc"))
	if m.helpOpen {
		t.Fatal("esc should close help first")
	}
	if !m.listOpen {
		t.Fatal("esc should leave the list open while closing help")
	}
	m = press(t, m, key("esc"))
	if m.listOpen {
		t.Fatal("second esc should close the list")
	}

	// Moving focus closes an open list.
	m = press(t, m, key("ctrl+l"))
	m = press(t, m, key("shift+tab"))
	if m.listOpen {
		t.Fatal("shift+tab should close the open list")
	}
}

func TestAddFormCandidateSelection(t *testing.T) {
	m := newAddFormModel(testFormConfig())

	// kind: enter adopts the highlighted candidate, replacing the value.
	m = typeText(t, m, "ch")
	m = press(t, m, key("ctrl+l"))
	m = press(t, m, key("tab")) // highlight "changed"
	m = press(t, m, key("enter"))
	if got := m.kind.Value(); got != "changed" {
		t.Errorf("kind value = %q, want %q", got, "changed")
	}
	if m.listOpen {
		t.Error("adopting a candidate should close the list")
	}

	// targets: adoption appends tokens instead of replacing.
	m = press(t, m, altDigit('2'))
	m = press(t, m, key("ctrl+l"))
	m = press(t, m, key("enter")) // adopt "api"
	m = press(t, m, key("ctrl+l"))
	m = press(t, m, key("tab")) // highlight "cli"
	m = press(t, m, key("enter"))
	if got := m.targets.Value(); got != "api cli" {
		t.Errorf("targets value = %q, want %q", got, "api cli")
	}
}

func TestAddFormListSwallowsOtherKeys(t *testing.T) {
	m := newAddFormModel(testFormConfig())
	m = press(t, m, key("ctrl+l"))
	m = typeText(t, m, "x")
	if got := m.kind.Value(); got != "" {
		t.Errorf("typing with an open list should be swallowed, kind value = %q", got)
	}
}

func TestAddFormBodyNewline(t *testing.T) {
	m := newAddFormModel(testFormConfig())
	m = press(t, m, altDigit('3'))
	m = typeText(t, m, "line1")
	m = press(t, m, key("enter"))
	m = typeText(t, m, "line2")
	if got := m.body.Value(); got != "line1\nline2" {
		t.Errorf("body value = %q, want %q", got, "line1\nline2")
	}

	// Enter on a single-line field must not insert anything.
	m = press(t, m, altDigit('1'))
	m = typeText(t, m, "fixed")
	m = press(t, m, key("enter"))
	if got := m.kind.Value(); got != "fixed" {
		t.Errorf("kind value after enter = %q, want %q", got, "fixed")
	}
}

func TestAddFormSubmitAndCancel(t *testing.T) {
	t.Run("ctrl+d submits", func(t *testing.T) {
		m := newAddFormModel(testFormConfig())
		m = typeText(t, m, "changed")
		next, cmd := m.Update(key("ctrl+d"))
		m = next.(addFormModel)
		if !m.submitted {
			t.Error("ctrl+d should mark the form submitted")
		}
		if cmd == nil {
			t.Error("ctrl+d should quit the program")
		}
		opts, ok := m.result("20260101T000000.000000000Z")
		if !ok {
			t.Fatal("result should report submitted")
		}
		if opts.Kind != "changed" {
			t.Errorf("Kind = %q, want %q", opts.Kind, "changed")
		}
		if opts.DebugDatetime != "20260101T000000.000000000Z" {
			t.Errorf("DebugDatetime = %q", opts.DebugDatetime)
		}
	})

	t.Run("ctrl+c cancels", func(t *testing.T) {
		m := newAddFormModel(testFormConfig())
		m = typeText(t, m, "changed")
		next, cmd := m.Update(key("ctrl+c"))
		m = next.(addFormModel)
		if m.submitted {
			t.Error("ctrl+c must not mark the form submitted")
		}
		if !m.cancelled {
			t.Error("ctrl+c should mark the form cancelled")
		}
		if cmd == nil {
			t.Error("ctrl+c should quit the program")
		}
		if _, ok := m.result(""); ok {
			t.Error("result must report not-submitted after cancel")
		}
	})
}

func TestAddFormView(t *testing.T) {
	m := newAddFormModel(testFormConfig())
	if m.Init() == nil {
		t.Error("Init should return the cursor blink command")
	}

	view := m.viewString()
	for _, want := range []string{"kind", "targets", "body", "links", "ctrl+d: submit"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q", want)
		}
	}
	if strings.Contains(view, "candidates (") {
		t.Error("View() should not render candidates while the list is closed")
	}

	// Candidate list rendering on the targets field.
	m = press(t, m, altDigit('2'))
	m = press(t, m, key("ctrl+l"))
	view = m.viewString()
	for _, want := range []string{"candidates (", "api", "cli"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() with open targets list missing %q", want)
		}
	}

	// Help panel rendering.
	m = press(t, m, key("ctrl+g"))
	if view = m.viewString(); !strings.Contains(view, "cancel without saving") {
		t.Error("View() with help open should render the key binding help")
	}
}

func TestAddFormResultMultiTokenParsing(t *testing.T) {
	m := newAddFormModel(testFormConfig())

	m = typeText(t, m, "changed")
	m = press(t, m, altDigit('2'))
	m = typeText(t, m, "t1 t2,t3")
	m = press(t, m, altDigit('3'))
	m = typeText(t, m, "Body text")
	m = press(t, m, key("enter"))
	m = press(t, m, altDigit('4'))
	m = typeText(t, m, "https://example.com/a,https://example.com/b https://example.com/c")
	m = press(t, m, key("ctrl+d"))

	opts, ok := m.result("")
	if !ok {
		t.Fatal("result should report submitted")
	}
	if opts.Kind != "changed" {
		t.Errorf("Kind = %q, want %q", opts.Kind, "changed")
	}
	if want := []string{"t1", "t2", "t3"}; !reflect.DeepEqual(opts.Targets, want) {
		t.Errorf("Targets = %v, want %v", opts.Targets, want)
	}
	if opts.Body != "Body text" {
		t.Errorf("Body = %q, want %q (trailing newline trimmed)", opts.Body, "Body text")
	}
	want := []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"}
	if !reflect.DeepEqual(opts.Links, want) {
		t.Errorf("Links = %v, want %v", opts.Links, want)
	}
}
