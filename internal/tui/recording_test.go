package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/recording"
	"github.com/ivoronin/wch/internal/session"
)

// --- Replay & recording integration ---------------------------------------

func preloadedReplaySession(n int) *session.Session {
	s := session.NewSession("kubectl", time.Second)
	for i := 0; i < n; i++ {
		_, _, _ = s.RecordIfChanged(session.Execution{
			Timestamp: time.Date(2026, 5, 30, 12, 0, i, 0, time.UTC),
			Stdout:    fmt.Sprintf("frame %d\n", i),
		})
	}
	return s
}

// Regression: opening an input prompt must leave the textinput focused. Without focus,
// textinput.Update drops every keypress and the user can't type. Previously openInput built
// the inputState with an unfocused textinput value and then called Focus() on a separate
// local copy, so the stored input remained focused=false.
func TestRecordInputFocusedAndAcceptsKeystrokes(t *testing.T) {
	m := New(Config{Command: "kubectl", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = pressKey(t, m, 'r')

	in, ok := m.state.(inputState)
	if !ok {
		t.Fatalf("setup: m.state = %T, want inputState", m.state)
	}
	if !in.input.Focused() {
		t.Fatalf("input must be focused immediately after openInput; was not")
	}

	before := in.input.Value()
	m = pressKey(t, m, 'X') // arbitrary printable
	got := m.state.(inputState).input.Value()
	if got == before {
		t.Errorf("typing 'X' didn't change the input value (still %q) — keystroke was dropped", got)
	}
	if !strings.HasSuffix(got, "X") {
		t.Errorf("input value = %q, want a trailing X", got)
	}
}

// Pressing 'r' on a live model switches into inputState with a pre-filled default filename.
func TestRecordKeyOpensInputMode(t *testing.T) {
	m := New(Config{Command: "kubectl get pods -A", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = pressKey(t, m, 'r')

	in, ok := m.state.(inputState)
	if !ok {
		t.Fatalf("after r-key, m.state = %T, want inputState", m.state)
	}
	v := in.input.Value()
	if !strings.HasPrefix(v, "kubectl_get_pods_A_") || !strings.HasSuffix(v, ".wch.jsonl") {
		t.Errorf("pre-filled value = %q, want kubectl_get_pods_A_<ts>.wch.jsonl", v)
	}
	if m.session.IsRecording() {
		t.Errorf("session must not be recording yet — only the prompt is open")
	}
}

// Submitting a valid path starts recording, returns to viewState, and the backlog plus
// subsequent frames round-trip via session.Load.
func TestSubmitInputStartsRecording(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")
	m := New(Config{Command: "kubectl", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})

	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), Stdout: "alpha\n",
	}})

	m = pressKey(t, m, 'r')
	if _, ok := m.state.(inputState); !ok {
		t.Fatalf("setup: expected inputState after r, got %T", m.state)
	}
	m = submitInputValue(t, m, path)

	if !m.session.IsRecording() {
		t.Fatalf("expected IsRecording() after submit")
	}
	if _, ok := m.state.(viewState); !ok {
		t.Errorf("after successful submit, m.state = %T, want viewState", m.state)
	}
	if !m.notify.Active() {
		t.Errorf("expected notification bubble on successful start")
	}

	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 1, 0, time.UTC), Stdout: "beta\n",
	}})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 2, 0, time.UTC), Stdout: "gamma\n",
	}})

	m = pressKey(t, m, 'r')
	if m.session.IsRecording() {
		t.Errorf("expected IsRecording() == false after second toggle")
	}

	loaded, err := recording.Load(path)
	if err != nil {
		t.Fatalf("recording.Load: %v", err)
	}
	want := []string{"alpha\n", "beta\n", "gamma\n"}
	if len(loaded.History) != len(want) {
		t.Fatalf("loaded len=%d want %d", len(loaded.History), len(want))
	}
	for i, w := range want {
		if loaded.History[i].Stdout != w {
			t.Errorf("frame %d stdout=%q want %q", i, loaded.History[i].Stdout, w)
		}
	}
}

// Empty input is refused: warning bubble fires; the prompt stays open with the value.
func TestSubmitEmptyValueRefuses(t *testing.T) {
	m := New(Config{Command: "kubectl", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = pressKey(t, m, 'r')
	m = submitInputValue(t, m, "   ")

	if m.session.IsRecording() {
		t.Errorf("Recording must not start on empty submit")
	}
	if _, ok := m.state.(inputState); !ok {
		t.Errorf("after refusal, m.state = %T, want inputState (prompt stays open)", m.state)
	}
	if !m.notify.Active() {
		t.Errorf("expected warning bubble on empty submit")
	}
}

// Submitting a path that already exists is refused with a warning; prompt stays open.
func TestSubmitExistingFileRefuses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exists.wch.jsonl")
	if err := os.WriteFile(path, []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	m := New(Config{Command: "kubectl", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = pressKey(t, m, 'r')
	m = submitInputValue(t, m, path)

	if m.session.IsRecording() {
		t.Errorf("Recording must not start on existing-file submit")
	}
	if _, ok := m.state.(inputState); !ok {
		t.Errorf("after refusal, m.state = %T, want inputState", m.state)
	}
	if !m.notify.Active() {
		t.Errorf("expected warning bubble on existing-file submit")
	}
}

// Esc out of inputState returns to viewState without starting a recording.
func TestCancelClosesInput(t *testing.T) {
	m := New(Config{Command: "kubectl", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = pressKey(t, m, 'r')
	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})

	if m.session.IsRecording() {
		t.Errorf("cancel must not start a recording")
	}
	if _, ok := m.state.(viewState); !ok {
		t.Errorf("after cancel, m.state = %T, want viewState", m.state)
	}
}

// In replay, 'r' is a no-op and the Record binding is dropped from viewHelpBindings.
func TestRecordKeyInReplayMode(t *testing.T) {
	m := NewReplay(Config{}, preloadedReplaySession(1))
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})

	for _, b := range viewHelpBindings(m) {
		if b.Help().Key == commonKeys.Record.Help().Key {
			t.Errorf("Record binding should not appear in replay help; got %v", b.Help())
		}
	}

	m = pressKey(t, m, 'r')
	if m.session.IsRecording() {
		t.Errorf("replay must not start recording")
	}
	if _, ok := m.state.(inputState); ok {
		t.Errorf("replay must not open input prompt")
	}
}

// Successful submit pushes a "Recording started" bubble onto the notify stack.
func TestSubmitRecordPushesBubble(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wch.jsonl")
	m := New(Config{Command: "x", Interval: time.Second})
	m.ready = true
	m = feed(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})

	m = pressKey(t, m, 'r')
	m = submitInputValue(t, m, path)
	if !m.notify.Active() {
		t.Fatalf("expected notify.Active() after successful start")
	}
	rendered := ansi.Strip(m.View().Content)
	if !strings.Contains(rendered, "Recording started") {
		t.Errorf("rendered View missing start message; got:\n%s", rendered)
	}
}

// Cleanup finalises the recording when called after p.Run returns. Bubble Tea v2
// short-circuits Model.Update on QuitMsg, so the cleanup contract lives on Model and is
// invoked by main.go — not by the QuitMsg handler.
func TestCleanupFinalizesRecording(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")
	m := New(Config{Command: "x", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), Stdout: "kept\n",
	}})

	m = pressKey(t, m, 'r')
	m = submitInputValue(t, m, path)
	if !m.session.IsRecording() {
		t.Fatalf("setup: should be recording after submit")
	}

	if err := m.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if m.session.IsRecording() {
		t.Errorf("IsRecording() must be false after Cleanup")
	}

	loaded, err := recording.Load(path)
	if err != nil {
		t.Fatalf("recording.Load: %v", err)
	}
	if len(loaded.History) == 0 {
		t.Errorf("expected at least one frame in the finalized file")
	}
}
