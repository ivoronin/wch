// Package tui's intra-update messages. Only true asynchronous events live here: tick
// (timer) and execResult (background runner result). Recording-related messages live with
// the rest of the recording lifecycle in recording.go. State transitions, key intents, and
// scroll commands are direct function calls in the dispatcher chain; not messages.
package tui

import "github.com/ivoronin/wch/internal/session"

type (
	// execResultMsg carries the runner's output back to Update after a tick.
	execResultMsg struct{ exec session.Execution }

	// tickMsg fires every Config.Interval to schedule the next runner execution.
	tickMsg struct{}
)
