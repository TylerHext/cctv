package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#14b8a6")) // teal

	colHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("237"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	normalNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)

	stateWorking = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fa8072")) // salmon

	stateWaiting = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7dd3fc")) // light blue

	stateIdle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6b7280")) // grey

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	sepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("237"))

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255"))

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	wizardHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#14b8a6"))

	wizardLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	wizardSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7dd3fc"))

	wizardSugStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	wizardSugHighlight = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)
)
