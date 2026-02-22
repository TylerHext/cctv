# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**cctv** is a lightweight CLI/TUI tool for managing Claude Code sessions inside tmux. It wraps tmux session management with agent state detection and a Bubble Tea dashboard.

The tool is invoked as `cctv` — short for "Claude Code TV" (a control plane for watching your agents).

## Tech Stack

- **Language**: Go
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles)
- **Multiplexing**: tmux (external dependency, not reimplemented)

## Commands

```bash
go build -o cctv ./cmd/cctv     # Build binary
go run ./cmd/cctv               # Run without building
go test ./...                   # Run all tests
```

## Architecture

Two-layer design (V1):

### Layer 1 — Session Manager (`internal/session/`)
Thin wrapper around tmux CLI commands plus inline state detection:
- `tmux list-sessions` — enumerate sessions
- `tmux new-session -d -s <name>` — create detached session
- `tmux kill-session -t <name>` — kill session
- `tmux attach-session -t <name>` — attach; `cmd.Run()` blocks until detach, then returns
- `tmux capture-pane` — used for both state detection and the live preview pane

**State detection heuristics** (`internal/session/state.go`, called per-session on each poll):
1. `StateIdle` — tmux session doesn't exist, or foreground process is a shell
2. `StateWorking` — "esc to interrupt" visible in current pane screen (claude is streaming)
3. `StateWaiting` — a non-shell process is running but "esc to interrupt" is absent (claude waiting for user reply)

State detection runs live on each 2-second TUI poll. No JSON files are written to disk.

> **V2 consideration**: a `internal/supervisor/` background process writing
> `~/.cctv/sessions/<name>.json` would allow state to be read cheaply and
> support cross-process notification. Deferred from V1 in favor of simpler
> inline polling.

### Layer 2 — TUI Dashboard (`internal/tui/`)
Bubble Tea model that:
- Polls session list + state on a 2-second timer
- Renders a session list with status badges (Working / Waiting / Idle)
- Shows a live tmux pane preview in split view (toggled with Tab)
- After attach (tmux detach returns), automatically restarts the TUI

**Keybindings:**
| Key | Action |
|-----|--------|
| `Enter` | Attach to selected session |
| `n` | New session (prompts for name) |
| `x` | Kill selected session (confirm with y) |
| `q` / `Ctrl+C` | Quit |
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `Tab` | Toggle split preview pane |

## Project Structure

```
cmd/cctv/main.go          # Entry point — parses flags, routes to TUI or subcommand
internal/
  config/                 # Paths, constants; optional TOML config (~/.cctv/config.toml)
  session/                # tmux wrappers + DetectState heuristics
  tui/                    # Bubble Tea model and views
```

## Configuration

Optional TOML file at `~/.cctv/config.toml`. If absent, defaults are used.

```toml
[new_session]
layout   = "even-horizontal"   # tmux layout applied to new sessions
commands = ["claude"]          # commands sent to new session on creation
```

## Key Design Constraints

- **Don't fight tmux**: cctv orchestrates tmux, not replaces it. Attach by exec-ing `tmux attach-session` and waiting for it to return on detach.
- **Single binary, no daemons**: state is detected live via `tmux capture-pane`; the TUI polls every 2 seconds.
- **V1 simplicity**: inline polling beats a background supervisor for the common case. PTY wrapping and JSON state files are V2 concerns.
- **Notifications** (V2): when a session transitions to `Waiting`, surface it via `osascript` (macOS) or `notify-send` (Linux).
