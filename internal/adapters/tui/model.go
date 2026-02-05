// Package tui implements the Bubble Tea terminal user interface.
package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a tab in the TUI.
type Tab int

const (
	TabDashboard Tab = iota
	TabTasks
	TabWorkflows
	TabMetrics
	TabPlugins
	TabLogs
	TabAI
)

func (t Tab) String() string {
	return []string{"Dashboard", "Tasks", "Workflows", "Metrics", "Plugins", "Logs", "AI"}[t]
}

// Model represents the main TUI state.
type Model struct {
	activeTab       Tab
	tabs            []Tab
	width           int
	height          int
	help            help.Model
	keys            keyMap
	dashboard       *DashboardModel
	taskManager     *TaskManagerModel
	workflowManager *WorkflowManagerModel
	logViewer       *LogViewerModel
	pluginManager   *PluginManagerModel
	initialized     bool
}

// keyMap defines the key bindings.
type keyMap struct {
	Tab      key.Binding
	ShiftTab key.Binding
	Quit     key.Binding
	Help     key.Binding
	Enter    key.Binding
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.ShiftTab},
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Quit, k.Help},
	}
}

var defaultKeyMap = keyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
}

// NewModel creates a new TUI model.
func NewModel() Model {
	return Model{
		activeTab:       TabDashboard,
		tabs:            []Tab{TabDashboard, TabTasks, TabWorkflows, TabMetrics, TabPlugins, TabLogs, TabAI},
		help:            help.New(),
		keys:            defaultKeyMap,
		dashboard:       NewDashboardModel(),
		taskManager:     NewTaskManagerModel(),
		workflowManager: NewWorkflowManager(),
		logViewer:       NewLogViewerModel(),
		pluginManager:   NewPluginManagerModel(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.dashboard.Init(),
		m.taskManager.Init(),
		m.workflowManager.Init(),
		m.logViewer.Init(),
		m.pluginManager.Init(),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.initialized = true

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Tab):
			m.activeTab = Tab((int(m.activeTab) + 1) % len(m.tabs))
		case key.Matches(msg, m.keys.ShiftTab):
			m.activeTab = Tab((int(m.activeTab) - 1 + len(m.tabs)) % len(m.tabs))
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}
	}

	// Update active tab's model
	var cmd tea.Cmd
	switch m.activeTab {
	case TabDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case TabTasks:
		m.taskManager, cmd = m.taskManager.Update(msg)
	case TabWorkflows:
		m.workflowManager, cmd = m.workflowManager.Update(msg)
	case TabLogs:
		m.logViewer, cmd = m.logViewer.Update(msg)
	case TabPlugins:
		m.pluginManager, cmd = m.pluginManager.Update(msg)
	}

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.initialized {
		return "Loading..."
	}

	// Render tabs
	tabBar := m.renderTabs()

	// Render content based on active tab
	var content string
	contentHeight := m.height - 4
	switch m.activeTab {
	case TabDashboard:
		content = m.dashboard.View(m.width, contentHeight)
	case TabTasks:
		content = m.taskManager.View(m.width, contentHeight)
	case TabWorkflows:
		m.workflowManager.SetSize(m.width, contentHeight)
		content = m.workflowManager.View()
	case TabMetrics:
		content = m.renderMetricsTab()
	case TabPlugins:
		content = m.pluginManager.View(m.width, contentHeight)
	case TabLogs:
		content = m.logViewer.View(m.width, contentHeight)
	case TabAI:
		content = m.renderAITab()
	}

	// Render help
	helpView := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, helpView)
}

