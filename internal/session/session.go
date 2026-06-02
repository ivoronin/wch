package session

import (
	"errors"
	"slices"
	"time"
)

// Execution represents a single command execution result
type Execution struct {
	Timestamp time.Time
	Stdout    string
	Stderr    string
	ExitCode  int
	Error     error
}

// Output returns combined stdout and stderr
func (e *Execution) Output() string {
	if e.Stderr == "" {
		return e.Stdout
	}
	if e.Stdout == "" {
		return e.Stderr
	}
	return e.Stdout + "\n" + e.Stderr
}

// Session holds execution history and the active recorder, if any. The Recorder port is
// defined here (port.go); concrete adapters and the persistence format live in
// internal/recording. The session owns the recorder's lifetime (same lifetime as the
// session itself) but knows nothing about JSONL, file I/O, or path normalization —
// recording.Flow is the entry point external callers should use.
type Session struct {
	Command    string
	Interval   time.Duration
	History    []Execution
	MaxHistory int
	recorder   Recorder // nil ⇔ not recording
}

// NewSession creates a new session
func NewSession(command string, interval time.Duration) *Session {
	return &Session{
		Command:  command,
		Interval: interval,
	}
}

// RecordIfChanged adds an execution to history only if it differs materially from the
// previous one — output, exit code, OR error string. A frame that prints the same text but
// changes exit code or error must not be dropped, otherwise downstream UI (exit-code
// annotation, OSC9 bell, replay) loses the transition. When a recording is active and the
// frame is added, the frame is also written to the file; a write error auto-finalizes the
// recording (close + clear) and is returned. added reports whether the execution was novel;
// evicted reports whether MaxHistory just rotated the oldest entry out (callers tracking a
// history cursor need to decrement it).
func (s *Session) RecordIfChanged(exec Execution) (added bool, evicted bool, err error) {
	if len(s.History) > 0 {
		last := s.History[len(s.History)-1]
		if exec.Output() == last.Output() &&
			exec.ExitCode == last.ExitCode &&
			errString(exec.Error) == errString(last.Error) {
			return false, false, nil
		}
	}
	s.History = append(s.History, exec)
	if s.MaxHistory > 0 && len(s.History) > s.MaxHistory {
		// slices.Delete (rather than History[1:]) copy-shifts and zeros the freed tail slot,
		// so the evicted Execution's stdout/stderr strings are released for GC instead of
		// staying pinned in the backing array until the next slice growth.
		s.History = slices.Delete(s.History, 0, 1)
		evicted = true
	}
	if s.recorder != nil {
		if writeErr := s.recorder.WriteFrame(exec); writeErr != nil {
			_ = s.recorder.Close()
			s.recorder = nil
			return true, evicted, writeErr
		}
	}
	return true, evicted, nil
}

func errString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

// StartRecording arms the session to persist subsequent additions through rec.
// rec.Initialize is called with the current Command, Interval, and the History backlog.
// External callers should drive recording through recording.Flow rather than constructing
// the Recorder + calling StartRecording directly; this signature exists for Flow's use
// and for in-package tests.
func (s *Session) StartRecording(rec Recorder) error {
	if s.recorder != nil {
		return errors.New("session: already recording")
	}
	if err := rec.Initialize(s.Command, s.Interval, s.History); err != nil {
		return err
	}
	s.recorder = rec
	return nil
}

// StopRecording closes the recording and clears the handle. Idempotent.
func (s *Session) StopRecording() error {
	if s.recorder == nil {
		return nil
	}
	err := s.recorder.Close()
	s.recorder = nil
	return err
}

// IsRecording reports whether a recording is currently active. External callers
// (cmd/wch, internal/tui) should query recording.Flow.IsActive() instead — this method
// exists so the recording package's Flow can read through to the session's single
// source of truth. Used directly only by Flow and by in-package tests.
func (s *Session) IsRecording() bool {
	return s.recorder != nil
}
