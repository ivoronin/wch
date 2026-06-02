package recording

import (
	"encoding/json"
	"os"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

// JSONLRecorder is the on-disk session.Recorder: one JSON value per line, header first.
// The file is opened with O_EXCL — refusing to clobber an existing recording.
type JSONLRecorder struct {
	path string
	f    *os.File
	enc  *json.Encoder
}

// NewJSONLRecorder opens path with O_EXCL, ready for Initialize to be called next.
// On os.ErrExist the caller (Flow.Start) is responsible for the user-facing message.
func NewJSONLRecorder(path string) (*JSONLRecorder, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return &JSONLRecorder{path: path, f: f, enc: enc}, nil
}

// Initialize writes the header followed by every backlog frame. On any write error
// during initialization the file is closed AND removed — otherwise the orphan would
// block a same-path retry under O_EXCL until the user manually deletes it.
func (r *JSONLRecorder) Initialize(command string, interval time.Duration, backlog []session.Execution) error {
	if err := r.enc.Encode(Header{
		Format:   FormatTag,
		Version:  SupportedVersion,
		Command:  command,
		Interval: interval.String(),
	}); err != nil {
		r.abortAndRemove()
		return err
	}
	for _, e := range backlog {
		if err := r.enc.Encode(frameFrom(e)); err != nil {
			r.abortAndRemove()
			return err
		}
	}
	return nil
}

// WriteFrame persists one novel execution.
func (r *JSONLRecorder) WriteFrame(exec session.Execution) error {
	return r.enc.Encode(frameFrom(exec))
}

// Close releases the file handle. Idempotent.
func (r *JSONLRecorder) Close() error {
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}

// abortAndRemove closes the file, zeroes the handle (so Close is a true no-op after this),
// and unlinks the path. Used by Initialize on any write failure so the O_EXCL guard doesn't
// stay stuck on an orphan and so a subsequent Close call from the caller's error path is a
// clean no-op.
func (r *JSONLRecorder) abortAndRemove() {
	if r.f == nil {
		return
	}
	_ = r.f.Close()
	r.f = nil
	_ = os.Remove(r.path)
}

// Compile-time guarantee that JSONLRecorder satisfies session.Recorder.
var _ session.Recorder = (*JSONLRecorder)(nil)
