package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/forge-platform/forge/internal/adapters/daemon"
)

// GraphConfig defines configuration for a metric graph.
type GraphConfig struct {
	Name     string
	Title    string
	MaxValue float64
	Color    lipgloss.Color
	Icon     string
}

// MetricGraph represents a single graph panel.
type MetricGraph struct {
	config  GraphConfig
	history []float64
	current float64
}

// DashboardLayout defines available dashboard layouts.
type DashboardLayout int

const (
	LayoutGrid DashboardLayout = iota
	LayoutStacked
	LayoutFocused
)

func (l DashboardLayout) String() string {
	return []string{"Grid", "Stacked", "Focused"}[l]
}

// DashboardModel represents the dashboard tab state.
type DashboardModel struct {
	// Graphs
	graphs       []*MetricGraph
	focusedGraph int

	// Stats
	daemonStatus  string
	tasksRunning  int
	tasksQueued   int
	pluginsLoaded int
	metricsCount  int64
	seriesCount   int64
	uptime        string

	// UI state
	layout     DashboardLayout
	lastUpdate time.Time
	connected  bool
	client     *daemon.Client
	forgeDir   string

	// Key bindings
	keys dashboardKeyMap
}

// dashboardKeyMap defines dashboard-specific key bindings.
type dashboardKeyMap struct {
	CycleLayout  key.Binding
	NextGraph    key.Binding
	PrevGraph    key.Binding
	AddGraph     key.Binding
	RemoveGraph  key.Binding
	Refresh      key.Binding
}

func defaultDashboardKeyMap() dashboardKeyMap {
	return dashboardKeyMap{
		CycleLayout: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "cycle layout"),
		),
		NextGraph: key.NewBinding(
			key.WithKeys("n", "tab"),
			key.WithHelp("n", "next graph"),
		),
		PrevGraph: key.NewBinding(
			key.WithKeys("p", "shift+tab"),
			key.WithHelp("p", "prev graph"),
		),
		AddGraph: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add graph"),
		),
		RemoveGraph: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "remove graph"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

// NewDashboardModel creates a new dashboard model.
func NewDashboardModel() *DashboardModel {
	homeDir, _ := os.UserHomeDir()
	forgeDir := filepath.Join(homeDir, ".forge")

	// Default graphs
	graphs := []*MetricGraph{
		{
			config: GraphConfig{
				Name:     "cpu.usage",
				Title:    "CPU Usage",
				MaxValue: 100,
				Color:    lipgloss.Color("#7C3AED"),
				Icon:     "🔲",
			},
			history: make([]float64, 60),
		},
		{
			config: GraphConfig{
				Name:     "memory.usage",
				Title:    "Memory Usage",
				MaxValue: 100,
				Color:    lipgloss.Color("#10B981"),
				Icon:     "💾",
			},
			history: make([]float64, 60),
		},
		{
			config: GraphConfig{
				Name:     "disk.usage",
				Title:    "Disk I/O",
				MaxValue: 100,
				Color:    lipgloss.Color("#F59E0B"),
				Icon:     "💽",
			},
			history: make([]float64, 60),
		},
		{
			config: GraphConfig{
				Name:     "network.throughput",
				Title:    "Network",
				MaxValue: 100,
				Color:    lipgloss.Color("#3B82F6"),
				Icon:     "🌐",
			},
			history: make([]float64, 60),
		},
	}

	return &DashboardModel{
		graphs:       graphs,
		focusedGraph: 0,
		lastUpdate:   time.Now(),
		daemonStatus: "disconnected",
		layout:       LayoutGrid,
		forgeDir:     forgeDir,
		keys:         defaultDashboardKeyMap(),
	}
}

// tickMsg is sent periodically to update the dashboard.
type tickMsg time.Time

// daemonStatusMsg contains status from daemon.
type daemonStatusMsg struct {
	connected     bool
	uptime        string
	tasksRunning  int
	tasksQueued   int
	pluginsLoaded int
	metricsCount  int64
	seriesCount   int64
}

// metricsDataMsg contains metric values from daemon.
type metricsDataMsg struct {
	data map[string]float64 // metric name -> latest value
}

// Init initializes the dashboard.
func (m *DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.connectToDaemon(),
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

// connectToDaemon attempts to connect to the daemon.
func (m *DashboardModel) connectToDaemon() tea.Cmd {
	return func() tea.Msg {
		client, err := daemon.NewClient(m.forgeDir)
		if err != nil {
			return daemonStatusMsg{connected: false}
		}

		if err := client.Connect(); err != nil {
			return daemonStatusMsg{connected: false}
		}

		// Get status
		ctx := context.Background()
		status, err := client.Status(ctx)
		if err != nil {
			client.Close()
			return daemonStatusMsg{connected: false}
		}

		msg := daemonStatusMsg{connected: true}
		if uptime, ok := status["uptime"].(string); ok {
			msg.uptime = uptime
		}

		// Store client for later use
		m.client = client
		return msg
	}
}

// fetchMetrics fetches current metrics from daemon.
func (m *DashboardModel) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return daemonStatusMsg{connected: false}
		}

		// Get stats
		ctx := context.Background()
		stats, err := m.client.GetMetricStats(ctx)
		if err != nil {
			return daemonStatusMsg{connected: false}
		}

		msg := daemonStatusMsg{connected: true, uptime: m.uptime}
		if count, ok := stats["total_points"].(float64); ok {
			msg.metricsCount = int64(count)
		}
		if series, ok := stats["total_series"].(float64); ok {
			msg.seriesCount = int64(series)
		}

		return msg
	}
}

// fetchMetricValues fetches actual metric values from daemon.
func (m *DashboardModel) fetchMetricValues() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}

		data := make(map[string]float64)
		ctx := context.Background()

		// Fetch latest value for each configured graph
		for _, g := range m.graphs {
			metrics, err := m.client.QueryMetric(ctx, g.config.Name, 1)
			if err != nil || len(metrics) == 0 {
				continue
			}
			if val, ok := metrics[0]["value"].(float64); ok {
				data[g.config.Name] = val
			}
		}

		return metricsDataMsg{data: data}
	}
}

// Update handles dashboard updates.
func (m *DashboardModel) Update(msg tea.Msg) (*DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.lastUpdate = time.Time(msg)

		// Simulate data when not connected (for demo)
		if !m.connected {
			for _, g := range m.graphs {
				val := 30.0 + float64(time.Now().Second()%40)
				if g.config.Name == "memory.usage" {
					val = 50.0 + float64(time.Now().Second()%25)
				} else if g.config.Name == "disk.usage" {
					val = 20.0 + float64(time.Now().Second()%15)
				} else if g.config.Name == "network.throughput" {
					val = 10.0 + float64(time.Now().Second()%60)
				}
				g.history = append(g.history[1:], val)
				g.current = val
			}
		}

		var cmds []tea.Cmd
		cmds = append(cmds, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}))

		// Fetch stats periodically if connected
		if m.connected && time.Now().Second()%5 == 0 {
			cmds = append(cmds, m.fetchMetrics())
		}

		// Fetch metric values every second when connected
		if m.connected {
			cmds = append(cmds, m.fetchMetricValues())
		}

		return m, tea.Batch(cmds...)

	case daemonStatusMsg:
		m.connected = msg.connected
		if msg.connected {
			m.daemonStatus = "connected"
			m.uptime = msg.uptime
			m.metricsCount = msg.metricsCount
			m.seriesCount = msg.seriesCount
			m.tasksRunning = msg.tasksRunning
			m.tasksQueued = msg.tasksQueued
			m.pluginsLoaded = msg.pluginsLoaded
		} else {
			m.daemonStatus = "disconnected"
		}

	case metricsDataMsg:
		// Update graph data with real values from daemon
		for _, g := range m.graphs {
			if val, ok := msg.data[g.config.Name]; ok {
				g.history = append(g.history[1:], val)
				g.current = val
			}
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.CycleLayout):
			m.layout = DashboardLayout((int(m.layout) + 1) % 3)
		case key.Matches(msg, m.keys.NextGraph):
			m.focusedGraph = (m.focusedGraph + 1) % len(m.graphs)
		case key.Matches(msg, m.keys.PrevGraph):
			m.focusedGraph = (m.focusedGraph - 1 + len(m.graphs)) % len(m.graphs)
		case key.Matches(msg, m.keys.Refresh):
			return m, m.connectToDaemon()
		}
	}
	return m, nil
}

// View renders the dashboard.
func (m *DashboardModel) View(width, height int) string {
	if width < 40 || height < 10 {
		return "Terminal too small"
	}

	// Header
	header := titleStyle.Render("📊 Dashboard")
	statusLine := m.renderStatusLine()

	// Stats boxes
	statsBox := m.renderStatsBox(width - 4)

	// Graphs based on layout
	graphsView := m.renderGraphs(width, height-12)

	// Help line
	helpLine := subtitleStyle.Render(fmt.Sprintf("Layout: %s | [l] cycle layout | [n/p] navigate | [r] refresh", m.layout))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		subtitleStyle.Render(statusLine),
		"",
		statsBox,
		"",
		graphsView,
		"",
		helpLine,
	)
}

func (m *DashboardModel) renderStatusLine() string {
	status := renderStatus(m.daemonStatus)
	uptimeStr := ""
	if m.uptime != "" {
		uptimeStr = fmt.Sprintf(" | Uptime: %s", m.uptime)
	}
	return fmt.Sprintf("Last update: %s | Daemon: %s%s",
		m.lastUpdate.Format("15:04:05"),
		status,
		uptimeStr)
}

func (m *DashboardModel) renderStatsBox(width int) string {
	numStats := 6
	boxWidth := (width - 2) / numStats
	if boxWidth < 12 {
		boxWidth = 12
	}

	stats := []struct {
		label string
		value string
		icon  string
	}{
		{"Tasks Run", fmt.Sprintf("%d", m.tasksRunning), "⚡"},
		{"Tasks Queue", fmt.Sprintf("%d", m.tasksQueued), "📋"},
		{"Plugins", fmt.Sprintf("%d", m.pluginsLoaded), "🔌"},
		{"Metrics", formatNumber(m.metricsCount), "📈"},
		{"Series", fmt.Sprintf("%d", m.seriesCount), "📊"},
		{"Graphs", fmt.Sprintf("%d", len(m.graphs)), "📉"},
	}

	var boxes []string
	for _, s := range stats {
		box := boxStyle.Width(boxWidth).Render(
			lipgloss.JoinVertical(lipgloss.Center,
				s.icon,
				metricValueStyle.Render(s.value),
				metricLabelStyle.Render(s.label),
			),
		)
		boxes = append(boxes, box)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
}

func (m *DashboardModel) renderGraphs(width, height int) string {
	if len(m.graphs) == 0 {
		return boxStyle.Width(width - 4).Render("No graphs configured. Press 'a' to add.")
	}

	switch m.layout {
	case LayoutStacked:
		return m.renderStackedLayout(width, height)
	case LayoutFocused:
		return m.renderFocusedLayout(width, height)
	default:
		return m.renderGridLayout(width, height)
	}
}

func (m *DashboardModel) renderGridLayout(width, height int) string {
	numGraphs := len(m.graphs)
	if numGraphs == 0 {
		return ""
	}

	// Calculate grid dimensions
	cols := 2
	if numGraphs == 1 {
		cols = 1
	} else if numGraphs > 4 {
		cols = 3
	}
	rows := (numGraphs + cols - 1) / cols
	graphWidth := (width - 4) / cols
	graphHeight := (height - 2) / rows
	if graphHeight < 6 {
		graphHeight = 6
	}

	var rowViews []string
	for row := 0; row < rows; row++ {
		var colViews []string
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= numGraphs {
				break
			}
			g := m.graphs[idx]
			isFocused := idx == m.focusedGraph

			style := graphStyle.Width(graphWidth - 2)
			if isFocused {
				style = style.BorderForeground(g.config.Color)
			}

			graphView := m.renderSingleGraph(g, graphWidth-6, graphHeight-4)
			colViews = append(colViews, style.Render(graphView))
		}
		rowViews = append(rowViews, lipgloss.JoinHorizontal(lipgloss.Top, colViews...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rowViews...)
}

func (m *DashboardModel) renderStackedLayout(width, height int) string {
	numGraphs := len(m.graphs)
	graphHeight := (height - 2) / numGraphs
	if graphHeight < 5 {
		graphHeight = 5
	}

	var views []string
	for idx, g := range m.graphs {
		isFocused := idx == m.focusedGraph
		style := graphStyle.Width(width - 4)
		if isFocused {
			style = style.BorderForeground(g.config.Color)
		}

		graphView := m.renderSingleGraph(g, width-10, graphHeight-3)
		views = append(views, style.Render(graphView))
	}

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func (m *DashboardModel) renderFocusedLayout(width, height int) string {
	if m.focusedGraph >= len(m.graphs) {
		return ""
	}

	focused := m.graphs[m.focusedGraph]
	mainHeight := height - 6
	thumbWidth := (width - 4) / len(m.graphs)

	// Main graph
	mainStyle := graphStyle.
		Width(width - 4).
		BorderForeground(focused.config.Color)
	mainView := mainStyle.Render(m.renderSingleGraph(focused, width-10, mainHeight))

	// Thumbnails
	var thumbs []string
	for idx, g := range m.graphs {
		style := boxStyle.Width(thumbWidth - 2)
		if idx == m.focusedGraph {
			style = style.BorderForeground(g.config.Color)
		}
		thumb := style.Render(fmt.Sprintf("%s %s", g.config.Icon, g.config.Title[:minInt(8, len(g.config.Title))]))
		thumbs = append(thumbs, thumb)
	}
	thumbRow := lipgloss.JoinHorizontal(lipgloss.Top, thumbs...)

	return lipgloss.JoinVertical(lipgloss.Left, mainView, "", thumbRow)
}

func (m *DashboardModel) renderSingleGraph(g *MetricGraph, width, height int) string {
	if len(g.history) == 0 || width < 10 || height < 3 {
		return g.config.Title + "\n(no data)"
	}

	// Braille characters for graph rendering
	braille := []rune{'⠀', '⣀', '⣤', '⣶', '⣿'}

	// Normalize data to height
	maxVal := g.config.MaxValue
	if maxVal == 0 {
		maxVal = 100.0
	}
	lines := make([]string, height)

	pointsPerCol := len(g.history) / width
	if pointsPerCol < 1 {
		pointsPerCol = 1
	}

	for col := 0; col < width && col*pointsPerCol < len(g.history); col++ {
		val := g.history[col*pointsPerCol]
		normalized := int((val / maxVal) * float64(height*4))

		for row := 0; row < height; row++ {
			level := normalized - (height-1-row)*4
			if level < 0 {
				level = 0
			} else if level > 4 {
				level = 4
			}
			lines[row] += string(braille[level])
		}
	}

	// Header with current value
	header := fmt.Sprintf("%s %s: %.1f%%", g.config.Icon, g.config.Title, g.current)

	return lipgloss.JoinVertical(lipgloss.Left,
		metricLabelStyle.Render(header),
		strings.Join(lines, "\n"),
	)
}

// Helper functions

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

