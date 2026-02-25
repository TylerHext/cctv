package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tylerhext/cctv/internal/config"
	"github.com/tylerhext/cctv/internal/session"
)

const (
	pollInterval   = 2 * time.Second
	listPanelWidth = 46 // left panel width in split view
	boxInnerWidth  = 52 // content width inside the full-view border box
	minSplitWidth  = 110
)

type tickMsg time.Time

type sessionResult struct {
	sessions []session.Session
	err      error
}

type mode int

const (
	modeList mode = iota
	modeNewInput
	modeKillConfirm
)

type Model struct {
	sessions     []session.Session
	cursor       int
	mode         mode
	input        textinput.Model
	cfg          config.Config
	err          string
	width        int
	height       int
	showPreview  bool
	attachTarget string
}

func New(cfg config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "session name"
	ti.CharLimit = 64

	return Model{
		cfg:         cfg,
		input:       ti,
		showPreview: false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchSessions, tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchSessions() tea.Msg {
	sessions, err := session.List()
	return sessionResult{sessions: sessions, err: err}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case sessionResult:
		if msg.err != nil {
			m.err = "tmux unavailable: " + msg.err.Error()
			m.sessions = nil
			m.cursor = 0
			return m, nil
		}
		m.err = ""
		m.sessions = msg.sessions
		if m.cursor >= len(m.sessions) && len(m.sessions) > 0 {
			m.cursor = len(m.sessions) - 1
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchSessions, tickCmd())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		m.err = ""
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeNewInput:
			return m.updateNewInput(msg)
		case modeKillConfirm:
			return m.updateKillConfirm(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.sessions)-1 {
			m.cursor++
		}

	case "tab":
		m.showPreview = !m.showPreview

	case "enter":
		if len(m.sessions) > 0 {
			m.attachTarget = m.sessions[m.cursor].Name
			return m, tea.Quit
		}

	case "n":
		m.mode = modeNewInput
		m.input.Reset()
		m.input.Focus()
		return m, textinput.Blink

	case "x":
		if len(m.sessions) > 0 {
			m.mode = modeKillConfirm
		}
	}
	return m, nil
}

func (m Model) updateNewInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			m.mode = modeList
			return m, nil
		}
		if err := session.New(name, m.cfg); err != nil {
			m.err = err.Error()
			m.mode = modeList
			return m, fetchSessions
		}
		m.mode = modeList
		return m, fetchSessions

	case "esc":
		m.mode = modeList
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateKillConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if len(m.sessions) > 0 {
			name := m.sessions[m.cursor].Name
			if err := session.Kill(name); err != nil {
				m.err = err.Error()
			}
		}
		m.mode = modeList
		return m, fetchSessions

	default:
		m.mode = modeList
	}
	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	inSplit := m.showPreview && m.width >= minSplitWidth && len(m.sessions) > 0 && m.height > 0
	if inSplit {
		return m.splitView()
	}
	return m.fullView()
}

// fullView renders a centered border box with the help bar pinned to the bottom.
func (m Model) fullView() string {
	inner := m.renderList(boxInnerWidth)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("235")).
		Padding(1, 2).
		Width(boxInnerWidth).
		Render(inner)

	// Center the box horizontally.
	centered := box
	if m.width > 0 {
		centered = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, box)
	}

	help := helpStyle.Render("[n] new  [x] kill  [enter] attach  [j/k] nav  [tab] preview  [q] quit")

	if m.height > 0 {
		boxLines := strings.Count(box, "\n")
		available := m.height - 1 // one line reserved for help

		// Vertically center the box.
		if topPad := max(0, (available-boxLines)/2); topPad > 0 {
			centered = strings.Repeat("\n", topPad) + centered
		}

		// Fill remaining space so help lands on the last row.
		if pad := m.height - 1 - strings.Count(centered, "\n"); pad > 0 {
			centered += strings.Repeat("\n", pad)
		}
	}

	return centered + help
}

// splitView renders a left session list + right preview pane.
func (m Model) splitView() string {
	previewW := m.width - listPanelWidth - 1 // 1 for separator column
	contentH := m.height - 1                 // 1 line reserved for help bar

	// Left panel
	listContent := m.renderList(listPanelWidth)
	leftPanel := lipgloss.NewStyle().
		Width(listPanelWidth).
		Height(contentH).
		Render(listContent)

	// Separator
	sepLines := make([]string, contentH)
	for i := range sepLines {
		sepLines[i] = "│"
	}
	sep := sepStyle.Render(strings.Join(sepLines, "\n"))

	// Right panel (live preview)
	rightPanel := lipgloss.NewStyle().
		Width(previewW).
		Height(contentH).
		Render(m.renderPreview(previewW, contentH))

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, sep, rightPanel)

	help := helpStyle.Width(m.width).Render(
		"[n] new  [x] kill  [enter] attach  [j/k] nav  [tab] hide  [q] quit",
	)

	return body + "\n" + help
}

// renderList builds the session list content (shared by both views).
// width is used to size the column divider in the full (non-split) view.
func (m Model) renderList(width int) string {
	var b strings.Builder

	title := titleStyle.Render("cctv")
	if width > 0 {
		title = lipgloss.PlaceHorizontal(width, lipgloss.Center, title)
	}
	b.WriteString(title + "\n\n")

	// Layout (no dot): cursor(2) + name(22) + gap(3) + badge
	// NAME aligns with col 2 (cursor width), STATUS aligns with col 27 (2+22+3).
	b.WriteString(colHeaderStyle.Render("  "+fmt.Sprintf("%-25s", "NAME")+"STATUS") + "\n")
	sepWidth := min(48, max(0, width-4))
	b.WriteString(dividerStyle.Render("  "+strings.Repeat("─", sepWidth)) + "\n")

	if len(m.sessions) == 0 {
		b.WriteString(stateIdle.Render("  no sessions") + "\n")
	}

	for i, s := range m.sessions {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("▶ ")
		}

		nameStyle := normalNameStyle
		if i == m.cursor {
			nameStyle = selectedNameStyle
		}
		name := nameStyle.Render(fmt.Sprintf("%-22s", truncate(s.Name, 22)))

		var badge string
		switch s.Status {
		case session.StateWorking:
			badge = stateWorking.Render("◆ Working")
		case session.StateWaiting:
			badge = stateWaiting.Render("◉ Waiting")
		default:
			badge = stateIdle.Render("○ Idle")
		}

		b.WriteString(fmt.Sprintf("%s%s   %s\n", cursor, name, badge))
	}

	// Mode overlays sit inside the list panel in both views.
	switch m.mode {
	case modeNewInput:
		b.WriteString("\n" + inputPromptStyle.Render("New session name: ") + m.input.View() + "\n")
	case modeKillConfirm:
		if len(m.sessions) > 0 {
			b.WriteString("\n" + confirmStyle.Render(
				fmt.Sprintf("Kill %q? [y/N] ", m.sessions[m.cursor].Name),
			) + "\n")
		}
	}

	if m.err != "" {
		b.WriteString("\n" + errorStyle.Render("error: "+m.err) + "\n")
	}

	return b.String()
}

// renderPreview shows the live pane output for the selected session.
func (m Model) renderPreview(width, height int) string {
	if len(m.sessions) == 0 || m.cursor >= len(m.sessions) {
		return ""
	}

	s := m.sessions[m.cursor]

	var badge string
	switch s.Status {
	case session.StateWorking:
		badge = stateWorking.Render("◆ Working")
	case session.StateWaiting:
		badge = stateWaiting.Render("◉ Waiting")
	default:
		badge = stateIdle.Render("○ Idle")
	}

	header := previewHeaderStyle.Render(s.Name) + "  " + badge + "\n\n"
	bodyLines := height - 3 // rows available after header (2 lines) + trailing newline
	if bodyLines < 1 {
		return header
	}

	raw := session.CapturePane(s.Name, bodyLines)
	lines := strings.Split(raw, "\n")

	// Strip trailing blank lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// Keep only the last bodyLines lines.
	if len(lines) > bodyLines {
		lines = lines[len(lines)-bodyLines:]
	}

	var b strings.Builder
	b.WriteString(header)
	for _, line := range lines {
		// Truncate to panel width (rune-safe).
		runes := []rune(line)
		if len(runes) > width-1 {
			runes = runes[:width-1]
		}
		b.WriteString(previewStyle.Render(string(runes)) + "\n")
	}

	return b.String()
}

// truncate shortens s to at most n runes.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// AttachTarget returns the session name the user chose to attach to.
func (m Model) AttachTarget() string {
	return m.attachTarget
}

// Start runs the TUI in a loop; after detaching from tmux it restarts.
func Start(cfg config.Config) error {
	for {
		m := New(cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		final, err := p.Run()
		if err != nil {
			return err
		}

		fm, ok := final.(Model)
		if !ok {
			return nil
		}

		target := fm.AttachTarget()
		if target == "" {
			return nil
		}

		// Attach and wait; on detach (prefix+d) the loop restarts the TUI.
		_ = session.Attach(target)
	}
}
