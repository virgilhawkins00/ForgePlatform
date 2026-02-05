package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

// TaskItem represents a task in the list.
type TaskItem struct {
	ID         uuid.UUID
	Type       string
	Status     string
	Payload    string
	CreatedAt  time.Time
	StartedAt  *time.Time
	RetryCount int
	MaxRetries int
	Error      string
}

func (t TaskItem) Title() string {
	icon := "📋"
	switch t.Status {
	case "running":
		icon = "⚡"
	case "completed":
		icon = "✅"
	case "failed":
		icon = "❌"
	case "pending":
		icon = "⏳"
	}
	return fmt.Sprintf("%s %s", icon, t.Type)
}

func (t TaskItem) Description() string {
	return fmt.Sprintf("Status: %s | Retries: %d/%d | Created: %s",
		t.Status, t.RetryCount, t.MaxRetries, t.CreatedAt.Format("15:04:05"))
}

func (t TaskItem) FilterValue() string {
	return t.Type + " " + t.Status
}

// TaskManagerModel represents the task manager state.
type TaskManagerModel struct {
	// Task list
	list     list.Model
	tasks    []TaskItem
	selected *TaskItem

	// Wizard state
	wizardActive bool
	wizardStep   int
	wizardInputs []textinput.Model
	wizardTask   TaskItem

	// UI state
	width       int
	height      int
	showDetails bool

	// Key bindings
	keys taskManagerKeyMap
}

type taskManagerKeyMap struct {
	Create  key.Binding
	Delete  key.Binding
	Retry   key.Binding
	Cancel  key.Binding
	Details key.Binding
	Filter  key.Binding
	Refresh key.Binding
	Confirm key.Binding
	Back    key.Binding
}

func defaultTaskManagerKeyMap() taskManagerKeyMap {
	return taskManagerKeyMap{
		Create:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new task")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Retry:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Cancel:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "cancel")),
		Details: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Filter:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
		Refresh: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

// NewTaskManagerModel creates a new task manager model.
func NewTaskManagerModel() *TaskManagerModel {
	// Create sample tasks
	sampleTasks := []TaskItem{
		{
			ID: uuid.New(), Type: "backup", Status: "completed",
			CreatedAt: time.Now().Add(-1 * time.Hour), RetryCount: 0, MaxRetries: 3,
			Payload: `{"target": "/data", "destination": "s3://backup"}`,
		},
		{
			ID: uuid.New(), Type: "sync", Status: "running",
			CreatedAt: time.Now().Add(-30 * time.Minute), RetryCount: 0, MaxRetries: 3,
			Payload: `{"source": "api", "target": "warehouse"}`,
		},
		{
			ID: uuid.New(), Type: "cleanup", Status: "pending",
			CreatedAt: time.Now().Add(-10 * time.Minute), RetryCount: 0, MaxRetries: 3,
			Payload: `{"older_than": "7d"}`,
		},
		{
			ID: uuid.New(), Type: "export", Status: "failed",
			CreatedAt: time.Now().Add(-2 * time.Hour), RetryCount: 3, MaxRetries: 3,
			Error: "connection timeout", Payload: `{"format": "csv"}`,
		},
	}

	// Create list items
	items := make([]list.Item, len(sampleTasks))
	for i, t := range sampleTasks {
		items[i] = t
	}

	// Create list model
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(primaryColor).BorderForeground(primaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(mutedColor).BorderForeground(primaryColor)

	l := list.New(items, delegate, 80, 20)
	l.Title = "📋 Task Queue"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	// Create wizard inputs
	inputs := make([]textinput.Model, 3)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Task type (e.g., backup, sync, cleanup)"
	inputs[0].Focus()
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Payload JSON (e.g., {\"target\": \"/data\"})"
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Max retries (default: 3)"

	return &TaskManagerModel{
		list:         l,
		tasks:        sampleTasks,
		wizardInputs: inputs,
		keys:         defaultTaskManagerKeyMap(),
	}
}

// Init initializes the task manager.
func (m *TaskManagerModel) Init() tea.Cmd {
	return nil
}

// Update handles task manager updates.
func (m *TaskManagerModel) Update(msg tea.Msg) (*TaskManagerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 8)

	case tea.KeyMsg:
		// Wizard mode
		if m.wizardActive {
			return m.updateWizard(msg)
		}

		// Details mode
		if m.showDetails {
			if key.Matches(msg, m.keys.Back) {
				m.showDetails = false
				m.selected = nil
			}
			return m, nil
		}

		// Normal mode
		switch {
		case key.Matches(msg, m.keys.Create):
			m.wizardActive = true
			m.wizardStep = 0
			m.wizardTask = TaskItem{ID: uuid.New(), MaxRetries: 3}
			m.wizardInputs[0].Focus()
			return m, textinput.Blink

		case key.Matches(msg, m.keys.Details):
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.selected = &item
				m.showDetails = true
			}

		case key.Matches(msg, m.keys.Retry):
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.retryTask(item.ID)
			}

		case key.Matches(msg, m.keys.Cancel):
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.cancelTask(item.ID)
			}

		case key.Matches(msg, m.keys.Delete):
			if item, ok := m.list.SelectedItem().(TaskItem); ok {
				m.deleteTask(item.ID)
			}
		}
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *TaskManagerModel) updateWizard(msg tea.KeyMsg) (*TaskManagerModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.wizardActive = false
		m.wizardStep = 0
		for i := range m.wizardInputs {
			m.wizardInputs[i].SetValue("")
			m.wizardInputs[i].Blur()
		}
		return m, nil

	case tea.KeyEnter:
		// Save current step value
		switch m.wizardStep {
		case 0:
			m.wizardTask.Type = m.wizardInputs[0].Value()
		case 1:
			m.wizardTask.Payload = m.wizardInputs[1].Value()
		case 2:
			// Parse max retries
			m.wizardTask.MaxRetries = 3 // default
			m.wizardTask.CreatedAt = time.Now()
			m.wizardTask.Status = "pending"

			// Add task
			m.tasks = append(m.tasks, m.wizardTask)
			m.refreshList()

			// Reset wizard
			m.wizardActive = false
			m.wizardStep = 0
			for i := range m.wizardInputs {
				m.wizardInputs[i].SetValue("")
				m.wizardInputs[i].Blur()
			}
			return m, nil
		}

		// Move to next step
		m.wizardInputs[m.wizardStep].Blur()
		m.wizardStep++
		if m.wizardStep < len(m.wizardInputs) {
			m.wizardInputs[m.wizardStep].Focus()
			return m, textinput.Blink
		}
		return m, nil

	case tea.KeyShiftTab:
		if m.wizardStep > 0 {
			m.wizardInputs[m.wizardStep].Blur()
			m.wizardStep--
			m.wizardInputs[m.wizardStep].Focus()
			return m, textinput.Blink
		}
	}

	// Update current input
	var cmd tea.Cmd
	m.wizardInputs[m.wizardStep], cmd = m.wizardInputs[m.wizardStep].Update(msg)
	return m, cmd
}

func (m *TaskManagerModel) refreshList() {
	items := make([]list.Item, len(m.tasks))
	for i, t := range m.tasks {
		items[i] = t
	}
	m.list.SetItems(items)
}

func (m *TaskManagerModel) retryTask(id uuid.UUID) {
	for i := range m.tasks {
		if m.tasks[i].ID == id && m.tasks[i].Status == "failed" {
			m.tasks[i].Status = "pending"
			m.tasks[i].RetryCount = 0
			m.tasks[i].Error = ""
		}
	}
	m.refreshList()
}

func (m *TaskManagerModel) cancelTask(id uuid.UUID) {
	for i := range m.tasks {
		if m.tasks[i].ID == id && (m.tasks[i].Status == "pending" || m.tasks[i].Status == "running") {
			m.tasks[i].Status = "cancelled"
		}
	}
	m.refreshList()
}

func (m *TaskManagerModel) deleteTask(id uuid.UUID) {
	for i := range m.tasks {
		if m.tasks[i].ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			break
		}
	}
	m.refreshList()
}

// View renders the task manager.
func (m *TaskManagerModel) View(width, height int) string {
	if m.width == 0 {
		m.width = width
		m.height = height
		m.list.SetWidth(width - 4)
		m.list.SetHeight(height - 8)
	}

	if m.wizardActive {
		return m.renderWizard()
	}

	if m.showDetails && m.selected != nil {
		return m.renderDetails()
	}

	// Main task list view
	helpBar := subtitleStyle.Render("[n] new | [d] delete | [r] retry | [x] cancel | [enter] details | [/] filter")

	return lipgloss.JoinVertical(lipgloss.Left,
		m.list.View(),
		"",
		helpBar,
	)
}

func (m *TaskManagerModel) renderWizard() string {
	header := titleStyle.Render("✨ Create New Task")

	steps := []string{"Task Type", "Payload (JSON)", "Max Retries"}

	var content strings.Builder
	content.WriteString("\n")

	for i, step := range steps {
		prefix := "  "
		if i == m.wizardStep {
			prefix = "▶ "
		} else if i < m.wizardStep {
			prefix = "✓ "
		}

		content.WriteString(fmt.Sprintf("%s%s\n", prefix, step))
		if i == m.wizardStep {
			content.WriteString("   ")
			content.WriteString(m.wizardInputs[i].View())
			content.WriteString("\n")
		} else if i < m.wizardStep {
			content.WriteString(fmt.Sprintf("   %s\n", m.wizardInputs[i].Value()))
		}
		content.WriteString("\n")
	}

	helpBar := subtitleStyle.Render("[Enter] next step | [Shift+Tab] previous | [Esc] cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		boxStyle.Width(m.width - 4).Render(content.String()),
		helpBar,
	)
}

func (m *TaskManagerModel) renderDetails() string {
	t := m.selected
	header := titleStyle.Render(fmt.Sprintf("📋 Task Details: %s", t.Type))

	details := fmt.Sprintf(`
ID:         %s
Type:       %s
Status:     %s
Created:    %s
Retries:    %d / %d
Payload:    %s
`,
		t.ID.String(),
		t.Type,
		renderStatus(t.Status),
		t.CreatedAt.Format("2006-01-02 15:04:05"),
		t.RetryCount, t.MaxRetries,
		t.Payload,
	)

	if t.Error != "" {
		details += fmt.Sprintf("Error:      %s\n", logErrorStyle.Render(t.Error))
	}

	helpBar := subtitleStyle.Render("[Esc] back | [r] retry | [x] cancel | [d] delete")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		boxStyle.Width(m.width - 4).Render(details),
		"",
		helpBar,
	)
}

