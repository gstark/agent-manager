package tui

import "github.com/charmbracelet/lipgloss"

var (
	primary   = lipgloss.Color("#7D56F4")
	secondary = lipgloss.Color("#6C6C6C")
	accent    = lipgloss.Color("#04B575")
	subtle    = lipgloss.Color("#383838")
	text      = lipgloss.Color("#FAFAFA")
	dimText   = lipgloss.Color("#888888")

	activeTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			Padding(0, 2)

	inactiveTab = lipgloss.NewStyle().
			Foreground(dimText).
			Padding(0, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(text).
			Background(primary).
			Padding(0, 2).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimText).
			MarginTop(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(accent)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2)
)
