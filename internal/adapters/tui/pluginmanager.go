package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PluginItem represents a plugin in the list.
type PluginItem struct {
	Name        string
	Version     string
	Author      string
	Desc        string // Plugin description
	Status      string // installed, disabled, available
	Permissions []string
	Size        string
	Downloads   int
}

func (p PluginItem) Title() string {
	icon := "📦"
	switch p.Status {
	case "installed":
		icon = "✅"
	case "disabled":
		icon = "⏸️"
	case "available":
		icon = "📥"
	}
	return fmt.Sprintf("%s %s v%s", icon, p.Name, p.Version)
}

func (p PluginItem) Description() string {
	desc := p.Desc
	if len(desc) > 40 {
		desc = desc[:40]
	}
	return fmt.Sprintf("%s | By: %s | %s", p.Status, p.Author, desc)
}

func (p PluginItem) FilterValue() string {
	return p.Name + " " + p.Desc + " " + p.Author
}

// PluginManagerModel represents the plugin manager state.
type PluginManagerModel struct {
	// Plugin lists
	installedList  list.Model
	availableList  list.Model
	installed      []PluginItem
	available      []PluginItem

	// UI state
	width        int
	height       int
	activeTab    int // 0: installed, 1: available
	showDetails  bool
	selected     *PluginItem
	confirmModal bool
	confirmAction string

	// Key bindings
	keys pluginManagerKeyMap
}

type pluginManagerKeyMap struct {
	Install   key.Binding
	Uninstall key.Binding
	Enable    key.Binding
	Disable   key.Binding
	Details   key.Binding
	Tab       key.Binding
	Configure key.Binding
	Refresh   key.Binding
	Back      key.Binding
	Confirm   key.Binding
}

func defaultPluginManagerKeyMap() pluginManagerKeyMap {
	return pluginManagerKeyMap{
		Install:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
		Uninstall: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "uninstall")),
		Enable:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "enable")),
		Disable:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "disable")),
		Details:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Tab:       key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch tab")),
		Configure: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "configure")),
		Refresh:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Confirm:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
	}
}

// NewPluginManagerModel creates a new plugin manager model.
func NewPluginManagerModel() *PluginManagerModel {
	// Sample installed plugins
	installed := []PluginItem{
		{
			Name: "system-metrics", Version: "1.2.0", Author: "forge-team",
			Desc: "Collect system CPU, memory, and disk metrics",
			Status: "installed", Permissions: []string{"metric:write", "system:read"},
			Size: "2.1 MB", Downloads: 15420,
		},
		{
			Name: "docker-stats", Version: "1.0.5", Author: "community",
			Desc: "Monitor Docker containers and collect stats",
			Status: "disabled", Permissions: []string{"metric:write", "docker:read"},
			Size: "1.8 MB", Downloads: 8932,
		},
	}

	// Sample available plugins
	available := []PluginItem{
		{
			Name: "kubernetes-monitor", Version: "2.0.0", Author: "forge-team",
			Desc: "Full Kubernetes cluster monitoring and alerting",
			Status: "available", Permissions: []string{"metric:write", "k8s:read", "alert:write"},
			Size: "4.5 MB", Downloads: 25600,
		},
		{
			Name: "postgres-exporter", Version: "1.1.0", Author: "community",
			Desc: "Export PostgreSQL database metrics",
			Status: "available", Permissions: []string{"metric:write", "network:connect"},
			Size: "1.2 MB", Downloads: 12300,
		},
		{
			Name: "slack-notifier", Version: "1.3.2", Author: "integrations",
			Desc: "Send alerts and notifications to Slack channels",
			Status: "available", Permissions: []string{"alert:read", "network:connect"},
			Size: "0.8 MB", Downloads: 18700,
		},
	}

	// Create list models
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(primaryColor).BorderForeground(primaryColor)

	installedItems := make([]list.Item, len(installed))
	for i, p := range installed {
		installedItems[i] = p
	}
	installedList := list.New(installedItems, delegate, 80, 15)
	installedList.Title = "🔌 Installed Plugins"
	installedList.SetFilteringEnabled(true)
	installedList.Styles.Title = titleStyle

	availableItems := make([]list.Item, len(available))
	for i, p := range available {
		availableItems[i] = p
	}
	availableList := list.New(availableItems, delegate, 80, 15)
	availableList.Title = "📦 Available Plugins"
	availableList.SetFilteringEnabled(true)
	availableList.Styles.Title = titleStyle

	return &PluginManagerModel{
		installedList: installedList,
		availableList: availableList,
		installed:     installed,
		available:     available,
		keys:          defaultPluginManagerKeyMap(),
	}
}

// Init initializes the plugin manager.
func (m *PluginManagerModel) Init() tea.Cmd {
	return nil
}

// Update handles plugin manager updates.
func (m *PluginManagerModel) Update(msg tea.Msg) (*PluginManagerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.installedList.SetWidth(msg.Width - 4)
		m.installedList.SetHeight(msg.Height - 10)
		m.availableList.SetWidth(msg.Width - 4)
		m.availableList.SetHeight(msg.Height - 10)

	case tea.KeyMsg:
		// Confirmation modal
		if m.confirmModal {
			switch {
			case key.Matches(msg, m.keys.Confirm):
				m.executeAction()
				m.confirmModal = false
				m.confirmAction = ""
			case key.Matches(msg, m.keys.Back):
				m.confirmModal = false
				m.confirmAction = ""
			}
			return m, nil
		}

		// Details view
		if m.showDetails {
			if key.Matches(msg, m.keys.Back) {
				m.showDetails = false
				m.selected = nil
			}
			return m, nil
		}

		// Normal mode
		switch {
		case key.Matches(msg, m.keys.Tab):
			m.activeTab = (m.activeTab + 1) % 2

		case key.Matches(msg, m.keys.Details):
			var item PluginItem
			var ok bool
			if m.activeTab == 0 {
				item, ok = m.installedList.SelectedItem().(PluginItem)
			} else {
				item, ok = m.availableList.SelectedItem().(PluginItem)
			}
			if ok {
				m.selected = &item
				m.showDetails = true
			}

		case key.Matches(msg, m.keys.Install):
			if m.activeTab == 1 {
				if item, ok := m.availableList.SelectedItem().(PluginItem); ok {
					m.selected = &item
					m.confirmAction = "install"
					m.confirmModal = true
				}
			}

		case key.Matches(msg, m.keys.Uninstall):
			if m.activeTab == 0 {
				if item, ok := m.installedList.SelectedItem().(PluginItem); ok {
					m.selected = &item
					m.confirmAction = "uninstall"
					m.confirmModal = true
				}
			}

		case key.Matches(msg, m.keys.Enable):
			if m.activeTab == 0 {
				m.togglePlugin(true)
			}

		case key.Matches(msg, m.keys.Disable):
			if m.activeTab == 0 {
				m.togglePlugin(false)
			}
		}
	}

	// Update active list
	var cmd tea.Cmd
	if m.activeTab == 0 {
		m.installedList, cmd = m.installedList.Update(msg)
	} else {
		m.availableList, cmd = m.availableList.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *PluginManagerModel) executeAction() {
	if m.selected == nil {
		return
	}

	switch m.confirmAction {
	case "install":
		// Move from available to installed
		newPlugin := *m.selected
		newPlugin.Status = "installed"
		m.installed = append(m.installed, newPlugin)

		// Remove from available
		for i, p := range m.available {
			if p.Name == m.selected.Name {
				m.available = append(m.available[:i], m.available[i+1:]...)
				break
			}
		}
		m.refreshLists()

	case "uninstall":
		// Move from installed to available
		newPlugin := *m.selected
		newPlugin.Status = "available"
		m.available = append(m.available, newPlugin)

		// Remove from installed
		for i, p := range m.installed {
			if p.Name == m.selected.Name {
				m.installed = append(m.installed[:i], m.installed[i+1:]...)
				break
			}
		}
		m.refreshLists()
	}

	m.selected = nil
}

func (m *PluginManagerModel) togglePlugin(enable bool) {
	if item, ok := m.installedList.SelectedItem().(PluginItem); ok {
		for i := range m.installed {
			if m.installed[i].Name == item.Name {
				if enable {
					m.installed[i].Status = "installed"
				} else {
					m.installed[i].Status = "disabled"
				}
				break
			}
		}
		m.refreshLists()
	}
}

func (m *PluginManagerModel) refreshLists() {
	installedItems := make([]list.Item, len(m.installed))
	for i, p := range m.installed {
		installedItems[i] = p
	}
	m.installedList.SetItems(installedItems)

	availableItems := make([]list.Item, len(m.available))
	for i, p := range m.available {
		availableItems[i] = p
	}
	m.availableList.SetItems(availableItems)
}

// View renders the plugin manager.
func (m *PluginManagerModel) View(width, height int) string {
	if m.width == 0 {
		m.width = width
		m.height = height
		m.installedList.SetWidth(width - 4)
		m.installedList.SetHeight(height - 10)
		m.availableList.SetWidth(width - 4)
		m.availableList.SetHeight(height - 10)
	}

	if m.confirmModal {
		return m.renderConfirmModal()
	}

	if m.showDetails && m.selected != nil {
		return m.renderDetails()
	}

	// Tab bar
	tabBar := m.renderTabBar()

	// Active list
	var listView string
	if m.activeTab == 0 {
		listView = m.installedList.View()
	} else {
		listView = m.availableList.View()
	}

	// Help bar
	var helpBar string
	if m.activeTab == 0 {
		helpBar = subtitleStyle.Render("[e] enable | [d] disable | [u] uninstall | [enter] details | [tab] switch")
	} else {
		helpBar = subtitleStyle.Render("[i] install | [enter] details | [tab] switch | [/] search")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		"",
		listView,
		"",
		helpBar,
	)
}

func (m *PluginManagerModel) renderTabBar() string {
	tabs := []string{"Installed", "Available"}
	var renderedTabs []string

	for i, tab := range tabs {
		style := inactiveTabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		count := len(m.installed)
		if i == 1 {
			count = len(m.available)
		}
		renderedTabs = append(renderedTabs, style.Render(fmt.Sprintf("%s (%d)", tab, count)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func (m *PluginManagerModel) renderConfirmModal() string {
	if m.selected == nil {
		return ""
	}

	var action, icon string
	switch m.confirmAction {
	case "install":
		action = "install"
		icon = "📥"
	case "uninstall":
		action = "uninstall"
		icon = "🗑️"
	}

	content := fmt.Sprintf(`
%s Are you sure you want to %s "%s"?

Permissions required:
%s

Press [y] to confirm or [Esc] to cancel
`,
		icon, action, m.selected.Name,
		strings.Join(m.selected.Permissions, "\n  • "),
	)

	return highlightBoxStyle.
		Width(m.width - 20).
		Render(content)
}

func (m *PluginManagerModel) renderDetails() string {
	p := m.selected
	header := titleStyle.Render(fmt.Sprintf("🔌 Plugin: %s", p.Name))

	permList := strings.Join(p.Permissions, "\n  • ")
	if len(p.Permissions) > 0 {
		permList = "• " + permList
	}

	details := fmt.Sprintf(`
Name:        %s
Version:     %s
Author:      %s
Status:      %s
Size:        %s
Downloads:   %d

Description:
  %s

Permissions:
  %s
`,
		p.Name,
		p.Version,
		p.Author,
		renderStatus(p.Status),
		p.Size,
		p.Downloads,
		p.Desc,
		permList,
	)

	var helpBar string
	if p.Status == "available" {
		helpBar = subtitleStyle.Render("[i] install | [Esc] back")
	} else {
		helpBar = subtitleStyle.Render("[e] enable | [d] disable | [u] uninstall | [c] configure | [Esc] back")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		boxStyle.Width(m.width - 4).Render(details),
		"",
		helpBar,
	)
}

