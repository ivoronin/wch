package session

import "time"

// Recorder is the persistence port a Session writes to when armed for recording.
// Adapters live in internal/recording: JSONLRecorder for the on-disk format, and
// InMemoryRecorder for tests and general in-process callers. Outside this package,
// callers should drive recording through recording.Flow rather than this port directly.
type Recorder interface {
	// Initialize is called exactly once when the Session arms for recording. The adapter
	// is responsible for writing any header it needs and for persisting the supplied
	// backlog (every execution already in History at the moment recording begins).
	Initialize(command string, interval time.Duration, backlog []Execution) error

	// WriteFrame persists one novel execution. Called for every execution
	// RecordIfChanged accepted into History after Initialize has run.
	WriteFrame(exec Execution) error

	// Close releases any resources the adapter holds. Idempotent.
	Close() error
}
