package tui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/compat"

	"github.com/ivoronin/wch/internal/recording"
	"github.com/ivoronin/wch/internal/runner"
	"github.com/ivoronin/wch/internal/session"
	"github.com/ivoronin/wch/internal/tui/notify"
)

// Config holds TUI configuration.
type Config struct {
	Command        string
	Interval       time.Duration
	DiffEnabled    bool
	ShowStatus     bool
	NotifyOnChange bool
	AutoStart      *recording.AutoStartRequest // non-nil: start a recording to this path at launch
	MaxHistory     int                         // executions retained in memory; 0 = unlimited
}

// Model is the Bubble Tea model. Domain (session, runner), infrastructure (viewport,
// notify), cross-state toggles, and the position cursor (historyIndex) live here. UI mode
// is held in m.state — a sealed sum type defined in state.go.
type Model struct {
	// Core
	session *session.Session
	runner  *runner.Runner

	// Infrastructure: frames owns the rendered viewport (Frame + ShowAnchored + embedded
	// scrollview navigation). State.Body says "what to display"; frames decides "how".
	frames FrameViewModel
	notify notify.Model

	// Cross-state toggles
	prefs Preferences

	// Runtime
	executing bool
	width     int
	height    int
	ready     bool

	// Persistence wiring
	flow      *recording.Flow
	autoStart *recording.AutoStartRequest

	// History cursor for view/picker. dispatchExec advances it per state.FollowsTail.
	cursor Cursor

	// UI state. Exactly one of {viewState, pickerState, inputState, searchState} at all
	// times. Overlay states (input, search) carry prev — the state to restore on Esc.
	state state
}

// notifyTTL is how long a notification bubble stays visible before its Tick expires.
const notifyTTL = time.Second

// push enqueues a notification bubble and returns the resulting Model + Cmd. All recording-
// status notifications go through here so the wiring stays in one place.
func (m Model) push(level notify.Level, msg string) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.notify, cmd = m.notify.Push(msg, level, notifyTTL)
	return m, cmd
}

// New creates a live TUI model that watches cfg.Command. If cfg.AutoStart is non-nil,
// recording to that path starts during Init.
func New(cfg Config) Model {
	sess := session.NewSession(cfg.Command, cfg.Interval)
	sess.MaxHistory = cfg.MaxHistory
	return Model{
		session: sess,
		runner:  runner.New(cfg.Command),
		flow:    recording.New(sess),
		frames:  newFrameViewModel(sess),
		cursor:  noCursor(),
		state:   viewState{},
		prefs: Preferences{
			Diff:      cfg.DiffEnabled,
			StatusBar: cfg.ShowStatus,
			OSNotify:  cfg.NotifyOnChange,
		},
		autoStart: cfg.AutoStart,
		notify:    notify.New(),
	}
}

// NewReplay creates an offline TUI model replaying a loaded session. No runner, no recording,
// no ticking. Replay-ness is encoded by the nil runner.
func NewReplay(cfg Config, s *session.Session) Model {
	return Model{
		session: s,
		runner:  nil,
		flow:    recording.New(s),
		frames:  newFrameViewModel(s),
		cursor:  cursorAtTail(len(s.History)),
		state:   viewState{},
		prefs: Preferences{
			Diff:      cfg.DiffEnabled,
			StatusBar: cfg.ShowStatus,
			OSNotify:  false,
		},
		notify: notify.New(),
	}
}

func (m Model) isLive() bool { return m.runner != nil }

// Cleanup finalises any resources held by the Model (today: an active recording). Bubble
// Tea v2 short-circuits Model.Update on QuitMsg — Update is never called for that message —
// so the Model has no chance to flush its own teardown. main.go calls Cleanup after p.Run
// returns. Idempotent: flow.Stop is a no-op when no recording is active.
func (m Model) Cleanup() error {
	return m.flow.Stop()
}

// Init starts the TUI. Replay returns nil — no tick is ever scheduled. Live mode kicks off
// the first execution tick and, if AutoStart was configured, begins recording.
func (m Model) Init() tea.Cmd {
	bgQuery := tea.RequestBackgroundColor
	if !m.isLive() {
		return bgQuery
	}
	tick := func() tea.Msg { return tickMsg{} }
	if m.autoStart == nil {
		return tea.Batch(bgQuery, tick)
	}
	path := m.autoStart.Path
	return tea.Batch(
		bgQuery,
		tick,
		func() tea.Msg { return autoStartRecordingMsg{path: path} },
	)
}

// Update routes messages. Uniform events (resize, quit, tick, exec, autoStart) are handled
// at the top; key and exec dispatch consults the active state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var notifyCmd tea.Cmd
	m.notify, notifyCmd = m.notify.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m2, cmd := m.dispatchKey(msg)
		return m2, tea.Batch(cmd, notifyCmd)
	case tea.WindowSizeMsg:
		m2, cmd := m.handleResize(msg)
		return m2, tea.Batch(cmd, notifyCmd)
	case tickMsg:
		m2, cmd := m.handleTick()
		return m2, tea.Batch(cmd, notifyCmd)
	case execResultMsg:
		m2, cmd := m.dispatchExec(msg)
		return m2, tea.Batch(cmd, notifyCmd)
	case autoStartRecordingMsg:
		m2, cmd, _ := m.startRecording(msg.path)
		return m2, tea.Batch(cmd, notifyCmd)
	case tea.BackgroundColorMsg:
		compat.HasDarkBackground = msg.IsDark()
		return m, notifyCmd
	}
	// Anything else (bracketed-paste PasteMsg/PasteStart/PasteEndMsg, cursor.BlinkMsg from
	// textinput.Focus, tea.FocusMsg/BlurMsg, etc.) is forwarded to the active inputState's
	// textinput when one exists, so its blink loop and focus tracking stay alive across the
	// dispatcher. No-op in any other state.
	m2, cmd := m.forwardToInputTextinput(msg)
	return m2, tea.Batch(cmd, notifyCmd)
}

// forwardToInputTextinput hands a non-key message to the textinput of the active
// inputState. Used for paste-mode messages (PasteMsg, PasteStart/End) and the
// general fallback (cursor.BlinkMsg from textinput.Focus, FocusMsg/BlurMsg) that
// the key dispatcher can't carry. No-op in any other state.
func (m Model) forwardToInputTextinput(msg tea.Msg) (Model, tea.Cmd) {
	s, ok := m.state.(inputState)
	if !ok {
		return m, nil
	}
	in, cmd := s.input.Update(msg)
	s.input = in
	m.state = s
	return m, cmd
}

// dispatchKey is the central key router: the active state gets first dibs via its
// Handle method; if it reports handled=false the global key handler runs (q/t/
// navigation defaults). When the new state's bar visibility differs from the
// pre-handler state's the viewport is resized; when the state TYPE changes (or bar
// flips) we repaint so any overlay leftover (e.g. searchState's reverse-video
// highlight) is replaced by the new state's content in the same step.
func (m Model) dispatchKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	barBefore := m.barShown()
	stateKindBefore := stateKind(m.state)
	nextM, s, cmd, handled := m.state.Handle(m, msg)
	m = nextM
	m.state = s
	barChanged := m.barShown() != barBefore
	stateChanged := stateKind(m.state) != stateKindBefore
	if barChanged {
		m = m.withResizedScrollview()
	}
	if barChanged || stateChanged {
		m = m.repaint()
	}
	if handled {
		return m, cmd
	}
	return m.handleGlobalKey(msg)
}

// stateKind returns a small integer tag identifying the concrete state type. Used by
// dispatchKey to detect state transitions without depending on reflect.
func stateKind(s state) int {
	switch s.(type) {
	case viewState:
		return 1
	case pickerState:
		return 2
	case inputState:
		return 3
	case searchState:
		return 4
	}
	return 0
}

// handleGlobalKey runs the global fall-through bindings: quit, status-bar toggle, and the
// viewport-navigation defaults (arrows/Home/End/PgUp/PgDn/Shift+arrows). Mutations happen
// directly on the viewport; no intra-update messages.
func (m Model) handleGlobalKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, globalKeys.Quit):
		// Recording finalisation is owned by main.go's Cleanup() call after p.Run returns —
		// Bubble Tea v2 short-circuits QuitMsg before Model.Update sees it, so there is no
		// in-Update teardown seam.
		return m, tea.Quit
	case key.Matches(msg, globalKeys.ToggleBar):
		m.prefs.StatusBar = !m.prefs.StatusBar
		m = m.withResizedScrollview()
		m = m.repaint()
		return m, nil
	case key.Matches(msg, globalKeys.Help):
		m.prefs.HelpVisible = !m.prefs.HelpVisible
		return m, nil
	case key.Matches(msg, navKeys.Up):
		m.frames.ScrollUp(1)
	case key.Matches(msg, navKeys.Down):
		m.frames.ScrollDown(1)
	case key.Matches(msg, navKeys.Left):
		m.frames.ScrollLeft()
	case key.Matches(msg, navKeys.Right):
		m.frames.ScrollRight()
	case key.Matches(msg, navKeys.PageUp):
		m.frames.PageUp()
	case key.Matches(msg, navKeys.PageDown):
		m.frames.PageDown()
	case key.Matches(msg, navKeys.ScrollLeft):
		m.frames.ScrollLeftPage()
	case key.Matches(msg, navKeys.ScrollRight):
		m.frames.ScrollRightPage()
	case key.Matches(msg, navKeys.Home):
		m.frames.GotoLeftEdge()
	case key.Matches(msg, navKeys.End):
		m.frames.GotoRightEdge()
	}
	return m, nil
}

// dispatchExec appends the new execution, advances historyIndex if the active state's
// follow policy says so, and refreshes the viewport. view/picker anchor against the previous
// output; search owns its frozen body; input is transparent and delegates both freeze and
// follow policy to its prev.
func (m Model) dispatchExec(msg execResultMsg) (Model, tea.Cmd) {
	m.executing = false

	prev, _ := m.state.Body(m)
	prior := m.cursor
	wasAtTail := m.isFollowing()
	added, evicted, err := m.session.RecordIfChanged(msg.exec)
	var cmds []tea.Cmd
	if err != nil {
		var c tea.Cmd
		m, c = m.push(notify.LevelWarning, "Recording error: "+err.Error())
		cmds = append(cmds, c)
	}
	// MaxHistory eviction shifted every slot index down by 1. For a non-tail viewer the
	// cursor must follow so the user keeps reading the same frame (clamped to 0 when the
	// frame they were on was the one evicted).
	if evicted {
		m.cursor = m.cursor.AfterEvict()
	}
	if added && m.state.FollowsTail(wasAtTail) {
		m.cursor = m.cursor.ToTail(len(m.session.History))
	}
	// Skip the full diff/anchor recomputation when nothing the viewport derives from has
	// changed (duplicate tick: !added && cursor didn't move). Important for the wch
	// primary use case — watching slow-changing kubectl output, where most ticks are dedup'd.
	if added || m.cursor != prior {
		m = m.repaintAnchored(prev)
	}
	// prior.Valid() gates the very first frame (when there was no prior to compare against).
	// This replaces the older len(History) > 1 gate, which was unreachable when MaxHistory==1.
	if m.prefs.OSNotify && added && prior.Valid() {
		cmds = append(cmds, sendNotification())
	}
	cmds = append(cmds, m.scheduleNextTick())
	return m, tea.Batch(cmds...)
}

// repaint asks the active state for its body and commits it to the viewport without
// anchor preservation. Used by handleResize, handleGlobalKey's bar toggle, and dispatchKey
// on a state transition — the cursor (historyIndex) hasn't moved, so the eye-on-line
// invariant doesn't apply.
func (m Model) repaint() Model {
	body, ok := m.state.Body(m)
	if !ok {
		return m
	}
	m.frames.SetContent(body)
	return m
}

// snapTarget describes a viewport scroll-to-cell effect that follows a paint. Used by
// searchState for its event-driven snap-to-match (search entry and n/p navigation) --
// kept outside the state interface because Body has no way to know whether the call
// context is event-triggered (snap wanted) or geometry-driven (snap unwanted).
type snapTarget struct {
	line, col, length int
}

// repaintWith commits s.Body(m) to the viewport and, when snap is non-nil, scrolls so
// the (line, col, length) region is visible. Used at sites where the freshly-built
// state isn't yet installed on Model (search entry constructing a searchState; n/p
// navigation mutating the searchState locally before returning it). When snap is nil,
// behaves like repaint applied to s.
func (m Model) repaintWith(s state, snap *snapTarget) Model {
	body, ok := s.Body(m)
	if !ok {
		return m
	}
	m.frames.SetContent(body)
	if snap != nil {
		m.frames.EnsureLineVisible(snap.line)
		m.frames.EnsureColumnVisible(snap.col, snap.length)
	}
	return m
}

// repaintAnchored asks the active state for its body and commits it via ShowAnchored
// against prev. States that report IsFrozen (search; input over search) own their frozen
// body and are left untouched.
func (m Model) repaintAnchored(prev string) Model {
	if !m.cursor.Valid() {
		return m
	}
	if m.state.IsFrozen() {
		return m
	}
	body, ok := m.state.Body(m)
	if !ok {
		return m
	}
	m.frames.ShowAnchored(body, prev)
	return m
}

// handleTick processes the periodic execution tick.
func (m Model) handleTick() (Model, tea.Cmd) {
	if m.prefs.Paused {
		return m, m.scheduleNextTick()
	}
	m.executing = true
	return m, m.executeCmd()
}

// handleResize responds to terminal size changes. The viewport geometry is updated, then
// the content is re-committed via the active state's Body (state-aware: searchState
// re-overlays its captured body with the highlight; view/picker re-derive from
// historyIndex; input delegates to its prev). Scroll position is preserved — no snap.
func (m Model) handleResize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true
	m = m.withResizedScrollview()
	m = m.repaint()
	// ClearScreen forces a full terminal redraw, preventing visual artifacts in terminal
	// multiplexers like Zellij that don't handle Bubble Tea's differential rendering
	// correctly on resize.
	return m, tea.ClearScreen
}

// executeCmd runs the command and returns the result as a message.
func (m Model) executeCmd() tea.Cmd {
	return func() tea.Msg {
		return execResultMsg{exec: m.runner.Execute(context.Background())}
	}
}

// scheduleNextTick schedules the next execution tick.
func (m Model) scheduleNextTick() tea.Cmd {
	return tea.Tick(m.session.Interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// withResizedScrollview returns a model with updated viewport dimensions.
func (m Model) withResizedScrollview() Model {
	w, h := m.width, m.height
	if m.barShown() {
		h--
	}
	m.frames.SetSize(w, h)
	return m
}

// withCursor returns a model with the cursor moved and content updated. Clamped no-op moves
// (e.g. pressing Right at the tail or Left at the head) short-circuit so we don't run a
// full Strip+Align+SetContent for an identity re-render on every boundary keypress.
func (m Model) withCursor(newIdx int) Model {
	moved := m.cursor.Move(newIdx, len(m.session.History))
	if moved == m.cursor {
		return m
	}
	prev, _ := m.state.Body(m)
	m.cursor = moved
	return m.repaintAnchored(prev)
}

// isFollowing returns true if viewing the latest history item.
func (m Model) isFollowing() bool {
	return m.cursor.Following(len(m.session.History))
}

// sendNotification sends an OSC9 notification via raw terminal output.
func sendNotification() tea.Cmd {
	return tea.Raw("\033]9;wch: output changed\a")
}
