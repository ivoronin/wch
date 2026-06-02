package recording

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

// maxLineSize caps a single JSONL line. 256 MB is generous enough to load recordings made
// before the runner's per-stream cap landed, while bounding the buffer's worst-case growth.
const maxLineSize = 256 * 1024 * 1024

// Load reads a wch-history JSONL file into a Session. The header is validated for both
// format tag and version; mismatch returns a typed error so older binaries fail loud rather
// than silently mis-parsing a future format.
//
// Frame decoding is line-based and recoverable: a single malformed line (mid-stream corruption
// or a crash-truncated trailing line) is skipped so any fully-decoded frames after it still
// load. The skipped-line count and any I/O error are surfaced via the returned error so the
// caller can warn — Load still returns a non-nil Session in that case, signalling "partial
// load, here's what survived" rather than "load failed".
//
// The returned Session is not armed for recording — replay never persists.
func Load(path string) (*session.Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("recording: read header: %w", err)
		}
		return nil, errors.New("recording: empty file")
	}
	var header Header
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("recording: invalid header: %w", err)
	}
	if header.Format != FormatTag {
		return nil, fmt.Errorf("recording: unknown format %q (expected %q)", header.Format, FormatTag)
	}
	if header.Version != SupportedVersion {
		return nil, fmt.Errorf("recording: unsupported version %d (this build supports %d)", header.Version, SupportedVersion)
	}
	interval, err := time.ParseDuration(header.Interval)
	if err != nil {
		return nil, fmt.Errorf("recording: invalid interval %q: %w", header.Interval, err)
	}
	s := session.NewSession(header.Command, interval)
	var skipped int
	for scanner.Scan() {
		var frame Frame
		if err := json.Unmarshal(scanner.Bytes(), &frame); err != nil {
			skipped++
			continue
		}
		s.History = append(s.History, executionFrom(frame))
	}
	if err := scanner.Err(); err != nil {
		return s, fmt.Errorf("recording: read: %w", err)
	}
	if skipped > 0 {
		return s, fmt.Errorf("recording: skipped %d corrupt frame(s)", skipped)
	}
	return s, nil
}

// executionFrom converts a Frame back to a session.Execution. The inverse of frameFrom
// (in schema.go).
func executionFrom(f Frame) session.Execution {
	var err error
	if f.Error != "" {
		err = errors.New(f.Error)
	}
	return session.Execution{
		Timestamp: f.Ts,
		Stdout:    f.Stdout,
		Stderr:    f.Stderr,
		ExitCode:  f.Exit,
		Error:     err,
	}
}
