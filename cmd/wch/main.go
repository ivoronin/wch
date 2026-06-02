package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ivoronin/wch/internal/recording"
	"github.com/ivoronin/wch/internal/tui"
)

var version = "dev"

func main() {
	interval := flag.Duration("i", time.Second, "refresh interval")
	historyLimit := flag.Int("l", 86400, "history limit (executions retained in memory; 0 = unlimited)")
	disableDiff := flag.Bool("d", false, "disable diff highlighting")
	hideStatus := flag.Bool("t", false, "hide status bar")
	enableNotify := flag.Bool("b", false, "enable terminal notification on change")
	openPath := flag.String("r", "", "read a recorded session in replay mode (offline)")
	writePath := flag.String("w", "", "write a recording to <path> (started immediately; file must not already exist)")
	showVersion := flag.Bool("version", false, "show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wch [flags] <command>\n       wch -r <file>\n\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("wch %s\n", version)
		os.Exit(0)
	}

	if *openPath != "" && *writePath != "" {
		fmt.Fprintln(os.Stderr, "Error: -r and -w are mutually exclusive")
		flag.Usage()
		os.Exit(1)
	}

	var model tea.Model

	if *openPath != "" {
		if len(flag.Args()) > 0 {
			fmt.Fprintln(os.Stderr, "Error: -r is exclusive with a command")
			flag.Usage()
			os.Exit(1)
		}
		s, err := recording.Load(*openPath)
		if err != nil {
			if s == nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			// Partial load (corrupt/truncated lines): surface as a warning but proceed
			// with the frames that did decode.
			fmt.Fprintf(os.Stderr, "wch: warning: %v\n", err)
		}
		model = tui.NewReplay(tui.Config{
			Command:     s.Command,
			Interval:    s.Interval,
			DiffEnabled: !*disableDiff,
			ShowStatus:  !*hideStatus,
		}, s)
	} else {
		args := flag.Args()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: command required")
			flag.Usage()
			os.Exit(1)
		}
		var autoStart *recording.AutoStartRequest
		if *writePath != "" {
			p, err := recording.NormalizePath(*writePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			// Best-effort fast-fail before launching the TUI. The atomic guard against
			// clobber is JSONLRecorder's O_EXCL open; a race here still surfaces as an
			// in-TUI warning rather than data loss.
			if err := recording.PreflightCheck(p); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			autoStart = &recording.AutoStartRequest{Path: p}
		}
		command := strings.Join(args, " ")
		model = tui.New(tui.Config{
			Command:        command,
			Interval:       *interval,
			DiffEnabled:    !*disableDiff,
			ShowStatus:     !*hideStatus,
			NotifyOnChange: *enableNotify,
			AutoStart:      autoStart,
			MaxHistory:     *historyLimit,
		})
	}

	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	// Bubble Tea v2 short-circuits Model.Update on QuitMsg, so the TUI cannot finalise its
	// own recording on quit. Cleanup runs unconditionally — even when p.Run returned an
	// error — so an active recording is closed before we exit. Cleanup is idempotent
	// (no-op when no recording is active).
	var cleanupErr error
	if m, ok := finalModel.(tui.Model); ok {
		cleanupErr = m.Cleanup()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if cleanupErr != nil {
		fmt.Fprintf(os.Stderr, "wch: recording: %v\n", cleanupErr)
		os.Exit(1)
	}
}
