package tui

import "github.com/ivoronin/wch/internal/session"

// Execution messages
type (
	execResultMsg struct{ exec session.Execution }
	tickMsg       struct{}
)

// Scroll messages - intent to scroll the viewport
type (
	scrollUpMsg        struct{ lines int }
	scrollDownMsg      struct{ lines int }
	scrollLeftMsg      struct{}
	scrollRightMsg     struct{}
	pageUpMsg          struct{}
	pageDownMsg        struct{}
	scrollLeftPageMsg  struct{}
	scrollRightPageMsg struct{}
	gotoLeftEdgeMsg    struct{}
	gotoRightEdgeMsg   struct{}
)

// Mode messages
type switchModeMsg struct{ mode Mode }

// Cursor messages
type (
	moveCursorMsg      struct{ delta int }
	gotoFirstCursorMsg struct{}
	gotoLastCursorMsg  struct{}
)

// Global action messages
type (
	toggleDiffMsg  struct{}
	toggleBarMsg   struct{}
	togglePauseMsg struct{}
	escapeMsg      struct{}
)
