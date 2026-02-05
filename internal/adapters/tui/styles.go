package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F97316") // Orange
	infoColor      = lipgloss.Color("#3B82F6") // Blue
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	fgColor        = lipgloss.Color("#F9FAFB") // Light gray
)

// Styles
var (
	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fgColor).
			Background(primaryColor).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(mutedColor)

	// Content styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1, 2)

	highlightBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	// Status styles
	statusOKStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusWarningStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	statusInfoStyle = lipgloss.NewStyle().
			Foreground(infoColor)

	// Graph styles
	graphStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)

	// Log level styles
	logDebugStyle = lipgloss.NewStyle().Foreground(mutedColor)
	logInfoStyle  = lipgloss.NewStyle().Foreground(infoColor)
	logWarnStyle  = lipgloss.NewStyle().Foreground(warningColor)
	logErrorStyle = lipgloss.NewStyle().Foreground(errorColor)

	// Metric styles
	metricValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor)

	metricLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// AI chat styles
	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(fgColor).
				Background(primaryColor).
				Padding(0, 1).
				MarginBottom(1)
)

// Helper functions for styling
func renderStatus(status string) string {
	switch status {
	case "ok", "active", "running", "completed", "installed":
		return statusOKStyle.Render("● " + status)
	case "error", "failed", "dead":
		return statusErrorStyle.Render("● " + status)
	case "warning", "pending", "disabled":
		return statusWarningStyle.Render("● " + status)
	case "available":
		return statusInfoStyle.Render("○ " + status)
	default:
		return statusInfoStyle.Render("● " + status)
	}
}



