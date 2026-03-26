package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1)

	languageBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("250")).
				Background(lipgloss.Color("238")).
				Padding(0, 1)

	signatureStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

	installedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	missingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("240"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))

	npmSourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33"))

	pypiSourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	gitHubSourceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	homebrewSourceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("203"))

	noSourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	actionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	primaryKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("66")).
			Padding(0, 1)

	secondaryKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("238")).
				Padding(0, 1)

	keyHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	outdatedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	upToDateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)
