package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	listPanelWidth = 46
	boxInnerWidth  = 52
	maxSuggestions = 8
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
	modeWizard
)

type wizardStep int

const (
	wizardName wizardStep = iota
	wizardRepos
	wizardNotes
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
	attachTarget string

	// Wizard state
	wizStep      wizardStep
	wizSession   string
	wizProject   string
	wizRepos     []string
	wizNotes     []string
	wizAllRepos  []string
	wizAllNotes  []string
	wizSugCursor int
}

func New(cfg config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "session name"
	ti.CharLimit = 64

	return Model{
		cfg:   cfg,
		input: ti,
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
		case modeWizard:
			return m.updateWizard(msg)
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
		if len(m.sessions) > 0 {
			m.wizSession = m.sessions[m.cursor].Name
			m.wizStep = wizardName
			m.wizProject = ""
			m.wizRepos = nil
			m.wizNotes = nil
			m.wizAllRepos = nil
			m.wizAllNotes = nil
			m.wizSugCursor = 0
			m.mode = modeWizard
			m.input.Reset()
			m.input.Placeholder = "project name"
			m.input.Focus()
			return m, textinput.Blink
		}

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

// ── Wizard ───────────────────────────────────────────────────────────────────

func (m Model) updateWizard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.wizStep {
	case wizardName:
		return m.updateWizardName(msg)
	case wizardRepos, wizardNotes:
		return m.updateWizardPicker(msg)
	}
	return m, nil
}

func (m Model) updateWizardName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			m.mode = modeList
			return m, nil
		}
		m.wizProject = name
		m.wizStep = wizardRepos
		m.wizAllRepos = scanDirs(m.cfg.ReposPath)
		m.wizSugCursor = 0
		m.input.Reset()
		m.input.Placeholder = "repo name (enter to add, empty to continue)"
		return m, nil
	case "esc":
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateWizardPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var all, selected []string
	if m.wizStep == wizardRepos {
		all = m.wizAllRepos
		selected = m.wizRepos
	} else {
		all = m.wizAllNotes
		selected = m.wizNotes
	}

	suggestions := filterSuggestions(all, m.input.Value(), selected)

	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val == "" {
			if m.wizStep == wizardRepos {
				m.wizStep = wizardNotes
				m.wizAllNotes = scanNotes(m.cfg.NotesPath)
				m.wizSugCursor = 0
				m.input.Reset()
				m.input.Placeholder = "note path (enter to add, empty to finish)"
			} else {
				if err := m.executeWizard(); err != nil {
					m.err = err.Error()
				}
				m.mode = modeList
				return m, fetchSessions
			}
			return m, nil
		}
		if m.wizStep == wizardRepos {
			m.wizRepos = append(m.wizRepos, val)
		} else {
			m.wizNotes = append(m.wizNotes, val)
		}
		m.wizSugCursor = 0
		m.input.Reset()
		return m, nil

	case "tab":
		if len(suggestions) > 0 {
			idx := m.wizSugCursor
			if idx >= len(suggestions) {
				idx = 0
			}
			m.input.SetValue(suggestions[idx])
			m.input.CursorEnd()
		}
		return m, nil

	case "up", "ctrl+p":
		if m.wizSugCursor > 0 {
			m.wizSugCursor--
		}
		return m, nil

	case "down", "ctrl+n":
		if m.wizSugCursor < len(suggestions)-1 {
			m.wizSugCursor++
		}
		return m, nil

	case "esc":
		m.mode = modeList
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.wizSugCursor = 0
	return m, cmd
}

func (m Model) executeWizard() error {
	projectDir := filepath.Join(m.cfg.WorkspacePath, m.wizProject)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	var md strings.Builder
	md.WriteString("# " + m.wizProject + "\n\n")

	md.WriteString("## Repos\n")
	md.WriteString("Source: " + m.cfg.ReposPath + "\n\n")
	for _, r := range m.wizRepos {
		md.WriteString("- " + r + "\n")
	}

	md.WriteString("\n## Notes\n")
	md.WriteString("Source: " + m.cfg.NotesPath + "\n\n")
	for _, n := range m.wizNotes {
		md.WriteString("- " + n + "\n")
	}

	mdPath := filepath.Join(projectDir, "project.md")
	if err := os.WriteFile(mdPath, []byte(md.String()), 0644); err != nil {
		return fmt.Errorf("write project.md: %w", err)
	}

	return session.NewWindow(m.wizSession, m.wizProject, projectDir)
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	return m.fullView()
}

func (m Model) fullView() string {
	var inner string
	if m.mode == modeWizard {
		inner = m.renderWizard(boxInnerWidth)
	} else {
		inner = m.renderList(boxInnerWidth)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("235")).
		Padding(1, 2).
		Width(boxInnerWidth).
		Render(inner)

	centered := box
	if m.width > 0 {
		centered = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, box)
	}

	var helpText string
	if m.mode == modeWizard {
		helpText = "[tab] complete  [up/down] suggestions  [enter] add/next  [esc] cancel"
	} else {
		helpText = "[n] new  [x] kill  [enter] attach  [j/k] nav  [tab] project  [q] quit"
	}
	help := helpStyle.Render(helpText)

	if m.height > 0 {
		boxLines := strings.Count(box, "\n")
		available := m.height - 1

		if topPad := max(0, (available-boxLines)/2); topPad > 0 {
			centered = strings.Repeat("\n", topPad) + centered
		}

		if pad := m.height - 1 - strings.Count(centered, "\n"); pad > 0 {
			centered += strings.Repeat("\n", pad)
		}
	}

	return centered + help
}

func (m Model) renderList(width int) string {
	var b strings.Builder

	title := titleStyle.Render("cctv")
	if width > 0 {
		title = lipgloss.PlaceHorizontal(width, lipgloss.Center, title)
	}
	b.WriteString(title + "\n\n")

	b.WriteString(colHeaderStyle.Render("  "+fmt.Sprintf("%-25s", "NAME")+"STATUS") + "\n")
	sepWidth := min(48, max(0, width-4))
	b.WriteString(dividerStyle.Render("  "+strings.Repeat("─", sepWidth)) + "\n")

	if len(m.sessions) == 0 {
		b.WriteString(stateIdle.Render("  no sessions") + "\n")
	}

	for i, s := range m.sessions {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("▸ ")
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

func (m Model) renderWizard(width int) string {
	var b strings.Builder

	title := titleStyle.Render("cctv")
	if width > 0 {
		title = lipgloss.PlaceHorizontal(width, lipgloss.Center, title)
	}
	b.WriteString(title + "\n\n")

	stepLabels := []string{"Name", "Repos", "Notes"}
	step := int(m.wizStep) + 1
	header := wizardHeaderStyle.Render(
		fmt.Sprintf("New project in %q  (%d/3 %s)", m.wizSession, step, stepLabels[m.wizStep]),
	)
	b.WriteString(header + "\n\n")

	switch m.wizStep {
	case wizardName:
		b.WriteString(inputPromptStyle.Render("Project name: ") + m.input.View() + "\n")

	case wizardRepos:
		if len(m.wizRepos) > 0 {
			b.WriteString(wizardLabelStyle.Render("Selected repos:") + "\n")
			for _, r := range m.wizRepos {
				b.WriteString(wizardSelectedStyle.Render("  + "+r) + "\n")
			}
			b.WriteString("\n")
		}
		b.WriteString(inputPromptStyle.Render("Add repo: ") + m.input.View() + "\n")
		suggestions := filterSuggestions(m.wizAllRepos, m.input.Value(), m.wizRepos)
		m.renderSuggestions(&b, suggestions)

	case wizardNotes:
		if len(m.wizNotes) > 0 {
			b.WriteString(wizardLabelStyle.Render("Selected notes:") + "\n")
			for _, n := range m.wizNotes {
				b.WriteString(wizardSelectedStyle.Render("  + "+n) + "\n")
			}
			b.WriteString("\n")
		}
		b.WriteString(inputPromptStyle.Render("Add note: ") + m.input.View() + "\n")
		suggestions := filterSuggestions(m.wizAllNotes, m.input.Value(), m.wizNotes)
		m.renderSuggestions(&b, suggestions)
	}

	if m.err != "" {
		b.WriteString("\n" + errorStyle.Render("error: "+m.err) + "\n")
	}

	return b.String()
}

func (m Model) renderSuggestions(b *strings.Builder, suggestions []string) {
	if len(suggestions) == 0 {
		return
	}
	b.WriteString("\n")
	for i, s := range suggestions {
		if i == m.wizSugCursor {
			b.WriteString(wizardSugHighlight.Render("  > "+s) + "\n")
		} else {
			b.WriteString(wizardSugStyle.Render("    "+s) + "\n")
		}
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func scanDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs
}

func scanNotes(root string) []string {
	var notes []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			rel, _ := filepath.Rel(root, path)
			notes = append(notes, rel)
		}
		return nil
	})
	sort.Strings(notes)
	return notes
}

func filterSuggestions(all []string, query string, selected []string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	skip := make(map[string]bool, len(selected))
	for _, s := range selected {
		skip[s] = true
	}
	var out []string
	for _, item := range all {
		if skip[item] {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(item), query) {
			out = append(out, item)
		}
	}
	if len(out) > maxSuggestions {
		out = out[:maxSuggestions]
	}
	return out
}

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

		_ = session.Attach(target)
	}
}
