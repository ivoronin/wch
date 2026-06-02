package recording

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

// helpers

func mustStartJSONL(t *testing.T, s *session.Session, path string) *JSONLRecorder {
	t.Helper()
	rec, err := NewJSONLRecorder(path)
	if err != nil {
		t.Fatalf("NewJSONLRecorder: %v", err)
	}
	if err := s.StartRecording(rec); err != nil {
		t.Fatalf("StartRecording: %v", err)
	}
	return rec
}

func mustRecord(t *testing.T, s *session.Session, exec session.Execution) {
	t.Helper()
	if _, _, err := s.RecordIfChanged(exec); err != nil {
		t.Fatalf("RecordIfChanged: %v", err)
	}
}

// A full round-trip preserves ANSI in stdout, separate stderr, non-zero exit, and the error
// message string. The recording is closed before Load, replay never re-arms.
func TestRecordingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")

	s := session.NewSession("kubectl get pods", 5*time.Second)
	mustStartJSONL(t, s, path)

	mustRecord(t, s, session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 1, 0, time.UTC),
		Stdout:    "\x1b[1mNAME\x1b[0m\npod-1 \x1b[32mRunning\x1b[0m\n",
	})
	mustRecord(t, s, session.Execution{
		Timestamp: time.Date(2026, 5, 30, 12, 0, 2, 0, time.UTC),
		Stdout:    "out\n",
		Stderr:    "warn\n",
		ExitCode:  2,
		Error:     errors.New("exit status 2"),
	})

	if err := s.StopRecording(); err != nil {
		t.Fatalf("StopRecording: %v", err)
	}
	if s.IsRecording() {
		t.Errorf("IsRecording() should be false after Stop")
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Command != "kubectl get pods" {
		t.Errorf("Command=%q want %q", got.Command, "kubectl get pods")
	}
	if got.Interval != 5*time.Second {
		t.Errorf("Interval=%v want %v", got.Interval, 5*time.Second)
	}
	if len(got.History) != 2 {
		t.Fatalf("History len=%d want 2", len(got.History))
	}
	if got.History[0].Stdout != s.History[0].Stdout {
		t.Errorf("ANSI stdout not preserved: %q vs %q", got.History[0].Stdout, s.History[0].Stdout)
	}
	if got.History[1].Stderr != "warn\n" {
		t.Errorf("Stderr=%q want %q", got.History[1].Stderr, "warn\n")
	}
	if got.History[1].ExitCode != 2 {
		t.Errorf("ExitCode=%d want 2", got.History[1].ExitCode)
	}
	if got.History[1].Error == nil || got.History[1].Error.Error() != "exit status 2" {
		t.Errorf("Error not preserved: %v", got.History[1].Error)
	}
	if got.IsRecording() {
		t.Errorf("Loaded session must not be armed for recording")
	}
}

// StartRecording dumps every frame already in History.
func TestRecordingStartDumpsBacklog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")

	s := session.NewSession("x", time.Second)
	for i := 0; i < 4; i++ {
		mustRecord(t, s, session.Execution{
			Timestamp: time.Date(2026, 5, 30, 12, 0, i, 0, time.UTC),
			Stdout:    fmt.Sprintf("frame %d\n", i),
		})
	}
	mustStartJSONL(t, s, path)
	if err := s.StopRecording(); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.History) != 4 {
		t.Fatalf("backlog dump frames=%d want 4", len(got.History))
	}
	for i, f := range got.History {
		wantStdout := fmt.Sprintf("frame %d\n", i)
		if f.Stdout != wantStdout {
			t.Errorf("frame %d stdout=%q want %q", i, f.Stdout, wantStdout)
		}
	}
}

// A frame larger than bufio.Scanner's 64KB token cap round-trips because Load calls
// scanner.Buffer with maxLineSize (256 MB).
func TestRecordingLargeFrameRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")
	big := strings.Repeat("kubernetes is large\n", 5000) // > 64KB

	s := session.NewSession("x", time.Second)
	mustStartJSONL(t, s, path)
	mustRecord(t, s, session.Execution{Timestamp: time.Now().UTC(), Stdout: big})
	if err := s.StopRecording(); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.History) != 1 {
		t.Fatalf("History len=%d want 1", len(got.History))
	}
	if got.History[0].Stdout != big {
		t.Errorf("large stdout did not round-trip (lens %d vs %d)", len(got.History[0].Stdout), len(big))
	}
}

// A crash-truncated final line is tolerated: Load returns every fully-decoded frame and no
// error.
func TestRecordingTruncatedTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")

	s := session.NewSession("x", time.Second)
	mustStartJSONL(t, s, path)
	mustRecord(t, s, session.Execution{Timestamp: time.Now().UTC(), Stdout: "first\n"})
	mustRecord(t, s, session.Execution{Timestamp: time.Now().UTC(), Stdout: "second\n"})
	if err := s.StopRecording(); err != nil {
		t.Fatal(err)
	}

	// Append a partial JSON object (no closing brace, no newline).
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"ts":"2026-`); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err == nil {
		t.Fatal("Load should report the skipped corrupt tail as a non-fatal warning")
	}
	if got == nil {
		t.Fatalf("Load returned nil session despite recoverable error: %v", err)
	}
	if len(got.History) != 2 {
		t.Errorf("History len=%d want 2 (truncated tail should be dropped)", len(got.History))
	}
}

// NewJSONLRecorder opens the file with O_EXCL, so a second call at the same path refuses
// with os.ErrExist rather than clobbering the existing recording.
func TestRecordingRefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")

	first := session.NewSession("first", time.Second)
	mustStartJSONL(t, first, path)
	mustRecord(t, first, session.Execution{Timestamp: time.Now().UTC(), Stdout: "FIRST\n"})
	if err := first.StopRecording(); err != nil {
		t.Fatal(err)
	}

	_, err := NewJSONLRecorder(path)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("second NewJSONLRecorder err = %v, want os.ErrExist", err)
	}

	// First recording should still be readable, untouched.
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Command != "first" {
		t.Errorf("Command=%q want %q (first recording must survive)", got.Command, "first")
	}
	if len(got.History) != 1 || got.History[0].Stdout != "FIRST\n" {
		t.Errorf("History=%+v want one FIRST frame", got.History)
	}
}

// Sanity: the encoded file is JSONL (one well-formed JSON value per line, no extra escapes).
func TestRecordingIsJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wch.jsonl")
	s := session.NewSession("kubectl", time.Second)
	mustStartJSONL(t, s, path)
	mustRecord(t, s, session.Execution{Timestamp: time.Now().UTC(), Stdout: "hi\n"})
	if err := s.StopRecording(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 frame), got %d:\n%s", len(lines), raw)
	}
	for i, line := range lines {
		var v map[string]any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Errorf("line %d is not valid JSON: %v\nraw=%q", i, err, line)
		}
	}
}
