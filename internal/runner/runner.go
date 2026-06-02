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

// maxOutputBytes caps the captured stdout/stderr per stream so a runaway streaming command
// (e.g. `journalctl -f`, `cat /dev/urandom`) cannot exhaust memory through the history.
// 4 MB comfortably absorbs typical kubectl/ps/etc. output while bounding worst-case usage.
// Excess bytes are silently dropped — no marker is inserted into the captured stream
// because the same buffer ends up in the recording file, where a synthetic suffix would be
// indistinguishable from real output to a later replay or downstream consumer.
const maxOutputBytes = 4 * 1024 * 1024

// limitedBuffer is a bytes.Buffer that stops growing past maxOutputBytes, dropping the
// excess. exec.Cmd treats a short write as a failure that aborts the command, so Write must
// always report n == len(p) to keep the process running even after we've stopped recording.
// String is promoted from the embedded buffer.
type limitedBuffer struct {
	bytes.Buffer
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if space := maxOutputBytes - l.Buffer.Len(); space > 0 {
		l.Buffer.Write(p[:min(len(p), space)])
	}
	return len(p), nil
}

// Execute runs the command and returns the result. Timestamp is the start time of the
// invocation (when wch decided to run the command), not the finish time — finish-time
// stamps drift further from "what wch did" the slower the command is.
func (r *Runner) Execute(ctx context.Context) session.Execution {
	cmd := exec.CommandContext(ctx, "sh", "-c", r.command)

	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	result := session.Execution{Timestamp: time.Now()}
	err := cmd.Run()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

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
