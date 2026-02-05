package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogLevel represents a log severity level.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	return []string{"DEBUG", "INFO", "WARN", "ERROR"}[l]
}

func (l LogLevel) Style() lipgloss.Style {
	switch l {
	case LogLevelDebug:
		return logDebugStyle
	case LogLevelInfo:
		return logInfoStyle
	case LogLevelWarn:
		return logWarnStyle
	case LogLevelError:
		return logErrorStyle
	default:
		return logInfoStyle
	}
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Source    string
	Fields    map[string]string
}

// LogViewerModel represents the log viewer state.
type LogViewerModel struct {
	// All logs
	allLogs []LogEntry
	// Filtered logs (after applying level filter and search)
	filteredLogs []LogEntry
	// Viewport for scrolling
	viewport viewport.Model
	// Search input
	searchInput textinput.Model
	// Filter settings
	minLevel    LogLevel
	searchQuery string
	searching   bool
	// UI state
	ready       bool
	width       int
	height      int
	autoScroll  bool
	showDetails bool
	selectedIdx int
	// Key bindings
	keys logViewerKeyMap
}

type logViewerKeyMap struct {
	FilterDebug key.Binding
	FilterInfo  key.Binding
	FilterWarn  key.Binding
	FilterError key.Binding
	Search      key.Binding
	CancelSearch key.Binding
	ToggleAuto  key.Binding
	Clear       key.Binding
	Details     key.Binding
}

func defaultLogViewerKeyMap() logViewerKeyMap {
	return logViewerKeyMap{
		FilterDebug: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "all levels")),
		FilterInfo:  key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "info+")),
		FilterWarn:  key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "warn+")),
		FilterError: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "errors")),
		Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		CancelSearch: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		ToggleAuto:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "follow")),
		Clear:       key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
		Details:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
	}
}

// NewLogViewerModel creates a new log viewer model.
func NewLogViewerModel() *LogViewerModel {
	ti := textinput.New()
	ti.Placeholder = "Search logs..."
	ti.CharLimit = 100
	ti.Width = 40

	vp := viewport.New(80, 20)
	vp.SetContent("")

	// Sample logs for demo
	sampleLogs := []LogEntry{
		{Timestamp: time.Now().Add(-5 * time.Minute), Level: LogLevelInfo, Message: "Daemon started", Source: "daemon"},
		{Timestamp: time.Now().Add(-4 * time.Minute), Level: LogLevelInfo, Message: "Loading plugins...", Source: "plugin"},
		{Timestamp: time.Now().Add(-3 * time.Minute), Level: LogLevelWarn, Message: "No plugins found in plugins directory", Source: "plugin"},
		{Timestamp: time.Now().Add(-2 * time.Minute), Level: LogLevelInfo, Message: "TSDB initialized successfully", Source: "tsdb"},
		{Timestamp: time.Now().Add(-1 * time.Minute), Level: LogLevelDebug, Message: "Listening on unix socket", Source: "daemon"},
		{Timestamp: time.Now(), Level: LogLevelInfo, Message: "Ready to accept connections", Source: "daemon"},
	}

	m := &LogViewerModel{
		allLogs:     sampleLogs,
		viewport:    vp,
		searchInput: ti,
		minLevel:    LogLevelDebug,
		autoScroll:  true,
		keys:        defaultLogViewerKeyMap(),
	}
	m.applyFilters()
	return m
}

// Init initializes the log viewer.
func (m *LogViewerModel) Init() tea.Cmd {
	return nil
}

// AddLog adds a new log entry.
func (m *LogViewerModel) AddLog(entry LogEntry) {
	m.allLogs = append(m.allLogs, entry)
	m.applyFilters()
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

// Update handles log viewer updates.
func (m *LogViewerModel) Update(msg tea.Msg) (*LogViewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-8)
			m.viewport.SetContent(m.renderLogs())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 8
		}

	case tea.KeyMsg:
		if m.searching {
			switch {
			case key.Matches(msg, m.keys.CancelSearch):
				m.searching = false
				m.searchInput.Blur()
			case msg.Type == tea.KeyEnter:
				m.searchQuery = m.searchInput.Value()
				m.searching = false
				m.searchInput.Blur()
				m.applyFilters()
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.FilterDebug):
			m.minLevel = LogLevelDebug
			m.applyFilters()
		case key.Matches(msg, m.keys.FilterInfo):
			m.minLevel = LogLevelInfo
			m.applyFilters()
		case key.Matches(msg, m.keys.FilterWarn):
			m.minLevel = LogLevelWarn
			m.applyFilters()
		case key.Matches(msg, m.keys.FilterError):
			m.minLevel = LogLevelError
			m.applyFilters()
		case key.Matches(msg, m.keys.Search):
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, m.keys.ToggleAuto):
			m.autoScroll = !m.autoScroll
		case key.Matches(msg, m.keys.Clear):
			m.allLogs = nil
			m.filteredLogs = nil
			m.viewport.SetContent("")
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the log viewer.
func (m *LogViewerModel) View(width, height int) string {
	if !m.ready {
		m.width = width
		m.height = height
		m.viewport = viewport.New(width-4, height-8)
		m.viewport.SetContent(m.renderLogs())
		m.ready = true
	}

	// Header
	header := titleStyle.Render("📜 Log Viewer")

	// Filter bar
	filterBar := m.renderFilterBar()

	// Search bar (if searching)
	var searchBar string
	if m.searching {
		searchBar = boxStyle.Width(width - 8).Render(m.searchInput.View())
	}

	// Viewport with logs
	logContent := boxStyle.Width(width - 4).Height(height - 10).Render(m.viewport.View())

	// Status bar
	statusBar := m.renderStatusBar()

	var parts []string
	parts = append(parts, header, "", filterBar)
	if searchBar != "" {
		parts = append(parts, searchBar)
	}
	parts = append(parts, logContent, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *LogViewerModel) renderFilterBar() string {
	levels := []struct {
		level LogLevel
		key   string
	}{
		{LogLevelDebug, "1"},
		{LogLevelInfo, "2"},
		{LogLevelWarn, "3"},
		{LogLevelError, "4"},
	}

	var parts []string
	for _, l := range levels {
		style := lipgloss.NewStyle().Padding(0, 1)
		if m.minLevel <= l.level {
			style = style.Bold(true).Foreground(l.level.Style().GetForeground())
		} else {
			style = style.Foreground(mutedColor)
		}
		parts = append(parts, style.Render(fmt.Sprintf("[%s] %s", l.key, l.level.String())))
	}

	filterStr := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	// Add search indicator
	searchStr := ""
	if m.searchQuery != "" {
		searchStr = fmt.Sprintf(" | Search: %q", m.searchQuery)
	}

	// Add auto-scroll indicator
	autoStr := ""
	if m.autoScroll {
		autoStr = " | [f] Following"
	} else {
		autoStr = " | [f] Paused"
	}

	return subtitleStyle.Render(filterStr + searchStr + autoStr + " | [/] search | [c] clear")
}

func (m *LogViewerModel) renderStatusBar() string {
	total := len(m.allLogs)
	filtered := len(m.filteredLogs)
	scroll := int(m.viewport.ScrollPercent() * 100)

	return subtitleStyle.Render(fmt.Sprintf("Showing %d of %d logs | Scroll: %d%%", filtered, total, scroll))
}

func (m *LogViewerModel) applyFilters() {
	m.filteredLogs = nil

	var re *regexp.Regexp
	if m.searchQuery != "" {
		re, _ = regexp.Compile("(?i)" + regexp.QuoteMeta(m.searchQuery))
	}

	for _, log := range m.allLogs {
		// Level filter
		if log.Level < m.minLevel {
			continue
		}

		// Search filter
		if re != nil && !re.MatchString(log.Message) && !re.MatchString(log.Source) {
			continue
		}

		m.filteredLogs = append(m.filteredLogs, log)
	}

	m.viewport.SetContent(m.renderLogs())
}

func (m *LogViewerModel) renderLogs() string {
	if len(m.filteredLogs) == 0 {
		return subtitleStyle.Render("No logs to display")
	}

	var lines []string
	for _, log := range m.filteredLogs {
		// Format: [LEVEL] TIMESTAMP SOURCE: MESSAGE
		ts := log.Timestamp.Format("15:04:05.000")
		levelStyle := log.Level.Style()
		levelStr := levelStyle.Render(fmt.Sprintf("[%-5s]", log.Level.String()))
		srcStr := subtitleStyle.Render(log.Source + ":")

		line := fmt.Sprintf("%s %s %s %s", levelStr, ts, srcStr, log.Message)

		// Highlight search matches
		if m.searchQuery != "" {
			re, _ := regexp.Compile("(?i)(" + regexp.QuoteMeta(m.searchQuery) + ")")
			line = re.ReplaceAllString(line, highlightStyle.Render("$1"))
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Highlight style for search matches
var highlightStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#F59E0B")).
	Foreground(lipgloss.Color("#1F2937")).
	Bold(true)

