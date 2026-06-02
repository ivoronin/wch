// Package notify renders transient corner notifications ("bubbles") as overlays on top of
// existing Bubble Tea views. A new bubble pushed via Push appears in the configured corner;
// each carries its own timer Cmd that, threaded back through Update, removes the bubble
// after the requested TTL. Use Overlay to compose the bubble stack onto your own View
// output without disrupting underlying content.
package notify

import (
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Level categorizes a notification. Default styling tints the border accordingly.
type Level int

const (
	LevelInfo Level = iota
	LevelWarning
)

// Position picks the corner the stack grows from. Newest bubble appears closest to the
// corner; older ones slide outward.
type Position int

const (
	BottomRight Position = iota
	BottomLeft
	TopRight
	TopLeft
)

// defaultMaxVisible caps the visible stack so a flood of notifications doesn't cover the view.
const defaultMaxVisible = 5

// Model is a stack of transient notifications.
type Model struct {
	bubbles    []bubble
	position   Position
	maxVisible int
	nextID     int
	styles     map[Level]lipgloss.Style
}

// bubble is one notification in the stack.
type bubble struct {
	id      int
	message string
	level   Level
}

// expireMsg is the only message this package emits via tea.Tick; Update consumes it.
type expireMsg struct{ id int }

// Option configures a Model at construction.
type Option func(*Model)

// WithPosition picks the stack corner.
func WithPosition(p Position) Option { return func(m *Model) { m.position = p } }

// WithMaxVisible caps the number of simultaneous bubbles. n ≤ 0 disables the cap.
func WithMaxVisible(n int) Option { return func(m *Model) { m.maxVisible = n } }

// New constructs a Model. Defaults: BottomRight, maxVisible=5; per-level styling is set
// by defaultStyles (info untinted, warning yellow-bordered).
func New(opts ...Option) Model {
	m := Model{
		position:   BottomRight,
		maxVisible: defaultMaxVisible,
		styles:     defaultStyles(),
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

// Push enqueues a notification. The returned Cmd fires after ttl; threading it through
// Update will remove the matching bubble.
func (m Model) Push(message string, level Level, ttl time.Duration) (Model, tea.Cmd) {
	m.nextID++
	id := m.nextID
	m.bubbles = append(m.bubbles, bubble{id: id, message: message, level: level})
	// Evict the oldest if over the visible cap.
	if m.maxVisible > 0 && len(m.bubbles) > m.maxVisible {
		m.bubbles = m.bubbles[len(m.bubbles)-m.maxVisible:]
	}
	return m, tea.Tick(ttl, func(time.Time) tea.Msg {
		return expireMsg{id: id}
	})
}

// Update consumes the internal expiry messages. Foreign messages return the model unchanged.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	exp, ok := msg.(expireMsg)
	if !ok {
		return m, nil
	}
	if i := slices.IndexFunc(m.bubbles, func(b bubble) bool { return b.id == exp.id }); i >= 0 {
		m.bubbles = slices.Delete(m.bubbles, i, i+1)
	}
	return m, nil
}

// Active reports whether any bubble is currently being shown.
func (m Model) Active() bool {
	return len(m.bubbles) > 0
}
