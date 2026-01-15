package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color constants for the TUI.
var (
	ColorPrimary   = lipgloss.Color("5")  // Magenta
	ColorSecondary = lipgloss.Color("4")  // Blue
	ColorSuccess   = lipgloss.Color("2")  // Green
	ColorWarning   = lipgloss.Color("3")  // Yellow
	ColorError     = lipgloss.Color("1")  // Red
	ColorMuted     = lipgloss.Color("8")  // Gray
	ColorHighlight = lipgloss.Color("11") // Bright Yellow
	ColorWhite     = lipgloss.Color("15") // White
	ColorBlack     = lipgloss.Color("0")  // Black
)

// ServiceColors is a list of colors for service prefixes.
var ServiceColors = []lipgloss.Color{
	lipgloss.Color("6"),  // Cyan
	lipgloss.Color("2"),  // Green
	lipgloss.Color("3"),  // Yellow
	lipgloss.Color("4"),  // Blue
	lipgloss.Color("5"),  // Magenta
	lipgloss.Color("1"),  // Red
	lipgloss.Color("14"), // Bright Cyan
	lipgloss.Color("10"), // Bright Green
}

// Styles for the TUI components.
var (
	// Tab bar styles
	TabBarStyle = lipgloss.NewStyle()

	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight).
			Background(lipgloss.Color("236")).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 2)

	TabSeparatorStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Log view styles (no border, just content)
	LogViewStyle = lipgloss.NewStyle()

	LogPrefixStyle = lipgloss.NewStyle().
			Bold(true)

	LogMessageStyle = lipgloss.NewStyle()

	LogTimestampStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Search highlight
	HighlightStyle = lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorBlack)

	// Search bar styles
	SearchBarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	SearchInputStyle = lipgloss.NewStyle()

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	StatusDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// State indicator styles
	StateRunningStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	StateBuildingStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	StateStoppedStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StateErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Divider style
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Selected line style
	SelectedLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(ColorWhite)
)

// GetServiceColor returns a color for a service based on its index.
func GetServiceColor(index int) lipgloss.Color {
	return ServiceColors[index%len(ServiceColors)]
}

// StateStyle returns the style for a process state.
func StateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return StateRunningStyle
	case "building", "starting":
		return StateBuildingStyle
	case "error":
		return StateErrorStyle
	default:
		return StateStoppedStyle
	}
}

// StateIcon returns an icon for a process state.
func StateIcon(state string) string {
	switch state {
	case "running":
		return "●"
	case "building", "starting":
		return "◐"
	case "stopping":
		return "◑"
	case "error":
		return "✗"
	case "stopped":
		return "○"
	default:
		return "○"
	}
}
