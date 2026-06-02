package tui

// Cursor is the position into Session.History that the viewport renders from. The -1
// idx means "no frame yet" -- live mode before the first execution, or any time the
// session has zero entries. All transitions go through methods so the evict-shift,
// follow-tail, clamp, and sentinel rules live next to each other instead of scattered
// across the bar, the state files, and dispatchExec.
//
// Constructors must be used: the zero value Cursor{idx: 0} would read as "at frame 0",
// which is invalid before any execution has landed. Use noCursor or cursorAtTail.
type Cursor struct{ idx int }

// noCursor is the initial cursor for a live model: no frame to display yet.
func noCursor() Cursor { return Cursor{idx: -1} }

// cursorAtTail returns a cursor positioned at the last frame of a history of length n.
// Returns noCursor when n == 0.
func cursorAtTail(historyLen int) Cursor {
	if historyLen == 0 {
		return noCursor()
	}
	return Cursor{idx: historyLen - 1}
}

// cursorAt returns a cursor at the given index without clamping. Used by tests and by
// preloaded-session initialization where the caller already knows the index is valid
// for the corresponding history.
func cursorAt(idx int) Cursor { return Cursor{idx: idx} }

// At returns the current index and a "valid" flag. ok == false when the cursor sits at
// the no-frame-yet sentinel.
func (c Cursor) At() (int, bool) { return c.idx, c.Valid() }

// Index returns the raw cursor index. Useful for callers that have a downstream guard
// (e.g. FrameViewModel.Frame handles i < 0 by returning ""), or that arithmetically
// transform the index before passing it back through Move.
func (c Cursor) Index() int { return c.idx }

// Valid reports whether the cursor points at a real frame.
func (c Cursor) Valid() bool { return c.idx >= 0 }

// Following reports whether the cursor sits at the tail of a history of length n. An
// empty history (n == 0) is considered "following" so the first execution becomes the
// tail under view/picker's FollowsTail policy.
func (c Cursor) Following(historyLen int) bool {
	return historyLen == 0 || c.idx == historyLen-1
}

// AfterEvict shifts the cursor down by 1, clamping at 0, after a MaxHistory eviction
// removed slot 0. Keeps a non-tail viewer reading the same frame; clamps to 0 when the
// frame they were on was the one evicted.
func (c Cursor) AfterEvict() Cursor {
	return Cursor{idx: max(0, c.idx-1)}
}

// ToTail returns a cursor at the last frame of a history of length n. Equivalent to
// cursorAtTail; provided as a method so transition sites (dispatchExec's follow-tail
// branch) read as "advance to tail".
func (c Cursor) ToTail(historyLen int) Cursor { return cursorAtTail(historyLen) }

// Move returns a cursor clamped to [0, historyLen-1]. Used by picker navigation.
// Returns noCursor when historyLen == 0.
func (c Cursor) Move(to, historyLen int) Cursor {
	if historyLen == 0 {
		return noCursor()
	}
	return Cursor{idx: max(0, min(historyLen-1, to))}
}
