package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

// WorkflowItem represents a workflow execution in the list.
type WorkflowItem struct {
	ID           uuid.UUID
	WorkflowName string
	Status       string
	Steps        []WorkflowStepItem
	StartedAt    time.Time
	CompletedAt  *time.Time
	Duration     time.Duration
	Error        string
}

// WorkflowStepItem represents a step in a workflow execution.
type WorkflowStepItem struct {
	ID         string
	Name       string
	Status     string
	RetryCount int
	Duration   time.Duration
	Error      string
}

func (w WorkflowItem) Title() string {
	icon := "📋"
	switch w.Status {
	case "running":
		icon = "🔄"
	case "completed":
		icon = "✅"
	case "failed":
		icon = "❌"
	case "cancelled":
		icon = "🚫"
	case "pending":
		icon = "⏳"
	}
	return fmt.Sprintf("%s %s", icon, w.WorkflowName)
}

func (w WorkflowItem) Description() string {
	dur := "-"
	if w.Duration > 0 {
		dur = w.Duration.Truncate(time.Millisecond).String()
	}
	return fmt.Sprintf("Status: %s | Steps: %d | Duration: %s | Started: %s",
		w.Status, len(w.Steps), dur, w.StartedAt.Format("15:04:05"))
}

func (w WorkflowItem) FilterValue() string {
	return w.WorkflowName + " " + w.Status
}

// WorkflowManagerModel represents the workflow manager TUI state.
type WorkflowManagerModel struct {
	list        list.Model
	executions  []WorkflowItem
	selected    *WorkflowItem
	width       int
	height      int
	showDetails bool
	keys        workflowKeyMap
}

type workflowKeyMap struct {
	Run     key.Binding
	Cancel  key.Binding
	Refresh key.Binding
	Details key.Binding
}

func newWorkflowKeyMap() workflowKeyMap {
	return workflowKeyMap{
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run workflow"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "cancel"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "toggle details"),
		),
	}
}

// NewWorkflowManager creates a new workflow manager model.
func NewWorkflowManager() *WorkflowManagerModel {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("170")).
		BorderLeftForeground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("241")).
		BorderLeftForeground(lipgloss.Color("170"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Workflow Executions"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	return &WorkflowManagerModel{
		list: l,
		keys: newWorkflowKeyMap(),
	}
}

// Init initializes the workflow manager.
func (m *WorkflowManagerModel) Init() tea.Cmd {
	return m.refreshExecutions()
}

// Update handles messages for the workflow manager.
func (m *WorkflowManagerModel) Update(msg tea.Msg) (*WorkflowManagerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Details):
			m.showDetails = !m.showDetails
		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.refreshExecutions())
		case key.Matches(msg, m.keys.Cancel):
			if m.selected != nil && m.selected.Status == "running" {
				cmds = append(cmds, m.cancelWorkflow(m.selected.ID))
			}
		}
	case refreshWorkflowsMsg:
		m.updateList(msg.executions)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	// Update selected item
	if item, ok := m.list.SelectedItem().(WorkflowItem); ok {
		m.selected = &item
	}

	return m, tea.Batch(cmds...)
}

// View renders the workflow manager.
func (m *WorkflowManagerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	if m.showDetails && m.selected != nil {
		// Split view: list on left, details on right
		listWidth := m.width / 2
		detailsWidth := m.width - listWidth - 1

		// Render list
		m.list.SetWidth(listWidth)
		listView := m.list.View()

		// Render details
		detailsView := m.renderDetails(detailsWidth)

		// Join horizontally
		lines := strings.Split(listView, "\n")
		detailLines := strings.Split(detailsView, "\n")

		for i := 0; i < len(lines) || i < len(detailLines); i++ {
			left := ""
			right := ""
			if i < len(lines) {
				left = lines[i]
			}
			if i < len(detailLines) {
				right = detailLines[i]
			}
			b.WriteString(left)
			b.WriteString(" │ ")
			b.WriteString(right)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(m.list.View())
	}

	// Help footer
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	help := helpStyle.Render("r: run • c: cancel • d: details • R: refresh • /: filter")
	b.WriteString("\n" + help)

	return b.String()
}

// renderDetails renders the workflow details panel.
func (m *WorkflowManagerModel) renderDetails(width int) string {
	if m.selected == nil {
		return "No workflow selected"
	}

	w := m.selected
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	b.WriteString(titleStyle.Render("Workflow Details"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Name: "))
	b.WriteString(valueStyle.Render(w.WorkflowName))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Status: "))
	b.WriteString(m.statusStyle(w.Status).Render(w.Status))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Started: "))
	b.WriteString(valueStyle.Render(w.StartedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	if w.CompletedAt != nil {
		b.WriteString(labelStyle.Render("Completed: "))
		b.WriteString(valueStyle.Render(w.CompletedAt.Format("2006-01-02 15:04:05")))
		b.WriteString("\n")
	}

	b.WriteString(labelStyle.Render("Duration: "))
	b.WriteString(valueStyle.Render(w.Duration.Truncate(time.Millisecond).String()))
	b.WriteString("\n")

	if w.Error != "" {
		b.WriteString("\n")
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render("Error: " + w.Error))
		b.WriteString("\n")
	}

	// Steps
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Steps"))
	b.WriteString("\n")

	for _, step := range w.Steps {
		icon := m.stepIcon(step.Status)
		b.WriteString(fmt.Sprintf("  %s %s (%s)\n", icon, step.Name, step.Duration.Truncate(time.Millisecond)))
		if step.Error != "" {
			b.WriteString(fmt.Sprintf("     └─ %s\n", step.Error))
		}
	}

	return b.String()
}

func (m *WorkflowManagerModel) statusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	case "completed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	case "cancelled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	}
}

func (m *WorkflowManagerModel) stepIcon(status string) string {
	switch status {
	case "running":
		return "🔄"
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "skipped":
		return "⏭️"
	default:
		return "⏳"
	}
}

// SetSize sets the terminal size.
func (m *WorkflowManagerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-4)
}

// Message types
type refreshWorkflowsMsg struct {
	executions []WorkflowItem
}

// Helper methods
func (m *WorkflowManagerModel) refreshExecutions() tea.Cmd {
	return func() tea.Msg {
		// TODO: Fetch from daemon
		return refreshWorkflowsMsg{executions: []WorkflowItem{}}
	}
}

func (m *WorkflowManagerModel) cancelWorkflow(id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		// TODO: Call daemon to cancel
		return refreshWorkflowsMsg{executions: m.executions}
	}
}

func (m *WorkflowManagerModel) updateList(executions []WorkflowItem) {
	m.executions = executions
	items := make([]list.Item, len(executions))
	for i, e := range executions {
		items[i] = e
	}
	m.list.SetItems(items)
}

