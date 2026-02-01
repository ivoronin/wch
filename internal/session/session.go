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

// AddExecution adds an execution to history
func (s *Session) AddExecution(exec Execution) {
	s.History = append(s.History, exec)
	if s.MaxHistory > 0 && len(s.History) > s.MaxHistory {
		s.History = s.History[1:]
	}
}

// LastExecution returns the most recent execution, or nil if none
func (s *Session) LastExecution() *Execution {
	if len(s.History) == 0 {
		return nil
	}
	return &s.History[len(s.History)-1]
}
