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

wch runs a command periodically and displays the output in a scrollable terminal UI. Unlike standard `watch(1)`, you can scroll through the output when it exceeds the terminal height. Changed lines are highlighted using character-level diff, making it easy to spot what changed between executions.

Press `b` to enter history browser and navigate through up to 1000 past executions using a timeline picker. Only executions where output actually changed are recorded, so you get meaningful history without noise.

## Features

- Scrollable view for output that exceeds terminal height (unlike `watch(1)`)
- History browser stores up to 1000 executions, navigate with arrow keys or h/l
- Character-level diff highlighting between executions
- Terminal notifications on output change (OSC 9, supported by iTerm2 and others)
- Vim-style navigation (hjkl, g/G, pgup/pgdown)
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
wch kubectl get pods                    # watch with 1s interval
wch -i 5s kubectl get pods              # 5 second interval
wch -d kubectl get pods                 # disable diff highlighting
wch -t kubectl get pods                 # hide status bar
wch -b kubectl get pods                 # disable notifications
```

## Configuration

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-i` | Refresh interval | `1s` |
| `-d` | Disable diff highlighting | `false` |
| `-t` | Hide status bar | `false` |
| `-b` | Disable notifications | `false` |

## License

[GPL-3.0](LICENSE)
