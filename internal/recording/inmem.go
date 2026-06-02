package recording

import (
	"errors"
	"slices"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

// ErrInjectedWriteFailure is the error InMemoryRecorder returns from WriteFrame after
// the count configured via FailWriteAfter has been exhausted. Use errors.Is to detect.
var ErrInjectedWriteFailure = errors.New("recording: injected write failure")

// InMemoryRecorder is a session.Recorder that retains every header + frame in memory.
// Useful for tests (in this package and in session tests) and any in-process caller
// that wants to inspect recording behavior without touching the filesystem.
type InMemoryRecorder struct {
	header       Header
	frames       []Frame
	closed       bool
	writesBefore int // FailWriteAfter target; -1 = never fail
	writesDone   int
}

// NewInMemoryRecorder returns an empty in-memory recorder. Initialize must be called
// before WriteFrame; the zero state is intentionally invalid to surface misuse.
func NewInMemoryRecorder() *InMemoryRecorder {
	return &InMemoryRecorder{writesBefore: -1}
}

// FailWriteAfter arms the recorder to return ErrInjectedWriteFailure from the
// (n+1)-th WriteFrame call onward. Used by session.RecordIfChanged auto-finalize
// tests. Call before Initialize.
func (r *InMemoryRecorder) FailWriteAfter(n int) {
	r.writesBefore = n
}

// Initialize records the header (Command, Interval), resets the in-memory frame slice,
// and seeds it with the supplied backlog. The wch JSONL format is described by
// Header.Format/Version, so those are filled in from the package-level constants.
func (r *InMemoryRecorder) Initialize(command string, interval time.Duration, backlog []session.Execution) error {
	r.header = Header{
		Format:   FormatTag,
		Version:  SupportedVersion,
		Command:  command,
		Interval: interval.String(),
	}
	r.frames = r.frames[:0]
	for _, e := range backlog {
		r.frames = append(r.frames, frameFrom(e))
	}
	return nil
}

// WriteFrame appends a frame to the in-memory log, honoring FailWriteAfter if set.
func (r *InMemoryRecorder) WriteFrame(exec session.Execution) error {
	if r.writesBefore >= 0 && r.writesDone >= r.writesBefore {
		return ErrInjectedWriteFailure
	}
	r.writesDone++
	r.frames = append(r.frames, frameFrom(exec))
	return nil
}

// Close marks the recorder closed. Idempotent.
func (r *InMemoryRecorder) Close() error {
	r.closed = true
	return nil
}

// InMemoryHeader is the inspected view returned by Header(). Only the parsed Interval
// and Command are exposed; Format and Version are wch-internal constants.
type InMemoryHeader struct {
	Command  string
	Interval time.Duration
}

// Header returns the header recorded at Initialize time.
func (r *InMemoryRecorder) Header() InMemoryHeader {
	d, _ := time.ParseDuration(r.header.Interval)
	return InMemoryHeader{Command: r.header.Command, Interval: d}
}

// Frames returns a defensive copy of every frame captured (backlog + WriteFrame).
func (r *InMemoryRecorder) Frames() []Frame {
	return slices.Clone(r.frames)
}

// Closed reports whether Close has been called.
func (r *InMemoryRecorder) Closed() bool {
	return r.closed
}

// Compile-time assertion: InMemoryRecorder must satisfy session.Recorder.
var _ session.Recorder = (*InMemoryRecorder)(nil)
