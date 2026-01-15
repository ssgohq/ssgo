package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

const (
	maxLogEntries = 10000 // Maximum log entries to keep
)

// wrappedLine represents a single visual line after wrapping.
type wrappedLine struct {
	logIndex int    // Index into filteredLogs
	text     string // The rendered text for this line
}

// LogView is a scrollable log viewport with line selection and wrapping.
type LogView struct {
	// All logs and filtered view
	allLogs      []LogEntry
	filteredLogs []LogEntry

	// Wrapped lines for display
	wrappedLines []wrappedLine

	// Filter settings
	serviceFilter string // Empty means show all
	searchTerm    string

	// Service colors mapping
	serviceColors map[string]lipgloss.Color

	// Selection
	selectedLogIndex int // Current cursor position in filteredLogs (-1 = none)
	selectionAnchor  int // Anchor point for shift-selection (-1 = none)
	selectionStart   int // First log index in selection (inclusive)
	selectionEnd     int // Last log index in selection (inclusive)

	// Last copy info
	lastCopyCount int

	// Scroll position (first visible line index in wrappedLines)
	scrollOffset int

	// Dimensions
	width  int
	height int

	// Max service name width for alignment
	maxServiceWidth int
}

// NewLogView creates a new log view.
func NewLogView(services []string) *LogView {
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
		allLogs:          make([]LogEntry, 0, 1000),
		filteredLogs:     make([]LogEntry, 0, 1000),
		wrappedLines:     make([]wrappedLine, 0, 1000),
		serviceColors:    colors,
		selectedLogIndex: -1,
		selectionAnchor:  -1,
		selectionStart:   -1,
		selectionEnd:     -1,
		maxServiceWidth:  maxWidth,
	}
}

// SetSize sets the viewport size.
func (l *LogView) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.rebuildWrappedLines()
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
		wasAtBottom := l.isAtBottom()
		l.filteredLogs = append(l.filteredLogs, entry)
		l.addWrappedLinesForLog(len(l.filteredLogs) - 1)

		// Auto-scroll if was at bottom and no selection
		if wasAtBottom && l.selectedLogIndex == -1 {
			l.scrollToBottom()
		}
	}
}

// isContinuationLine returns true if the log message starts with whitespace,
// indicating it's a continuation of the previous message (like multi-line error output).
func (l *LogView) isContinuationLine(logIndex int) bool {
	if logIndex < 0 || logIndex >= len(l.filteredLogs) {
		return false
	}
	msg := l.filteredLogs[logIndex].Message
	if len(msg) == 0 {
		return false
	}
	// Check if starts with space or tab
	return msg[0] == ' ' || msg[0] == '\t'
}

// updateSelectionGroup expands selection to include continuation lines.
func (l *LogView) updateSelectionGroup() {
	if l.selectedLogIndex < 0 {
		l.selectionStart = -1
		l.selectionEnd = -1
		return
	}

	entry := l.filteredLogs[l.selectedLogIndex]

	// Find the start of the group (go backwards to find the first non-continuation)
	l.selectionStart = l.selectedLogIndex
	for l.selectionStart > 0 {
		prevEntry := l.filteredLogs[l.selectionStart-1]
		// Stop if different service or current line is not a continuation
		if prevEntry.Service != entry.Service {
			break
		}
		if !l.isContinuationLine(l.selectionStart) {
			break
		}
		l.selectionStart--
	}

	// Find the end of the group (go forward to find continuation lines)
	l.selectionEnd = l.selectedLogIndex
	for l.selectionEnd < len(l.filteredLogs)-1 {
		nextIdx := l.selectionEnd + 1
		nextEntry := l.filteredLogs[nextIdx]
		// Stop if different service or next line is not a continuation
		if nextEntry.Service != entry.Service {
			break
		}
		if !l.isContinuationLine(nextIdx) {
			break
		}
		l.selectionEnd++
	}
}

// isInSelectionGroup returns true if the log index is part of the current selection group.
func (l *LogView) isInSelectionGroup(logIndex int) bool {
	if l.selectionStart < 0 || l.selectionEnd < 0 {
		return false
	}
	return logIndex >= l.selectionStart && logIndex <= l.selectionEnd
}

// expandSelectionToContinuations expands selection to include continuation lines at boundaries.
func (l *LogView) expandSelectionToContinuations() {
	if l.selectionStart < 0 || l.selectionEnd < 0 {
		return
	}

	// Expand start backwards to include continuation lines
	for l.selectionStart > 0 && l.isContinuationLine(l.selectionStart) {
		prevEntry := l.filteredLogs[l.selectionStart-1]
		currEntry := l.filteredLogs[l.selectionStart]
		if prevEntry.Service == currEntry.Service {
			l.selectionStart--
		} else {
			break
		}
	}

	// Expand end forwards to include continuation lines
	for l.selectionEnd < len(l.filteredLogs)-1 {
		nextIdx := l.selectionEnd + 1
		if l.isContinuationLine(nextIdx) {
			nextEntry := l.filteredLogs[nextIdx]
			currEntry := l.filteredLogs[l.selectionEnd]
			if nextEntry.Service == currEntry.Service {
				l.selectionEnd++
			} else {
				break
			}
		} else {
			break
		}
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

	l.selectedLogIndex = -1
	l.selectionAnchor = -1
	l.selectionStart = -1
	l.selectionEnd = -1
	l.rebuildWrappedLines()
}

// rebuildWrappedLines rebuilds all wrapped lines from filtered logs.
func (l *LogView) rebuildWrappedLines() {
	l.wrappedLines = l.wrappedLines[:0]

	for i := range l.filteredLogs {
		l.addWrappedLinesForLog(i)
	}

	// Ensure scroll offset is valid
	l.clampScrollOffset()
}

// addWrappedLinesForLog adds wrapped lines for a single log entry.
func (l *LogView) addWrappedLinesForLog(logIndex int) {
	if logIndex < 0 || logIndex >= len(l.filteredLogs) {
		return
	}

	entry := l.filteredLogs[logIndex]
	lines := l.wrapLogEntry(entry)

	for _, line := range lines {
		l.wrappedLines = append(l.wrappedLines, wrappedLine{
			logIndex: logIndex,
			text:     line,
		})
	}
}

// wrapLogEntry wraps a log entry into multiple lines if needed.
func (l *LogView) wrapLogEntry(entry LogEntry) []string {
	if l.width <= 0 {
		return []string{entry.Message}
	}

	// Calculate prefix width
	prefixWidth := l.maxServiceWidth + 3 // "[name] "

	// Add level indicator width
	levelWidth := 0
	switch entry.Level {
	case LogLevelSuccess, LogLevelWarning, LogLevelError:
		levelWidth = 2 // "✓ " etc
	}

	totalPrefixWidth := prefixWidth + levelWidth
	contentWidth := l.width - totalPrefixWidth

	if contentWidth < 20 {
		contentWidth = 20
	}

	// Wrap the message
	wrapped := wordwrap.String(entry.Message, contentWidth)
	lines := strings.Split(wrapped, "\n")

	return lines
}

// renderLine renders a single wrapped line with optional selection highlight.
func (l *LogView) renderLine(wl wrappedLine, lineIndex int) string {
	if wl.logIndex < 0 || wl.logIndex >= len(l.filteredLogs) {
		return ""
	}

	entry := l.filteredLogs[wl.logIndex]
	isSelected := l.isInSelectionGroup(wl.logIndex)

	// Get service color
	color := l.serviceColors[entry.Service]
	if color == "" {
		color = ColorMuted
	}

	// Check if this is the first line of this log entry
	isFirstLine := lineIndex == 0 || l.wrappedLines[lineIndex-1].logIndex != wl.logIndex

	var prefix string
	var levelIndicator string

	if isFirstLine {
		paddedName := fmt.Sprintf("%-*s", l.maxServiceWidth, entry.Service)

		if isSelected {
			// When selected, use plain text for prefix (will be styled by SelectedLineStyle)
			prefix = fmt.Sprintf("[%s]", paddedName)
		} else {
			// Normal: colored service name
			serviceStyle := LogPrefixStyle.Foreground(color)
			prefix = serviceStyle.Render(fmt.Sprintf("[%s]", paddedName))
		}

		// Level indicator
		switch entry.Level {
		case LogLevelSuccess:
			if isSelected {
				levelIndicator = " ✓"
			} else {
				levelIndicator = " " + StateRunningStyle.Render("✓")
			}
		case LogLevelWarning:
			if isSelected {
				levelIndicator = " !"
			} else {
				levelIndicator = " " + StateBuildingStyle.Render("!")
			}
		case LogLevelError:
			if isSelected {
				levelIndicator = " ✗"
			} else {
				levelIndicator = " " + StateErrorStyle.Render("✗")
			}
		}
	} else {
		// Continuation line - use padding
		prefix = strings.Repeat(" ", l.maxServiceWidth+3)
		switch entry.Level {
		case LogLevelSuccess, LogLevelWarning, LogLevelError:
			prefix += "  " // Match level indicator width
		}
	}

	// Message text
	message := wl.text

	// Apply search highlighting only when not selected (to avoid style conflicts)
	if l.searchTerm != "" && !isSelected {
		message = l.highlightText(message, l.searchTerm)
	}

	// Build the line
	line := fmt.Sprintf("%s%s %s", prefix, levelIndicator, message)

	// Apply selection highlight
	if isSelected {
		// Pad to full width for highlight effect
		lineWidth := lipgloss.Width(line)
		if lineWidth < l.width {
			line = line + strings.Repeat(" ", l.width-lineWidth)
		}
		line = SelectedLineStyle.Render(line)
	}

	return line
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
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			l.moveToPrevLog(false)
		case "down", "j":
			l.moveToNextLog(false)
		case "shift+up", "K":
			l.moveToPrevLog(true)
		case "shift+down", "J":
			l.moveToNextLog(true)
		case "home", "g":
			l.selectFirstLog()
		case "end", "G":
			l.selectLastLog()
		case "pgup", "ctrl+u":
			l.pageUp()
		case "pgdown", "ctrl+d":
			l.pageDown()
		case "c":
			cmd, count := l.copySelectedLog()
			l.lastCopyCount = count
			return cmd
		case "esc":
			l.clearSelection()
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			return l.handleMouseClick(msg.Y)
		}

	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			l.scrollUp(3)
		} else if msg.Button == tea.MouseWheelDown {
			l.scrollDown(3)
		}
	}

	return nil
}

// moveToPrevLog moves selection to the previous log entry.
// If extend is true (shift held), extends the selection instead of moving it.
func (l *LogView) moveToPrevLog(extend bool) {
	if len(l.filteredLogs) == 0 {
		return
	}

	if l.selectedLogIndex == -1 {
		// Start from last visible log
		l.selectedLogIndex = l.getLogIndexAtVisualLine(l.scrollOffset + l.height - 1)
		if l.selectedLogIndex == -1 {
			l.selectedLogIndex = len(l.filteredLogs) - 1
		}
		l.selectionAnchor = l.selectedLogIndex
	} else if l.selectedLogIndex > 0 {
		if extend {
			// Set anchor if not set
			if l.selectionAnchor == -1 {
				l.selectionAnchor = l.selectedLogIndex
			}
			l.selectedLogIndex--
		} else {
			// Normal move - reset anchor and move before current selection
			if l.selectionStart > 0 {
				l.selectedLogIndex = l.selectionStart - 1
			}
			l.selectionAnchor = l.selectedLogIndex
		}
	}

	l.updateSelectionRange(extend)
	l.ensureSelectedVisible()
}

// moveToNextLog moves selection to the next log entry.
// If extend is true (shift held), extends the selection instead of moving it.
func (l *LogView) moveToNextLog(extend bool) {
	if len(l.filteredLogs) == 0 {
		return
	}

	if l.selectedLogIndex == -1 {
		// Start from first visible log
		l.selectedLogIndex = l.getLogIndexAtVisualLine(l.scrollOffset)
		if l.selectedLogIndex == -1 {
			l.selectedLogIndex = 0
		}
		l.selectionAnchor = l.selectedLogIndex
	} else if l.selectedLogIndex < len(l.filteredLogs)-1 {
		if extend {
			// Set anchor if not set
			if l.selectionAnchor == -1 {
				l.selectionAnchor = l.selectedLogIndex
			}
			l.selectedLogIndex++
		} else {
			// Normal move - reset anchor and move after current selection
			if l.selectionEnd < len(l.filteredLogs)-1 {
				l.selectedLogIndex = l.selectionEnd + 1
			}
			l.selectionAnchor = l.selectedLogIndex
		}
	}

	l.updateSelectionRange(extend)
	l.ensureSelectedVisible()
}

// updateSelectionRange updates selection range based on anchor and current position.
func (l *LogView) updateSelectionRange(extend bool) {
	if l.selectedLogIndex < 0 {
		l.selectionStart = -1
		l.selectionEnd = -1
		return
	}

	if extend && l.selectionAnchor >= 0 {
		// Selection is from anchor to current position
		if l.selectedLogIndex < l.selectionAnchor {
			l.selectionStart = l.selectedLogIndex
			l.selectionEnd = l.selectionAnchor
		} else {
			l.selectionStart = l.selectionAnchor
			l.selectionEnd = l.selectedLogIndex
		}
		// Expand to include continuation lines at boundaries
		l.expandSelectionToContinuations()
	} else {
		// Single selection - use continuation group logic
		l.updateSelectionGroup()
	}
}

// selectFirstLog selects the first log entry.
func (l *LogView) selectFirstLog() {
	if len(l.filteredLogs) == 0 {
		return
	}
	l.selectedLogIndex = 0
	l.selectionAnchor = 0
	l.updateSelectionGroup()
	l.ensureSelectedVisible()
}

// selectLastLog selects the last log entry.
func (l *LogView) selectLastLog() {
	if len(l.filteredLogs) == 0 {
		return
	}
	l.selectedLogIndex = len(l.filteredLogs) - 1
	l.selectionAnchor = l.selectedLogIndex
	l.updateSelectionGroup()
	l.ensureSelectedVisible()
}

// pageUp moves up by a page.
func (l *LogView) pageUp() {
	if len(l.filteredLogs) == 0 {
		return
	}

	// Move selection up by approximately a page worth of logs
	pageLogCount := l.height / 2
	if pageLogCount < 1 {
		pageLogCount = 1
	}

	if l.selectedLogIndex == -1 {
		l.selectedLogIndex = l.getLogIndexAtVisualLine(l.scrollOffset)
	}

	l.selectedLogIndex -= pageLogCount
	if l.selectedLogIndex < 0 {
		l.selectedLogIndex = 0
	}

	l.selectionAnchor = l.selectedLogIndex
	l.updateSelectionGroup()
	l.ensureSelectedVisible()
}

// pageDown moves down by a page.
func (l *LogView) pageDown() {
	if len(l.filteredLogs) == 0 {
		return
	}

	pageLogCount := l.height / 2
	if pageLogCount < 1 {
		pageLogCount = 1
	}

	if l.selectedLogIndex == -1 {
		l.selectedLogIndex = l.getLogIndexAtVisualLine(l.scrollOffset)
	}

	l.selectedLogIndex += pageLogCount
	if l.selectedLogIndex >= len(l.filteredLogs) {
		l.selectedLogIndex = len(l.filteredLogs) - 1
	}

	l.selectionAnchor = l.selectedLogIndex
	l.updateSelectionGroup()
	l.ensureSelectedVisible()
}

// clearSelection clears the current selection.
func (l *LogView) clearSelection() {
	l.selectedLogIndex = -1
	l.selectionAnchor = -1
	l.selectionStart = -1
	l.selectionEnd = -1
}

// copySelectedLog copies the selected log entries to clipboard.
// Returns the command and the number of lines copied.
func (l *LogView) copySelectedLog() (tea.Cmd, int) {
	if l.selectionStart < 0 || l.selectionEnd < 0 {
		return nil, 0
	}

	// Collect all log entries with their prefixes
	var lines []string
	for i := l.selectionStart; i <= l.selectionEnd; i++ {
		if i < len(l.filteredLogs) {
			entry := l.filteredLogs[i]
			line := fmt.Sprintf("[%s] %s", entry.Service, entry.Message)
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return nil, 0
	}

	text := strings.Join(lines, "\n")
	return tea.SetClipboard(text), len(lines)
}

// handleMouseClick handles a mouse click at the given y position.
func (l *LogView) handleMouseClick(y int) tea.Cmd {
	// y is relative to viewport, need to account for header (tabs + divider = 2 lines)
	lineIndex := l.scrollOffset + y - 2

	if lineIndex < 0 || lineIndex >= len(l.wrappedLines) {
		return nil
	}

	logIndex := l.wrappedLines[lineIndex].logIndex
	if logIndex < 0 || logIndex >= len(l.filteredLogs) {
		return nil
	}

	l.selectedLogIndex = logIndex
	l.selectionAnchor = logIndex
	l.updateSelectionGroup()

	// Copy on click
	cmd, count := l.copySelectedLog()
	l.lastCopyCount = count
	return cmd
}

// getLogIndexAtVisualLine returns the log index at a visual line.
func (l *LogView) getLogIndexAtVisualLine(lineIndex int) int {
	if lineIndex < 0 || lineIndex >= len(l.wrappedLines) {
		return -1
	}
	return l.wrappedLines[lineIndex].logIndex
}

// getFirstVisualLineForLog returns the first visual line index for a log.
func (l *LogView) getFirstVisualLineForLog(logIndex int) int {
	for i, wl := range l.wrappedLines {
		if wl.logIndex == logIndex {
			return i
		}
	}
	return -1
}

// getLastVisualLineForLog returns the last visual line index for a log.
func (l *LogView) getLastVisualLineForLog(logIndex int) int {
	lastLine := -1
	for i, wl := range l.wrappedLines {
		if wl.logIndex == logIndex {
			lastLine = i
		} else if wl.logIndex > logIndex {
			break
		}
	}
	return lastLine
}

// ensureSelectedVisible scrolls to make the selected log group visible.
func (l *LogView) ensureSelectedVisible() {
	if l.selectionStart < 0 || l.selectionEnd < 0 {
		return
	}

	// Get the visual line range for the entire selection group
	firstLine := l.getFirstVisualLineForLog(l.selectionStart)
	lastLine := l.getLastVisualLineForLog(l.selectionEnd)

	if firstLine < 0 || lastLine < 0 {
		return
	}

	// Scroll up if needed
	if firstLine < l.scrollOffset {
		l.scrollOffset = firstLine
	}

	// Scroll down if needed
	if lastLine >= l.scrollOffset+l.height {
		l.scrollOffset = lastLine - l.height + 1
	}

	l.clampScrollOffset()
}

// scrollUp scrolls up by n lines.
func (l *LogView) scrollUp(n int) {
	l.scrollOffset -= n
	l.clampScrollOffset()
}

// scrollDown scrolls down by n lines.
func (l *LogView) scrollDown(n int) {
	l.scrollOffset += n
	l.clampScrollOffset()
}

// scrollToBottom scrolls to the bottom.
func (l *LogView) scrollToBottom() {
	l.scrollOffset = len(l.wrappedLines) - l.height
	l.clampScrollOffset()
}

// clampScrollOffset ensures scroll offset is within valid range.
func (l *LogView) clampScrollOffset() {
	maxOffset := len(l.wrappedLines) - l.height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if l.scrollOffset > maxOffset {
		l.scrollOffset = maxOffset
	}
	if l.scrollOffset < 0 {
		l.scrollOffset = 0
	}
}

// isAtBottom returns true if scrolled to bottom.
func (l *LogView) isAtBottom() bool {
	return l.scrollOffset >= len(l.wrappedLines)-l.height
}

// View renders the log view.
func (l *LogView) View() string {
	if l.height <= 0 {
		return ""
	}

	var lines []string

	endLine := l.scrollOffset + l.height
	if endLine > len(l.wrappedLines) {
		endLine = len(l.wrappedLines)
	}

	for i := l.scrollOffset; i < endLine; i++ {
		line := l.renderLine(l.wrappedLines[i], i)
		lines = append(lines, line)
	}

	// Pad with empty lines if needed
	for len(lines) < l.height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// ScrollPercent returns the current scroll position as a percentage.
func (l *LogView) ScrollPercent() float64 {
	if len(l.wrappedLines) <= l.height {
		return 1.0
	}
	maxScroll := len(l.wrappedLines) - l.height
	return float64(l.scrollOffset) / float64(maxScroll)
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
	l.wrappedLines = l.wrappedLines[:0]
	l.selectedLogIndex = -1
	l.selectionAnchor = -1
	l.selectionStart = -1
	l.selectionEnd = -1
	l.scrollOffset = 0
}

// ToggleAutoScroll is kept for compatibility but now just scrolls to bottom.
func (l *LogView) ToggleAutoScroll() {
	l.scrollToBottom()
	l.selectedLogIndex = -1
	l.selectionAnchor = -1
	l.selectionStart = -1
	l.selectionEnd = -1
}

// IsAutoScrolling returns true if no selection (auto-scroll mode).
func (l *LogView) IsAutoScrolling() bool {
	return l.selectedLogIndex == -1
}

// HasSelection returns true if a log is selected.
func (l *LogView) HasSelection() bool {
	return l.selectedLogIndex >= 0
}

// SelectedLogMessage returns the selected log group message.
func (l *LogView) SelectedLogMessage() string {
	if l.selectionStart < 0 || l.selectionEnd < 0 {
		return ""
	}
	var lines []string
	for i := l.selectionStart; i <= l.selectionEnd; i++ {
		if i < len(l.filteredLogs) {
			entry := l.filteredLogs[i]
			line := fmt.Sprintf("[%s] %s", entry.Service, entry.Message)
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// LastCopyCount returns the number of lines from the last copy operation.
func (l *LogView) LastCopyCount() int {
	return l.lastCopyCount
}
