package runner

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

const errorExitCode = -1 // Used when error is not an ExitError

// Runner executes commands
type Runner struct {
	command string
}

// New creates a new runner
func New(command string) *Runner {
	return &Runner{
		command: command,
	}
}

// Execute runs the command and returns the result
func (r *Runner) Execute(ctx context.Context) session.Execution {
	cmd := exec.CommandContext(ctx, "sh", "-c", r.command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := session.Execution{
		Timestamp: time.Now(),
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
	}

	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = errorExitCode
		}
	}

	return result
}
