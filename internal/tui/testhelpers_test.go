package tui

// Helpers shared by every *_test.go in this package. Lives without a paired source
// because it's explicitly helper-only — no Test* functions, just setup code that
// would otherwise be duplicated across multiple test files.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/session"
)

// podTable renders a kubectl-like table: a stable header plus one row per name, each row
// carrying the given (volatile) age.
func podTable(age string, names []string) string {
	rows := []string{"NAME READY STATUS AGE"}
	for _, n := range names {
		rows = append(rows, n+" 1/1 Running "+age)
	}
	return strings.Join(rows, "\n")
}

func podNames(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("pod-%02d", i+1)
	}
	return out
}

func feed(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

// pressKey simulates a single key press through the full dispatcher (Update path).
func pressKey(t *testing.T, m Model, code rune) Model {
	t.Helper()
	return feed(t, m, tea.KeyPressMsg{Code: code, Text: string(code)})
}

// submitInputValue sets the textinput's value directly on m.state (which must be inputState)
// and presses Enter so the configured submit function runs. Bypasses character-by-character
// typing, which would otherwise dominate the test setup.
func submitInputValue(t *testing.T, m Model, value string) Model {
	t.Helper()
	s, ok := m.state.(inputState)
	if !ok {
		t.Fatalf("submitInputValue: m.state = %T, want inputState", m.state)
	}
	s.input.SetValue(value)
	m.state = s
	return feed(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

func newSizedModel(t *testing.T, initial string) Model {
	t.Helper()
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	return feed(t, m, execResultMsg{exec: session.Execution{Stdout: initial}})
}

// makePaintModel returns a model sized 40x10 with a single recorded execution.
// All Body equivalence tests share the same setup so that the "what should the
// viewport look like?" baseline is identical across cases.
func makePaintModel(t *testing.T) Model {
	t.Helper()
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("5m", podNames(20))}})
	return m
}
