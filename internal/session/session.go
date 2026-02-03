package session

import "time"

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

// Session holds execution history
type Session struct {
	Command    string
	Interval   time.Duration
	History    []Execution
	MaxHistory int
}

// NewSession creates a new session
func NewSession(command string, interval time.Duration) *Session {
	return &Session{
		Command:  command,
		Interval: interval,
	}
}

// RecordIfChanged adds an execution to history only if output changed.
// Returns true if recorded, false if skipped (no change).
func (s *Session) RecordIfChanged(exec Execution) bool {
	// Always add first execution
	if len(s.History) == 0 {
		s.History = append(s.History, exec)
		return true
	}

	// Only add if output changed
	last := s.History[len(s.History)-1]
	if exec.Output() == last.Output() {
		return false // No change, don't store
	}

	s.History = append(s.History, exec)
	if s.MaxHistory > 0 && len(s.History) > s.MaxHistory {
		s.History = s.History[1:]
	}
	return true
}
