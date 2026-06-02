# wch

Watch command output with history you can rewind

[![CI](https://github.com/ivoronin/wch/actions/workflows/test.yml/badge.svg)](https://github.com/ivoronin/wch/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/ivoronin/wch)](https://github.com/ivoronin/wch/releases)

## Table of Contents

[Overview](#overview) · [Features](#features) · [Installation](#installation) · [Usage](#usage) · [Configuration](#configuration) · [License](#license)

```bash
# Instead of:
watch -n 1 kubectl get pods

# Use:
wch kubectl get pods
```

## Overview

wch runs a command on an interval and displays the scrollable output. Two design choices set it apart from `watch(1)` and other modern replacements. Rows in the output are matched across refreshes by identity (a stable token like a pod `NAME`) instead of exact line equality, so the line you're reading stays anchored in place when rows insert above the viewport, and a row whose `AGE` ticks every refresh reads as one cell change rather than a delete + insert. The TUI itself shows only the command's output and a single status line — no border, no line numbers, no help banner, no config file, no keymap rebinding, no theme to pick (light/dark is detected from the terminal background at startup).

## Features

- Scroll position anchored to content (row identity, not line offset)
- Minimal UI surface (no border, line numbers, help banner, config file, keymap rebinding; theme auto-detected)
- Word-level diff highlighting between executions, tolerant of volatile fields (`AGE`, `RESTARTS`) so a row whose value ticks each refresh doesn't read as a delete + insert
- History keeps up to `-l` past executions (default 86400 ≈ 24h at 1s interval; `-l 0` for unlimited), navigable with arrow keys
- Record sessions to a JSONL file (`-w <path>`) and replay them offline with full history navigation (`-r <file>`)
- Scrollable view for output that exceeds terminal height (unlike `watch(1)`)
- Terminal notifications on output change (OSC 9, supported by iTerm2 and others)
- Keyboard navigation (arrow keys, PgUp/PgDn, Home/End)
- Pause/resume execution
- Toggleable status bar and diff highlighting
- Horizontal scrolling for wide output
- Configurable refresh interval

## Installation

### GitHub Releases

Download from [Releases](https://github.com/ivoronin/wch/releases).

### Homebrew

```bash
brew install ivoronin/ivoronin/wch
```

## Usage

### Basic

```bash
wch kubectl get pods                          # watch with 1s interval
wch -i 5s kubectl get pods                    # 5 second interval
wch -d kubectl get pods                       # disable diff highlighting
wch -t kubectl get pods                       # hide status bar
wch -b kubectl get pods                       # enable notifications
wch -w session.wch.jsonl kubectl get pods     # record session while watching
wch -r session.wch.jsonl                      # replay recorded session offline
```

## Configuration

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-i` | Refresh interval | `1s` |
| `-d` | Disable diff highlighting | `false` |
| `-t` | Hide status bar | `false` |
| `-b` | Enable notifications | `false` |
| `-w` | Write recording to path (must not exist) | — |
| `-r` | Read a recorded session (offline replay) | — |

## License

[GPL-3.0](LICENSE)
