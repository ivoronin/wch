package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/session"
)

// helpSections must have no empty fields, no duplicate keys within a section, and a
// non-empty section list. Cheap structural invariants that catch typos at PR time.
func TestHelpSectionsValid(t *testing.T) {
	sections := helpSections()
	if len(sections) == 0 {
		t.Fatalf("helpSections() returned no sections")
	}
	for _, s := range sections {
		if s.title == "" {
			t.Errorf("section with empty title: %+v", s)
		}
		if len(s.bindings) == 0 {
			t.Errorf("section %q has no bindings", s.title)
		}
		seen := map[string]bool{}
		for _, b := range s.bindings {
			if b.keys == "" {
				t.Errorf("section %q: empty keys in %+v", s.title, b)
			}
			if b.desc == "" {
				t.Errorf("section %q: empty desc for keys %q", s.title, b.keys)
			}
			if seen[b.keys] {
				t.Errorf("section %q: duplicate keys %q", s.title, b.keys)
			}
			seen[b.keys] = true
		}
	}
}

// renderHelpPanel must fit a typical 80×24 terminal. Tightest reasonable bound: width ≤ 72
// (leaves 4-cell margin per side), height ≤ 24 (fits exactly on a 24-row terminal).
func TestRenderHelpPanelFitsTypicalTerminal(t *testing.T) {
	panel := renderHelpPanel()
	w := lipgloss.Width(panel)
	h := lipgloss.Height(panel)
	if w > 72 {
		t.Errorf("panel width = %d, want <= 72 (leaves margin in 80-col terminal)", w)
	}
	if h > 24 {
		t.Errorf("panel height = %d, want <= 24 (fits 24-row terminal)", h)
	}
}

func TestRenderHelpPanelContainsAllSections(t *testing.T) {
	plain := ansi.Strip(renderHelpPanel())
	for _, s := range helpSections() {
		if !strings.Contains(plain, s.title) {
			t.Errorf("rendered panel missing section %q", s.title)
		}
	}
}

// Keys should carry SGR Bold (attribute 1). Lipgloss may compose it with fg/bg into a
// multi-attribute sequence like "\x1b[1;37;40m", so check for the bold attribute embedded
// in any of the standard SGR shapes.
func TestRenderHelpPanelBoldKeys(t *testing.T) {
	out := renderHelpPanel()
	if !strings.Contains(out, "\x1b[1m") && !strings.Contains(out, "\x1b[1;") {
		t.Errorf("expected at least one SGR-Bold span in the panel; got: %q", out)
	}
}

// Pressing 'h' from viewState toggles the help overlay flag on Model. A second 'h' closes it.
func TestHelpKeyToggles(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.prefs.HelpVisible {
		t.Fatalf("setup: helpVisible should start false")
	}

	m = pressKey(t, m, 'h')
	if !m.prefs.HelpVisible {
		t.Errorf("after first h, helpVisible = false; want true")
	}

	m = pressKey(t, m, 'h')
	if m.prefs.HelpVisible {
		t.Errorf("after second h, helpVisible = true; want false (toggle)")
	}
}

// While helpVisible, other keys still reach the active state's handler — the overlay does
// not block dispatch. Pressing 'b' transitions to picker; the overlay stays open.
func TestHelpStickyDuringModeAction(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), Stdout: "frame\n",
	}})

	m = pressKey(t, m, 'h')
	m = pressKey(t, m, 'b') // open picker

	if _, ok := m.state.(pickerState); !ok {
		t.Errorf("after b with help open, state = %T, want pickerState", m.state)
	}
	if !m.prefs.HelpVisible {
		t.Errorf("help overlay should stay open while user transitions to picker")
	}
}

// Pressing 'q' while help is open quits the app — global key dispatch is unaffected by
// the overlay.
func TestHelpQuitPassthrough(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = pressKey(t, m, 'h')

	_, cmd := m.dispatchKey(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatalf("expected tea.Quit cmd from q with help visible")
	}
	if msg := cmd(); msg != (tea.QuitMsg{}) {
		t.Errorf("q cmd produced %T, want tea.QuitMsg{}", msg)
	}
}

// In inputState, every key (including 'h') is consumed by the textinput. Help does NOT
// open, and the 'h' character lands in the input's value.
func TestHelpKeyInInputModeTypesLiteral(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = pressKey(t, m, 'r') // open record input
	if _, ok := m.state.(inputState); !ok {
		t.Fatalf("setup: expected inputState after r, got %T", m.state)
	}
	before := m.state.(inputState).input.Value()

	m = pressKey(t, m, 'h')

	if m.prefs.HelpVisible {
		t.Errorf("helpVisible should stay false when h is pressed inside inputState")
	}
	if got := m.state.(inputState).input.Value(); !strings.HasSuffix(got, "h") || got == before {
		t.Errorf("h keystroke should have appended to input value; before=%q after=%q", before, got)
	}
}

func TestHelpOverlayRendersInView(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m.ready = true
	m = feed(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = pressKey(t, m, 'h')

	plain := ansi.Strip(m.View().Content)
	for _, section := range []string{"Global", "View", "Picker", "Search", "Input"} {
		if !strings.Contains(plain, section) {
			t.Errorf("rendered View should contain section %q with help visible; got:\n%s", section, plain)
		}
	}
}
