We are building a CLI tool to wrap claude code/tmux. It needs to be super lightweight, easy to install, minimal, out of your way, and the core feature is to simply provide an overview of currently running claude sessions and a way to attach/kill/create sessions. bonus points to detect the state of the session so we can add status indicators. here is some more context, all ideas, nothing set in stone: 

Think of it as:

tmux session manager
+ claude process supervisor
+ state detector
+ TUI dashboard

Instead of:

tmux list-sessions
attach manually
wonder if agent is done

You’d get:

┌─────────────────────────────────────┐
│  Claude Sessions                    │
├─────────────────────────────────────┤
│ ● api-refactor        🟡 Working    │
│ ● resume-builder      🔴 Needs input│
│ ● lca-bushings        🟢 Completed  │
│ ● random-scratch      ⚪ Idle       │
└─────────────────────────────────────┘

Then:

Enter → attach to session

n → new session

k → kill

auto notifications when 🔴


🔧 The Big Technical Question

Can you reliably detect Claude Code state?

Yes — if you control how it's launched.

Instead of running:

tmux new -s api-refactor
claude

You’d launch via wrapper:

cctv new api-refactor

That wrapper:

Creates tmux session

Starts claude inside a PTY

Pipes stdout through a small state detector

Writes status to a session metadata file

Updates dashboard in real-time

Now you own observability.

🧩 How To Detect Agent State

You define state rules like:

🟡 Working

Streaming tokens

Recent stdout activity

🔴 Needs Input

Prompt indicator appears

Process blocked on stdin

Specific Claude CLI prompt markers

🟢 Completed

Output finished

No streaming for X seconds

Last line matches completion pattern

⚪ Idle

No claude process running

This is far easier if you wrap the agent process instead of attaching blindly.

🧱 Architecture
Layer 1 — Session Manager

Uses:

tmux list-sessions

tmux has-session

tmux new-session

tmux kill-session

tmux attach

You don’t need to reimplement multiplexing.
You leverage tmux.

Layer 2 — Agent Supervisor

Instead of running claude directly:

You run:

cctv

This:

Spawns claude via PTY

Observes output

Writes JSON state:

~/.cc-admin/sessions/api-refactor.json

Example:

{
  "status": "needs_input",
  "last_activity": 1708552001,
  "pid": 12345
}
Layer 3 — TUI Dashboard

Now the fun part.

Best TUI stacks:
🥇 Go + Bubble Tea (cleanest)

bubbletea

lipgloss

bubbles

insane ecosystem

compiles to single binary

very “serious dev tool” vibes

🥈 Python + Textual

faster to prototype

more UI-ish

heavier

🥉 Node + Ink

React in terminal

cool but less stable for system tools

If you want this to feel legit, Go + Bubble Tea is perfect.

🧠 Important: Don’t Fight tmux

You don’t replace tmux.
You orchestrate it.

Your tool is:

A control plane for agent sessions.

tmux remains the data plane.

That’s a powerful separation.

🟢 Can You Build It Using Claude Code Itself?

Yes — and that’s actually meta-brilliant.

You can:

Define architecture

Ask Claude to scaffold the Go project

Ask it to implement PTY wrapper

Ask it to build Bubble Tea list view

Iterate inside one of the sessions

This is the exact kind of structured system project coding agents are good at.

🧭 The Hard Parts

Reliable state detection

Handling PTY edge cases

Avoiding weird buffering behavior

Graceful shutdown handling

All solvable.

💡 A Simpler V1

You don’t even need PTY introspection initially.

You could:

Run Claude in each tmux session

Periodically inspect session output using:

tmux capture-pane -pt api-refactor

Then:

Parse last ~50 lines

Detect state heuristically

That avoids deep process wrapping entirely.

It’s less elegant.
But much simpler.
