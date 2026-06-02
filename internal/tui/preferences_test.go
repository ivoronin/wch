package tui

import (
	"testing"
	"time"
)

// TestPreferencesDefaults pins that New(Config{...}) copies the cfg-derived
// preferences into m.prefs and that the runtime-only flags (Paused,
// HelpVisible) start false. A future Config field that forgets to populate
// prefs would break this test.
func TestPreferencesDefaults(t *testing.T) {
	cfg := Config{
		Command:        "x",
		Interval:       time.Second,
		DiffEnabled:    true,
		ShowStatus:     true,
		NotifyOnChange: true,
	}
	m := New(cfg)

	want := Preferences{
		Diff:        true,
		StatusBar:   true,
		OSNotify:    true,
		Paused:      false,
		HelpVisible: false,
	}
	// NOTE: switch to reflect.DeepEqual or per-field asserts when adding non-comparable fields.
	if m.prefs != want {
		t.Errorf("New prefs = %+v, want %+v", m.prefs, want)
	}
}
