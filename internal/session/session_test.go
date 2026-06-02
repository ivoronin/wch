package session_test

import (
	"testing"
	"time"

	"github.com/ivoronin/wch/internal/recording"
	"github.com/ivoronin/wch/internal/session"
)

// RecordIfChanged appends the first execution unconditionally.
func TestRecordIfChangedFirstExecution(t *testing.T) {
	s := session.NewSession("x", time.Second)
	added, _, err := s.RecordIfChanged(session.Execution{Stdout: "hello"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !added {
		t.Errorf("first execution should be added")
	}
	if len(s.History) != 1 {
		t.Errorf("History len=%d want 1", len(s.History))
	}
}

// Duplicate output (same Output()) is dropped, History stays the same length.
func TestRecordIfChangedDeduplicates(t *testing.T) {
	s := session.NewSession("x", time.Second)
	_, _, _ = s.RecordIfChanged(session.Execution{Stdout: "a\n"})
	added, _, err := s.RecordIfChanged(session.Execution{Stdout: "a\n"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if added {
		t.Errorf("duplicate output should not be added")
	}
	if len(s.History) != 1 {
		t.Errorf("History len=%d want 1", len(s.History))
	}
}

// A changed output is added.
func TestRecordIfChangedAddsChanged(t *testing.T) {
	s := session.NewSession("x", time.Second)
	_, _, _ = s.RecordIfChanged(session.Execution{Stdout: "a\n"})
	added, _, err := s.RecordIfChanged(session.Execution{Stdout: "b\n"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !added {
		t.Errorf("changed output should be added")
	}
	if len(s.History) != 2 {
		t.Errorf("History len=%d want 2", len(s.History))
	}
}

// Once MaxHistory is set and exceeded, the oldest frame is trimmed.
func TestRecordIfChangedTrimsAtMaxHistory(t *testing.T) {
	s := session.NewSession("x", time.Second)
	s.MaxHistory = 3
	for i := 0; i < 5; i++ {
		_, _, _ = s.RecordIfChanged(session.Execution{Stdout: string(rune('a' + i))})
	}
	if got, want := len(s.History), 3; got != want {
		t.Fatalf("History len=%d want %d", got, want)
	}
	// We should have kept the *last* three.
	if s.History[0].Stdout != "c" || s.History[2].Stdout != "e" {
		t.Errorf("trimmed wrong end; got %q..%q", s.History[0].Stdout, s.History[2].Stdout)
	}
}

// StopRecording is idempotent — safe to call on a session that isn't recording.
func TestRecordingStopIdempotent(t *testing.T) {
	s := session.NewSession("x", time.Second)
	if err := s.StopRecording(); err != nil {
		t.Errorf("Stop on non-recording session: %v", err)
	}

	rec := recording.NewInMemoryRecorder()
	if err := s.StartRecording(rec); err != nil {
		t.Fatalf("StartRecording: %v", err)
	}
	if err := s.StopRecording(); err != nil {
		t.Errorf("first Stop: %v", err)
	}
	if err := s.StopRecording(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
	if s.IsRecording() {
		t.Errorf("IsRecording() should be false")
	}
}

// StartRecording while a recording is already active returns an error and leaves the existing
// recording untouched.
func TestRecordingStartWhileActiveErrors(t *testing.T) {
	s := session.NewSession("x", time.Second)
	rec1 := recording.NewInMemoryRecorder()
	if err := s.StartRecording(rec1); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	rec2 := recording.NewInMemoryRecorder()
	if err := s.StartRecording(rec2); err == nil {
		t.Errorf("expected Start while already recording to return an error")
	}
	if !s.IsRecording() {
		t.Errorf("IsRecording() should still be true (existing recording untouched)")
	}
	if rec2.Closed() {
		t.Errorf("second recorder must not have been touched")
	}
	_ = s.StopRecording()
}

// When a write fails mid-recording, RecordIfChanged returns the error, the session auto-
// finalizes (recorder closed, handle cleared), IsRecording() flips to false.
func TestRecordingAutoFinalizeOnWriteError(t *testing.T) {
	s := session.NewSession("x", time.Second)
	rec := recording.NewInMemoryRecorder()
	rec.FailWriteAfter(0) // any WriteFrame fails immediately
	if err := s.StartRecording(rec); err != nil {
		t.Fatalf("StartRecording: %v", err)
	}
	added, _, err := s.RecordIfChanged(session.Execution{Stdout: "boom\n"})
	if err == nil {
		t.Fatal("expected write error, got nil")
	}
	if !added {
		t.Errorf("frame should still count as added to in-memory History before the write attempt")
	}
	if s.IsRecording() {
		t.Errorf("IsRecording() should be false after auto-finalize")
	}
	// And the frame *is* in History — persistence failed but in-memory dedupe succeeded.
	if len(s.History) != 1 {
		t.Errorf("History len=%d want 1", len(s.History))
	}
}
