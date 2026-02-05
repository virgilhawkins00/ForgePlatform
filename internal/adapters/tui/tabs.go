package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTabs renders the tab bar.
func (m Model) renderTabs() string {
	var tabs []string

	for _, tab := range m.tabs {
		style := inactiveTabStyle
		if tab == m.activeTab {
			style = activeTabStyle
		}
		tabs = append(tabs, style.Render(tab.String()))
	}

	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return tabBarStyle.Width(m.width).Render(tabRow)
}

// Note: Task, Plugin, and Log tabs are now rendered by their respective models
// (TaskManagerModel, PluginManagerModel, LogViewerModel)

// renderMetricsTab renders the metrics tab content.
func (m Model) renderMetricsTab() string {
	header := titleStyle.Render("📈 Metrics Explorer")

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		subtitleStyle.Render("Query metrics from the time-series database"),
		"",
		"Series: (daemon not connected)",
		"",
		"Use arrow keys to navigate, Enter to select",
	)

	return boxStyle.Width(m.width - 4).Render(content)
}

// renderAITab renders the AI chat tab content.
func (m Model) renderAITab() string {
	header := titleStyle.Render("🤖 AI Assistant")

	// Chat history placeholder
	messages := []string{
		assistantMessageStyle.Render("Hello! I'm your Forge AI assistant. How can I help you today?"),
		"",
		subtitleStyle.Render("Type your message and press Enter to send"),
	}

	// Input area
	inputArea := boxStyle.
		BorderForeground(primaryColor).
		Width(m.width - 8).
		Render("Type here...")

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		subtitleStyle.Render("Model: llama3.2 | Temperature: 0.7"),
		"",
		strings.Join(messages, "\n"),
		"",
		inputArea,
	)

	return content
}

