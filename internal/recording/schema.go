// Package recording owns the on-disk recording format and the recording lifecycle.
// The session.Recorder port lives in internal/session; the adapters (JSONL on disk and
// in-memory) live here, alongside Flow — the single entry point CLI and TUI use to
// start, stop, and inspect a recording.
package recording

import (
	"time"

	"github.com/ivoronin/wch/internal/session"
)

// FormatTag identifies a wch-history file. Every recording begins with a header line
// containing this tag.
const FormatTag = "wch-history"

// SupportedVersion is the only file-format version this build accepts.
const SupportedVersion = 1

// Header is the first JSONL line of every wch-history recording.
type Header struct {
	Format   string `json:"format"`
	Version  int    `json:"version"`
	Command  string `json:"command"`
	Interval string `json:"interval"`
}

// Frame is one captured execution as it sits on disk.
type Frame struct {
	Ts     time.Time `json:"ts"`
	Exit   int       `json:"exit"`
	Stdout string    `json:"stdout"`
	Stderr string    `json:"stderr,omitempty"`
	Error  string    `json:"error,omitempty"`
}

// frameFrom converts a session.Execution to a Frame. The conversion lives in the
// recording package because session does not know about the on-disk schema; both
// JSONLRecorder and InMemoryRecorder use it.
func frameFrom(e session.Execution) Frame {
	errMsg := ""
	if e.Error != nil {
		errMsg = e.Error.Error()
	}
	return Frame{
		Ts:     e.Timestamp,
		Exit:   e.ExitCode,
		Stdout: e.Stdout,
		Stderr: e.Stderr,
		Error:  errMsg,
	}
}
