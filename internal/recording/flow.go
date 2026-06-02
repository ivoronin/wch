package recording

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ivoronin/wch/internal/pathutil"
	"github.com/ivoronin/wch/internal/session"
)

// ErrPathExists is returned by PreflightCheck (and by Flow.Start, wrapping the underlying
// os.ErrExist from the recorder factory) when a recording target path already exists.
// Detect with errors.Is.
var ErrPathExists = errors.New("recording: path exists")

// AutoStartRequest signals a desire to begin recording immediately on TUI startup -- the
// typed alternative to the empty-string-means-no convention. nil means no request.
// Built by the CLI from the validated `-w` flag value; passed through tui.Config; the TUI
// fires a deferred message at Init time that calls Flow.Start(Path).
type AutoStartRequest struct {
	Path string
}

// maxSanitizedCommandLen caps the command-derived portion of a default recording filename
// so the full name (command + timestamp + extension) stays well under typical FS limits.
const maxSanitizedCommandLen = 80

// NormalizePath trims input whitespace, expands a leading "~" or "~/" via the user's
// home directory, and rejects empty or whitespace-only input. The only string-shape
// canonicalization recording does -- no Clean, no Abs.
func NormalizePath(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	expanded, err := pathutil.ExpandTilde(trimmed)
	if err != nil {
		return "", err
	}
	if expanded == "" {
		return "", errors.New("recording: empty path")
	}
	return expanded, nil
}

// PreflightCheck reports whether a recording can be created at path. Today the only
// failure mode is ErrPathExists -- JSONLRecorder's O_EXCL open is the atomic guard,
// but the CLI uses PreflightCheck to refuse early (before launching the TUI) with a
// clear stderr message. Returns nil when the path is available.
func PreflightCheck(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrPathExists, path)
	}
	return nil
}

// DefaultFilename builds a CWD-relative filename from the watched command and a moment
// in time, e.g. "kubectl_get_pods_A_20260530-153045.wch.jsonl".
func DefaultFilename(command string, now time.Time) string {
	base := sanitizeCommand(command)
	if base == "" {
		base = "wch"
	}
	return base + "_" + now.Format("20060102-150405") + ".wch.jsonl"
}

// sanitizeCommand collapses runs of non-alphanumerics into a single underscore, trims
// leading/trailing underscores, and caps the result at maxSanitizedCommandLen.
func sanitizeCommand(s string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) > maxSanitizedCommandLen {
		out = strings.TrimRight(out[:maxSanitizedCommandLen], "_")
	}
	return out
}

// Flow is the single entry point CLI and TUI use to drive a recording. It owns adapter
// construction and error classification; the underlying Session keeps the active Recorder
// and writes frames through RecordIfChanged.
//
// The factory field is unexported and accessible only to in-package tests via newFlowWith
// -- production callers use New, which wires the JSONLRecorder factory.
type Flow struct {
	session *session.Session
	factory func(path string) (session.Recorder, error)
}

// New constructs a Flow that opens JSONL recordings on disk.
func New(s *session.Session) *Flow {
	return &Flow{
		session: s,
		factory: func(path string) (session.Recorder, error) { return NewJSONLRecorder(path) },
	}
}

// Start opens a recorder at path and arms the session. Returns an error classified via
// errors.Is(err, ErrPathExists) for the "file exists" case. The IsActive guard precedes
// the factory call so an already-active flow never creates an orphan file that would
// then block a future Start with os.ErrExist.
func (f *Flow) Start(path string) error {
	if f.IsActive() {
		return errors.New("recording: already in progress")
	}
	rec, err := f.factory(path)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", ErrPathExists, path)
		}
		return err
	}
	if err := f.session.StartRecording(rec); err != nil {
		_ = rec.Close()
		return err
	}
	return nil
}

// Stop closes the active recording, if any. Idempotent -- safe to call when nothing is
// recording.
func (f *Flow) Stop() error {
	return f.session.StopRecording()
}

// IsActive reports whether a recording is currently in progress. Reads through to the
// session so it stays correct after RecordIfChanged auto-finalizes on a write error.
func (f *Flow) IsActive() bool {
	return f.session.IsRecording()
}
