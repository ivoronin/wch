package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivoronin/wch/internal/tui"
)

var version = "dev"

func main() {
	interval := flag.Duration("i", time.Second, "refresh interval")
	disableDiff := flag.Bool("d", false, "disable diff highlighting")
	hideStatus := flag.Bool("t", false, "hide status bar")
	disableNotify := flag.Bool("b", false, "disable terminal notification on change")
	showVersion := flag.Bool("version", false, "show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wch [flags] <command>\n\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("wch %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: command required")
		flag.Usage()
		os.Exit(1)
	}

	command := strings.Join(args, " ")

	cfg := tui.Config{
		Command:        command,
		Interval:       *interval,
		DiffEnabled:    !*disableDiff,
		ShowStatus:     !*hideStatus,
		NotifyOnChange: !*disableNotify,
	}

	model := tui.New(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
