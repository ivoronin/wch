package tui

import (
	"errors"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/recording"
	"github.com/ivoronin/wch/internal/tui/notify"
)

const (
	recordPromptLabel    = "Record to: "
	recordStartedMessage = "Recording started"
	recordStoppedMessage = "Recording stopped"
)

// autoStartRecordingMsg fires once during Init when AutoStart was non-nil: a deferred
// flow.Start so the failure path (rare; the CLI already verified the file doesn't exist)
// can surface a warning bubble instead of crashing the program.
type autoStartRecordingMsg struct{ path string }

// newRecordInput builds the textinput for the record-filename prompt: bar-matched palette,
// branded prompt label, value pre-filled, cursor parked at the end so Enter accepts the
// default and the user can backspace into the command portion.
func newRecordInput(initial string) textinput.Model {
	in := newBarInput(recordPromptLabel)
	in.SetValue(initial)
	in.SetCursor(len(initial))
	return in
}

// startRecording asks Flow to begin a recording and translates the outcome into a
// notification bubble. ok reports whether the recording is now active. Shared by the
// auto-start launch path and the interactive record-filename submit.
func (m Model) startRecording(path string) (Model, tea.Cmd, bool) {
	err := m.flow.Start(path)
	switch {
	case err == nil:
		m2, cmd := m.push(notify.LevelInfo, recordStartedMessage)
		return m2, cmd, true
	case errors.Is(err, recording.ErrPathExists):
		m2, cmd := m.push(notify.LevelWarning, "File exists: "+path)
		return m2, cmd, false
	default:
		m2, cmd := m.push(notify.LevelWarning, "Recording error: "+err.Error())
		return m2, cmd, false
	}
}

// toggleRecord is the 'r'-key shared handler for view and picker. Recording in progress →
// stop + notification; idle → open the filename input. Replay no-ops defensively (the key
// is not advertised in replay help either).
func (m Model) toggleRecord(from state) (Model, state, tea.Cmd) {
	if !m.isLive() {
		return m, from, nil
	}
	if m.flow.IsActive() {
		var cmd tea.Cmd
		if err := m.flow.Stop(); err != nil {
			m, cmd = m.push(notify.LevelWarning, "Recording stop error: "+err.Error())
		} else {
			m, cmd = m.push(notify.LevelInfo, recordStoppedMessage)
		}
		return m, from, cmd
	}
	return m.openInput(from, newRecordInput(recording.DefaultFilename(m.session.Command, time.Now())), applyRecordSubmit)
}

// applyRecordSubmit validates the typed path and either starts recording (popping back to
// the predecessor) or flashes a warning while keeping the input open with the user's value
// intact for them to correct.
func applyRecordSubmit(m Model, s inputState) (Model, state, tea.Cmd) {
	path, err := recording.NormalizePath(s.input.Value())
	if err != nil {
		m2, cmd := m.push(notify.LevelWarning, "Empty or invalid path")
		return m2, s, cmd
	}
	m, cmd, ok := m.startRecording(path)
	if !ok {
		return m, s, cmd
	}
	return m, s.prev, cmd
}
