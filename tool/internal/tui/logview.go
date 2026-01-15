package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxLogEntries = 10000 // Maximum log entries to keep
)

// LogView is a scrollable log viewport with search highlighting.
type LogView struct {
	viewport viewport.Model

	// All logs and filtered view
	allLogs      []LogEntry
	filteredLogs []LogEntry

	// Filter settings
	serviceFilter string // Empty means show all
	searchTerm    string

	// Service colors mapping
	serviceColors map[string]lipgloss.Color

	// Auto-scroll to bottom
	autoScroll bool

	// Dimensions
	width  int
	height int

	// Max service name width for alignment
	maxServiceWidth int
}

// NewLogView creates a new log view.
func NewLogView(services []string) *LogView {
	vp := viewport.New(80, 20)
	vp.Style = LogViewStyle

	// Build service color map
	colors := make(map[string]lipgloss.Color)
	maxWidth := 0
	for i, name := range services {
		colors[name] = GetServiceColor(i)
		if len(name) > maxWidth {
			maxWidth = len(name)
		}
	}

	return &LogView{
		viewport:        vp,
		allLogs:         make([]LogEntry, 0, 1000),
		filteredLogs:    make([]LogEntry, 0, 1000),
		serviceColors:   colors,
		autoScroll:      true,
		maxServiceWidth: maxWidth,
	}
}

// SetSize sets the viewport size.
func (l *LogView) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.viewport.Width = width
	l.viewport.Height = height
	l.refresh()
}

// SetServiceFilter sets the service filter.
func (l *LogView) SetServiceFilter(service string) {
	l.serviceFilter = service
	l.refresh()
}

// SetSearchTerm sets the search term for highlighting.
func (l *LogView) SetSearchTerm(term string) {
	l.searchTerm = strings.ToLower(term)
	l.refresh()
}

// AddLog adds a new log entry.
func (l *LogView) AddLog(entry LogEntry) {
	l.allLogs = append(l.allLogs, entry)

	// Trim if too many logs
	if len(l.allLogs) > maxLogEntries {
		l.allLogs = l.allLogs[len(l.allLogs)-maxLogEntries:]
	}

	// Check if it matches current filter
	if l.matchesFilter(entry) {
		l.filteredLogs = append(l.filteredLogs, entry)
		l.updateViewport()
	}
}

// matchesFilter checks if an entry matches the current filters.
func (l *LogView) matchesFilter(entry LogEntry) bool {
	// Service filter
	if l.serviceFilter != "" && l.serviceFilter != "All" && entry.Service != l.serviceFilter {
		return false
	}

	// Search filter
	if l.searchTerm != "" {
		if !strings.Contains(strings.ToLower(entry.Message), l.searchTerm) {
			return false
		}
	}

	return true
}

// refresh rebuilds the filtered log list.
func (l *LogView) refresh() {
	l.filteredLogs = l.filteredLogs[:0]

	for _, entry := range l.allLogs {
		if l.matchesFilter(entry) {
			l.filteredLogs = append(l.filteredLogs, entry)
		}
	}

	l.updateViewport()
}

// updateViewport updates the viewport content.
func (l *LogView) updateViewport() {
	var lines []string

	for _, entry := range l.filteredLogs {
		line := l.renderLogEntry(entry)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	l.viewport.SetContent(content)

	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

// renderLogEntry renders a single log entry with optional highlighting.
func (l *LogView) renderLogEntry(entry LogEntry) string {
	// Get service color
	color := l.serviceColors[entry.Service]
	if color == "" {
		color = ColorMuted
	}

	// Format service prefix with padding
	serviceStyle := LogPrefixStyle.Foreground(color)
	paddedName := fmt.Sprintf("%-*s", l.maxServiceWidth, entry.Service)
	prefix := serviceStyle.Render(fmt.Sprintf("[%s]", paddedName))

	// Add level indicator
	var levelIndicator string
	switch entry.Level {
	case LogLevelSuccess:
		levelIndicator = StateRunningStyle.Render("✓ ")
	case LogLevelWarning:
		levelIndicator = StateBuildingStyle.Render("! ")
	case LogLevelError:
		levelIndicator = StateErrorStyle.Render("✗ ")
	default:
		levelIndicator = ""
	}

	// Apply search highlighting
	message := entry.Message
	if l.searchTerm != "" {
		message = l.highlightText(message, l.searchTerm)
	}

	// Build full line
	fullLine := fmt.Sprintf("%s %s%s", prefix, levelIndicator, message)

	// Truncate to fit width using lipgloss (handles ANSI codes correctly)
	lineWidth := lipgloss.Width(fullLine)
	if lineWidth > l.width {
		// Truncate to width - 1 to make room for ellipsis
		truncStyle := lipgloss.NewStyle().MaxWidth(l.width - 1)
		fullLine = truncStyle.Render(fullLine) + "…"
	}

	return fullLine
}

// highlightText highlights occurrences of term in text.
func (l *LogView) highlightText(text, term string) string {
	if term == "" {
		return text
	}

	lower := strings.ToLower(text)
	var result strings.Builder
	lastEnd := 0

	for {
		idx := strings.Index(lower[lastEnd:], term)
		if idx == -1 {
			result.WriteString(text[lastEnd:])
			break
		}

		idx += lastEnd

		// Add text before match
		result.WriteString(text[lastEnd:idx])

		// Add highlighted match
		matchEnd := idx + len(term)
		highlighted := HighlightStyle.Render(text[idx:matchEnd])
		result.WriteString(highlighted)

		lastEnd = matchEnd
	}

	return result.String()
}

// Update handles input events.
func (l *LogView) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "home", "g":
			l.viewport.GotoTop()
			l.autoScroll = false
		case "end", "G":
			l.viewport.GotoBottom()
			l.autoScroll = true
		case "up", "k":
			l.viewport.ScrollUp(1)
			l.autoScroll = false
		case "down", "j":
			l.viewport.ScrollDown(1)
			// Re-enable auto-scroll if at bottom
			if l.viewport.AtBottom() {
				l.autoScroll = true
			}
		case "pgup", "ctrl+u":
			l.viewport.HalfPageUp()
			l.autoScroll = false
		case "pgdown", "ctrl+d":
			l.viewport.HalfPageDown()
			if l.viewport.AtBottom() {
				l.autoScroll = true
			}
		}
	}

	l.viewport, cmd = l.viewport.Update(msg)
	return cmd
}

// View renders the log view.
func (l *LogView) View() string {
	return l.viewport.View()
}

// ScrollPercent returns the current scroll position as a percentage.
func (l *LogView) ScrollPercent() float64 {
	return l.viewport.ScrollPercent()
}

// LogCount returns the number of filtered logs.
func (l *LogView) LogCount() int {
	return len(l.filteredLogs)
}

// TotalLogCount returns the total number of logs.
func (l *LogView) TotalLogCount() int {
	return len(l.allLogs)
}

// Clear clears all logs.
func (l *LogView) Clear() {
	l.allLogs = l.allLogs[:0]
	l.filteredLogs = l.filteredLogs[:0]
	l.viewport.SetContent("")
}

// ToggleAutoScroll toggles auto-scroll behavior.
func (l *LogView) ToggleAutoScroll() {
	l.autoScroll = !l.autoScroll
	if l.autoScroll {
		l.viewport.GotoBottom()
	}
}

// IsAutoScrolling returns whether auto-scroll is enabled.
func (l *LogView) IsAutoScrolling() bool {
	return l.autoScroll
}
