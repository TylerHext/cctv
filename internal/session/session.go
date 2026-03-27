package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tylerhext/cctv/internal/config"
)

type Session struct {
	Name   string
	Status State
}

// List returns all tmux sessions enriched with detected state.
func List() ([]Session, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		// tmux exits non-zero when there are no sessions — treat as empty.
		return nil, nil
	}

	var sessions []Session
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		sessions = append(sessions, Session{
			Name:   name,
			Status: DetectState(name),
		})
	}
	return sessions, nil
}

// New creates a detached tmux session and runs any configured commands inside it.
func New(name string, cfg config.Config) error {
	if err := exec.Command("tmux", "new-session", "-d", "-s", name).Run(); err != nil {
		return fmt.Errorf("create session %q: %w", name, err)
	}

	if cfg.NewSession.Layout != "" {
		_ = exec.Command("tmux", "select-layout", "-t", name, cfg.NewSession.Layout).Run()
	}

	for _, cmd := range cfg.NewSession.Commands {
		send := exec.Command("tmux", "send-keys", "-t", name, cmd, "Enter")
		if err := send.Run(); err != nil {
			return fmt.Errorf("send command %q to session %q: %w", cmd, name, err)
		}
	}

	return nil
}

// CapturePane returns the last numLines lines of a tmux pane's output.
func CapturePane(name string, numLines int) string {
	out, err := exec.Command("tmux", "capture-pane", "-pt", name, "-S", fmt.Sprintf("-%d", numLines)).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// Kill destroys a tmux session.
func Kill(name string) error {
	if err := exec.Command("tmux", "kill-session", "-t", name).Run(); err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}
	return nil
}

// NewWindow creates a new tmux window in the given session with a 3-pane layout:
// one tall pane on the left, two stacked panes on the right. All panes start in workDir.
func NewWindow(sessionName, windowName, workDir string) error {
	target := func(pane string) string {
		return fmt.Sprintf("%s:%s%s", sessionName, windowName, pane)
	}

	if err := exec.Command("tmux", "new-window", "-t", sessionName, "-n", windowName, "-c", workDir).Run(); err != nil {
		return fmt.Errorf("create window %q in %q: %w", windowName, sessionName, err)
	}
	if err := exec.Command("tmux", "split-window", "-t", target(""), "-h", "-c", workDir).Run(); err != nil {
		return fmt.Errorf("split horizontal: %w", err)
	}
	if err := exec.Command("tmux", "split-window", "-t", target(".1"), "-v", "-c", workDir).Run(); err != nil {
		return fmt.Errorf("split vertical: %w", err)
	}
	_ = exec.Command("tmux", "select-pane", "-t", target(".0")).Run()
	return nil
}

// Attach runs tmux attach-session and waits for it to exit (e.g. on detach).
// Control returns to the caller, so cctv can restart its TUI afterward.
func Attach(name string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
