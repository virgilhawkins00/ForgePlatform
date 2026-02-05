package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/forge-platform/forge/internal/core/domain"
)

// AlertViewerKeyMap defines the key bindings for the alert viewer.
type AlertViewerKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Ack      key.Binding
	Silence  key.Binding
	Refresh  key.Binding
	ViewRule key.Binding
	Quit     key.Binding
}

// ShortHelp returns keybindings shown in the mini help.
func (k AlertViewerKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Ack, k.Silence, k.Refresh}
}

// FullHelp returns keybindings for the expanded help.
func (k AlertViewerKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Ack, k.Silence},
		{k.ViewRule, k.Refresh, k.Quit},
	}
}

var alertViewerKeys = AlertViewerKeyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Ack:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "acknowledge")),
	Silence:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "silence")),
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	ViewRule: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view rule")),
	Quit:     key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
}

// AlertViewer is a TUI component for viewing and managing alerts.
type AlertViewer struct {
	table        table.Model
	alerts       []*domain.Alert
	rules        []*domain.AlertRule
	stats        map[string]interface{}
	selectedTab  int // 0: Active, 1: History, 2: Rules
	keys         AlertViewerKeyMap
	help         help.Model
	width        int
	height       int
	err          error
}

// NewAlertViewer creates a new alert viewer.
func NewAlertViewer() *AlertViewer {
	columns := []table.Column{
		{Title: "State", Width: 12},
		{Title: "Severity", Width: 10},
		{Title: "Rule", Width: 25},
		{Title: "Value", Width: 12},
		{Title: "Started", Width: 18},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true)
	t.SetStyles(s)

	return &AlertViewer{
		table:       t,
		alerts:      make([]*domain.Alert, 0),
		rules:       make([]*domain.AlertRule, 0),
		stats:       make(map[string]interface{}),
		selectedTab: 0,
		keys:        alertViewerKeys,
		help:        help.New(),
	}
}

// Init initializes the alert viewer.
func (m *AlertViewer) Init() tea.Cmd {
	return m.refreshAlerts
}

// Update handles messages.
func (m *AlertViewer) Update(msg tea.Msg) (*AlertViewer, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 12)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Refresh):
			return m, m.refreshAlerts

		case key.Matches(msg, m.keys.Ack):
			if len(m.alerts) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.alerts) {
					return m, m.acknowledgeAlert(m.alerts[idx])
				}
			}

		case key.Matches(msg, m.keys.Silence):
			if len(m.alerts) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.alerts) {
					return m, m.silenceAlert(m.alerts[idx])
				}
			}
		}

	case alertsRefreshedMsg:
		m.alerts = msg.alerts
		m.stats = msg.stats
		m.updateTableRows()

	case alertAckedMsg:
		return m, m.refreshAlerts

	case errMsg:
		m.err = msg.err
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the alert viewer.
func (m *AlertViewer) View() string {
	var b strings.Builder

	// Header with stats
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Help
	b.WriteString(m.help.View(m.keys))

	return b.String()
}

func (m *AlertViewer) renderHeader() string {
	firing := 0
	pending := 0
	resolved := 0

	for _, a := range m.alerts {
		switch a.State {
		case domain.AlertStateFiring:
			firing++
		case domain.AlertStatePending:
			pending++
		case domain.AlertStateResolved:
			resolved++
		}
	}

	firingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	resolvedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	return fmt.Sprintf("Alerts: %s %s %s",
		firingStyle.Render(fmt.Sprintf("🔥 %d firing", firing)),
		pendingStyle.Render(fmt.Sprintf("⏳ %d pending", pending)),
		resolvedStyle.Render(fmt.Sprintf("✅ %d resolved", resolved)),
	)
}

func (m *AlertViewer) updateTableRows() {
	rows := make([]table.Row, len(m.alerts))
	for i, a := range m.alerts {
		rows[i] = table.Row{
			m.formatState(a.State),
			m.formatSeverity(a.Severity),
			a.RuleName,
			fmt.Sprintf("%.2f", a.Value),
			a.StartsAt.Format("2006-01-02 15:04"),
		}
	}
	m.table.SetRows(rows)
}

func (m *AlertViewer) formatState(state domain.AlertState) string {
	switch state {
	case domain.AlertStateFiring:
		return "🔥 firing"
	case domain.AlertStatePending:
		return "⏳ pending"
	case domain.AlertStateResolved:
		return "✅ resolved"
	case domain.AlertStateSilenced:
		return "🔇 silenced"
	case domain.AlertStateAcknowledged:
		return "👁 acked"
	default:
		return string(state)
	}
}

func (m *AlertViewer) formatSeverity(severity domain.AlertSeverity) string {
	switch severity {
	case domain.AlertSeverityCritical:
		return "🔴 critical"
	case domain.AlertSeverityWarning:
		return "🟡 warning"
	case domain.AlertSeverityInfo:
		return "🔵 info"
	default:
		return string(severity)
	}
}

// Message types
type alertsRefreshedMsg struct {
	alerts []*domain.Alert
	stats  map[string]interface{}
}

type alertAckedMsg struct {
	alertID string
}

type errMsg struct {
	err error
}

func (m *AlertViewer) refreshAlerts() tea.Msg {
	// In a real implementation, this would call the daemon
	// For now, return empty data
	return alertsRefreshedMsg{
		alerts: m.alerts,
		stats:  m.stats,
	}
}

func (m *AlertViewer) acknowledgeAlert(alert *domain.Alert) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, call daemon to acknowledge
		alert.Acknowledge("tui-user", "Acknowledged via TUI")
		return alertAckedMsg{alertID: alert.ID.String()}
	}
}

func (m *AlertViewer) silenceAlert(alert *domain.Alert) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, call daemon to silence
		alert.Silence()
		return alertAckedMsg{alertID: alert.ID.String()}
	}
}

// SetAlerts sets the alerts to display.
func (m *AlertViewer) SetAlerts(alerts []*domain.Alert) {
	m.alerts = alerts
	m.updateTableRows()
}

// SetStats sets the alert statistics.
func (m *AlertViewer) SetStats(stats map[string]interface{}) {
	m.stats = stats
}

// SetSize sets the component dimensions.
func (m *AlertViewer) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetWidth(width - 4)
	m.table.SetHeight(height - 12)
}

// Focus focuses the table.
func (m *AlertViewer) Focus() {
	m.table.Focus()
}

// Blur unfocuses the table.
func (m *AlertViewer) Blur() {
	m.table.Blur()
}

// SelectedAlert returns the currently selected alert.
func (m *AlertViewer) SelectedAlert() *domain.Alert {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.alerts) {
		return m.alerts[idx]
	}
	return nil
}

// RefreshFromDaemon refreshes alerts from the daemon.
func (m *AlertViewer) RefreshFromDaemon(getAlerts func() ([]*domain.Alert, error)) tea.Cmd {
	return func() tea.Msg {
		alerts, err := getAlerts()
		if err != nil {
			return errMsg{err: err}
		}
		return alertsRefreshedMsg{
			alerts: alerts,
			stats: map[string]interface{}{
				"total": len(alerts),
				"time":  time.Now().Format(time.RFC3339),
			},
		}
	}
}

