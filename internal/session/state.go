package session

import (
	"os/exec"
	"strings"
)

type State int

const (
	StateIdle    State = iota
	StateWorking       // claude running, actively streaming ("esc to interrupt" visible)
	StateWaiting       // claude running, finished turn, needs user reply
)

func (s State) String() string {
	switch s {
	case StateWorking:
		return "Working"
	case StateWaiting:
		return "Waiting"
	default:
		return "Idle"
	}
}

// shells are foreground commands that mean "no agent running".
var shells = map[string]bool{
	"bash": true, "zsh": true, "sh": true, "fish": true,
	"dash": true, "csh": true, "tcsh": true, "ksh": true,
	"nu": true,
}

func DetectState(sessionName string) State {
	if err := exec.Command("tmux", "has-session", "-t", sessionName).Run(); err != nil {
		return StateIdle
	}

	// Primary gate: if a shell is in the foreground, no agent is running.
	cmdOut, err := exec.Command(
		"tmux", "display-message", "-t", sessionName, "-p", "#{pane_current_command}",
	).Output()
	if err != nil {
		return StateIdle
	}
	if shells[strings.ToLower(strings.TrimSpace(string(cmdOut)))] {
		return StateIdle
	}

	// Claude (or node) is running. Check the VISIBLE screen only (no -S scrollback)
	// for the "esc to interrupt" hint Claude Code renders while streaming.
	// Without -S, capture-pane returns only what's currently on screen, so
	// once Claude finishes and redraws the input prompt, this text is gone.
	pane, err := exec.Command("tmux", "capture-pane", "-pt", sessionName).Output()
	if err != nil {
		return StateIdle
	}

	for _, line := range strings.Split(string(pane), "\n") {
		if strings.Contains(line, "esc to interrupt") {
			return StateWorking
		}
	}

	// Claude is running but not actively streaming → waiting for user reply.
	return StateWaiting
}
