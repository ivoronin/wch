package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/session"
	"github.com/ivoronin/wch/internal/tui/searchrender"
)

// searchTestModel builds a sized model with one frame containing repeated "Pending" entries
// for the search integration tests.
func searchTestModel(t *testing.T) Model {
	t.Helper()
	m := New(Config{Command: "kubectl get pods -A", Interval: time.Second})
	m = feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC),
		Stdout:    "pod-01 Running\npod-02 Pending\npod-03 Running\npod-04 Pending\npod-05 PENDING\n",
	}})
	return m
}

// Pressing '/' opens inputState with the search prompt and an empty value; the pre-search
// state lives on as input.prev.
func TestSearchOpenInput(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')

	in, ok := m.state.(inputState)
	if !ok {
		t.Fatalf("after /, m.state = %T, want inputState", m.state)
	}
	if in.input.Value() != "" {
		t.Errorf("search input should start empty; got %q", in.input.Value())
	}
	if in.input.Prompt != searchPromptLabel {
		t.Errorf("search input prompt = %q, want %q", in.input.Prompt, searchPromptLabel)
	}
	if _, ok := in.prev.(viewState); !ok {
		t.Errorf("input.prev = %T, want viewState", in.prev)
	}
}

// Submitting a query with matches transitions to searchState; body and matches are captured.
func TestSearchSubmitWithMatchesEntersSearchMode(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Pending")

	sm, ok := m.state.(searchState)
	if !ok {
		t.Fatalf("after submit with matches, m.state = %T, want searchState", m.state)
	}
	if sm.query != "Pending" {
		t.Errorf("query = %q, want Pending", sm.query)
	}
	if len(sm.matches) != 2 {
		t.Errorf("matches = %d, want 2", len(sm.matches))
	}
	if sm.selected != 0 {
		t.Errorf("selected = %d, want 0", sm.selected)
	}
	if _, ok := sm.prev.(viewState); !ok {
		t.Errorf("search.prev = %T, want viewState", sm.prev)
	}
}

// No matches → notification + pop back to the pre-input state.
func TestSearchSubmitNoMatchNotifiesAndPops(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "asdfqwer")

	if _, ok := m.state.(viewState); !ok {
		t.Errorf("after no-match submit, m.state = %T, want viewState", m.state)
	}
	if !m.notify.Active() {
		t.Errorf("expected notification bubble on no-match")
	}
}

// Empty/whitespace query is a silent cancel.
func TestSearchSubmitEmptyIsSilent(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "   ")

	if _, ok := m.state.(viewState); !ok {
		t.Errorf("after empty submit, m.state = %T, want viewState", m.state)
	}
	if m.notify.Active() {
		t.Errorf("empty submit must not push a notification")
	}
}

// n/p navigates matches with wrap.
func TestSearchNavigationWraps(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "pending") // smart-case → 3 matches

	sm := m.state.(searchState)
	if len(sm.matches) != 3 {
		t.Fatalf("setup: matches = %d, want 3", len(sm.matches))
	}

	m = pressKey(t, m, 'n')
	if m.state.(searchState).selected != 1 {
		t.Errorf("after 1×n, selected = %d, want 1", m.state.(searchState).selected)
	}
	m = pressKey(t, m, 'n')
	m = pressKey(t, m, 'n')
	if m.state.(searchState).selected != 0 {
		t.Errorf("after 3×n, selected = %d, want 0 (wrap)", m.state.(searchState).selected)
	}
	m = pressKey(t, m, 'N')
	if m.state.(searchState).selected != 2 {
		t.Errorf("after N from 0, selected = %d, want 2 (wrap)", m.state.(searchState).selected)
	}
}

// execResultMsg during searchState appends to history but does NOT change the viewport
// content (frozen snapshot).
func TestSearchFreezeDuringExecResult(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Pending")

	before := strings.Clone(m.state.(searchState).body)
	historyLenBefore := len(m.session.History)

	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 1, 0, time.UTC),
		Stdout:    "pod-99 Running\n",
	}})

	if _, ok := m.state.(searchState); !ok {
		t.Errorf("searchState lost across execResultMsg; got %T", m.state)
	}
	if got := m.state.(searchState).body; got != before {
		t.Errorf("frozen body changed after execResultMsg")
	}
	if len(m.session.History) == historyLenBefore {
		t.Errorf("history should still grow during searchState; remained %d", historyLenBefore)
	}
}

// search-from-tail: Esc must resume live following in one keystroke.
func TestSearchEscRestoresLiveTail(t *testing.T) {
	m := searchTestModel(t)
	if !m.isFollowing() {
		t.Fatalf("setup: model should start at the live tail")
	}
	indexAtSearchStart := m.cursor.Index()

	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Pending")

	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 1, 0, time.UTC),
		Stdout:    "pod-06 Pending\n",
	}})
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 2, 0, time.UTC),
		Stdout:    "pod-07 Running\n",
	}})

	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if _, ok := m.state.(viewState); !ok {
		t.Fatalf("after Esc, m.state = %T, want viewState", m.state)
	}
	if !m.isFollowing() {
		t.Errorf("after Esc, isFollowing() = false; want true (search-from-tail resumes following)")
	}
	if m.cursor.Index() == indexAtSearchStart {
		t.Errorf("after Esc, historyIndex = %d (stuck at search-start); should advance to %d",
			m.cursor.Index(), len(m.session.History)-1)
	}
}

// search-from-past: Esc returns to that same past frame; no follow.
func TestSearchEscFromPastFrameStaysOnPast(t *testing.T) {
	m := searchTestModel(t)
	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 1, 0, time.UTC),
		Stdout:    "pod-06 Running\n",
	}})
	m = m.withCursor(0)
	if m.isFollowing() {
		t.Fatalf("setup: should be viewing past frame")
	}
	pastIndex := m.cursor.Index()

	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Pending")

	m = feed(t, m, execResultMsg{exec: session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 2, 0, time.UTC),
		Stdout:    "pod-07 Running\n",
	}})

	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.cursor.Index() != pastIndex {
		t.Errorf("after Esc from past-frame search, historyIndex = %d, want %d", m.cursor.Index(), pastIndex)
	}
}

// Esc out of searchState returns to the pre-search state.
func TestSearchEscReturnsToPreSearchMode(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Pending")
	if _, ok := m.state.(searchState); !ok {
		t.Fatalf("setup: should be in searchState, got %T", m.state)
	}

	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if _, ok := m.state.(viewState); !ok {
		t.Errorf("after Esc, m.state = %T, want viewState", m.state)
	}
}

// picker → search → Esc returns to picker via the prev chain.
func TestSearchFromPickerEscReturnsToPicker(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, 'b') // enter picker
	if _, ok := m.state.(pickerState); !ok {
		t.Fatalf("setup: should be in pickerState, got %T", m.state)
	}

	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Pending")
	if _, ok := m.state.(searchState); !ok {
		t.Fatalf("should be searchState, got %T", m.state)
	}

	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if _, ok := m.state.(pickerState); !ok {
		t.Errorf("after Esc from search-via-picker, m.state = %T, want pickerState", m.state)
	}
}

// '/' inside searchState replaces the current search on the stack: input.prev skips the
// abandoned search. Esc from the replacement returns to the original predecessor (viewState).
func TestSearchSubmitFromSearchModeReplacesOnStack(t *testing.T) {
	m := searchTestModel(t)
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "Running")
	first := m.state.(searchState)

	// '/' from within searchState.
	m = pressKey(t, m, '/')
	in, ok := m.state.(inputState)
	if !ok {
		t.Fatalf("after / in search, m.state = %T, want inputState", m.state)
	}
	// input.prev should be the searchState's prev (viewState), NOT the old searchState.
	if _, ok := in.prev.(viewState); !ok {
		t.Errorf("input.prev = %T, want viewState (search-from-search replaces)", in.prev)
	}

	m = submitInputValue(t, m, "Pending")
	second, ok := m.state.(searchState)
	if !ok {
		t.Fatalf("after second submit, m.state = %T, want searchState", m.state)
	}
	if second.query == first.query {
		t.Errorf("new searchState kept old query %q; should be Pending", second.query)
	}

	m = feed(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if _, ok := m.state.(viewState); !ok {
		t.Errorf("after Esc from replacing search, m.state = %T, want viewState", m.state)
	}
}

// TestSearchStateBodyReturnsOverlay: searchState.Body returns the captured body
// with the selected match wrapped in the searchrender overlay.
func TestSearchStateBodyReturnsOverlay(t *testing.T) {
	m := makePaintModel(t)
	body := m.frames.Frame(m.cursor.Index(), m.prefs.Diff)
	ss := searchState{
		query:    "pod-15",
		body:     body,
		matches:  findMatches(body, "pod-15"),
		selected: 0,
	}
	if len(ss.matches) == 0 {
		t.Fatalf("test setup: no matches for pod-15 in body")
	}
	sel := ss.matches[0]
	want := searchrender.Render(body, sel.line, sel.col, sel.length)

	got, ok := ss.Body(m)
	if !ok {
		t.Fatalf("searchState.Body: ok=false with matches present")
	}
	if got != want {
		t.Errorf("searchState.Body did not produce the searchrender overlay")
	}
}

// TestSearchStateBodyZeroMatches: with zero matches, searchState.Body reports
// ok=false so the orchestrator skips repaint.
func TestSearchStateBodyZeroMatches(t *testing.T) {
	m := makePaintModel(t)
	ss := searchState{body: "anything", matches: nil}
	if _, ok := ss.Body(m); ok {
		t.Errorf("searchState.Body: ok=true with zero matches")
	}
}

// TestRestartSearchClearsOverlay: pressing '/' from within searchState
// transitions to inputState{prev: viewState}; the viewport must show the
// live (un-highlighted) body, not the prior search's overlay.
//
// The dispatchKey bar-transition hook ends with m.repaint(), which for inputState
// delegates to its prev's Body, which yields the live frame. Before the seam
// landed, the viewport stayed on the prior search overlay until the next exec.
func TestRestartSearchClearsOverlay(t *testing.T) {
	m := New(Config{Command: "x", Interval: time.Second})
	m.prefs.StatusBar = false
	m = feed(t, m, tea.WindowSizeMsg{Width: 40, Height: 10})
	m = feed(t, m, execResultMsg{exec: session.Execution{Stdout: podTable("5m", podNames(20))}})

	// Enter search.
	m = pressKey(t, m, '/')
	m = submitInputValue(t, m, "pod-15")
	if _, ok := m.state.(searchState); !ok {
		t.Fatalf("after submit: m.state = %T, want searchState", m.state)
	}
	overlay := m.frames.View()
	if !strings.Contains(overlay, "\x1b[7m") {
		t.Fatalf("test setup: overlay snapshot lacks reverse-video highlight; searchState did not paint as expected")
	}

	// Re-search via '/'. State transitions search → input{prev: view}.
	// The bar-transition resize hook must repaint, which (via input → view)
	// commits the live body to the viewport.
	m = pressKey(t, m, '/')
	if _, ok := m.state.(inputState); !ok {
		t.Fatalf("after second '/': m.state = %T, want inputState", m.state)
	}
	got := m.frames.View()

	if got == overlay {
		t.Fatalf("viewport still shows the search overlay after re-search; expected the live body to replace it")
	}
	if strings.Contains(got, "\x1b[7m") {
		t.Errorf("viewport still carries reverse-video overlay after re-search; got: %q", got)
	}
}

func TestSearchStateShowsBar(t *testing.T) {
	if got := (searchState{prev: viewState{}}).ShowsBar(); got != false {
		t.Errorf("searchState.ShowsBar() = %v, want false", got)
	}
}

func TestSearchStateIsFrozen(t *testing.T) {
	if got := (searchState{prev: viewState{}}).IsFrozen(); got != true {
		t.Errorf("searchState.IsFrozen() = %v, want true", got)
	}
}

func TestSearchStateFollowsTailUsesCapturedFlag(t *testing.T) {
	cases := []struct {
		name      string
		follow    bool
		wasAtTail bool
		want      bool
	}{
		{"follow=true ignores wasAtTail=false", true, false, true},
		{"follow=false ignores wasAtTail=true", false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := searchState{prev: viewState{}, follow: c.follow}
			if got := s.FollowsTail(c.wasAtTail); got != c.want {
				t.Errorf("FollowsTail(%v) = %v, want %v", c.wasAtTail, got, c.want)
			}
		})
	}
}

// TestSearchStateHandleEscReturnsToPrev pins Esc returns to the prev (pre-search) state.
func TestSearchStateHandleEscReturnsToPrev(t *testing.T) {
	m := makePaintModel(t)
	ss := searchState{prev: viewState{}}
	m.state = ss
	_, st, _, handled := ss.Handle(m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if !handled {
		t.Fatalf("searchState.Handle(Esc) reported handled=false")
	}
	if _, ok := st.(viewState); !ok {
		t.Errorf("searchState.Handle(Esc) returned %T, want viewState (prev)", st)
	}
}

func TestSearchHelpBindingsMinimal(t *testing.T) {
	bindings := searchHelpBindings()
	if len(bindings) != 4 {
		t.Errorf("searchHelpBindings len = %d, want 4 (Esc, n, Help, Quit)", len(bindings))
	}
}
