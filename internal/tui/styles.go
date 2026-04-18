package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colours — adaptive so they work on light and dark terminals.
	subtle  = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	accent  = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#AD8CFF"}
	userClr = lipgloss.AdaptiveColor{Light: "#0077B6", Dark: "#48CAE4"}
	errClr  = lipgloss.AdaptiveColor{Light: "#D00000", Dark: "#FF6B6B"}

	// Header bar.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accent).
			Padding(0, 1)

	// User message prefix.
	userPrefixStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(userClr)

	// Assistant message prefix.
	assistantPrefixStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accent)

	// Streaming cursor.
	cursorStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	// Input area border.
	inputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(subtle).
				Padding(0, 1)

	// Status / info text.
	statusStyle = lipgloss.NewStyle().
			Foreground(subtle)

	// Error text.
	errorStyle = lipgloss.NewStyle().
			Foreground(errClr).
			Bold(true)

	// Help text.
	helpStyle = lipgloss.NewStyle().
			Foreground(subtle)
)
