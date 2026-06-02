package tui

// Preferences are the runtime toggles the user flips during a session. Lives
// as a single nested field on Model (m.prefs) so all read/write sites mention
// the same prefix and the toggle set is visible at one declaration.
//
// CLI-derived preferences (Diff, StatusBar, OSNotify) are populated from
// Config in New / NewReplay. Runtime-only toggles (Paused, HelpVisible)
// default to false.
type Preferences struct {
	Diff      bool // toggled by 'd'; controls renderFrame's diff overlay
	StatusBar bool // toggled by 't'; user side of barShown's OR with state.ShowsBar
	// OSNotify gates the OSC9 ping on exec changes; set via -b at launch. Named for
	// the OSC9 channel, not the trigger; cfg.NotifyOnChange maps here.
	OSNotify    bool
	Paused      bool // toggled by 'p'; suppresses tick-driven execution
	HelpVisible bool // toggled by 'h'; gates the help overlay in View
}
