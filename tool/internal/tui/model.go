package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ServiceManager is an interface for managing services.
// This decouples the TUI from the runner package.
type ServiceManager interface {
	ServiceNames() []string
	ServiceStates() map[string]string
	RestartService(ctx context.Context, name string, rebuild bool) error
}

// Model is the main Bubble Tea model for the TUI.
type Model struct {
	// Components
	tabBar    *TabBar
	logView   *LogView
	searchBar *SearchBar

	// Service manager reference
	manager ServiceManager

	// Dimensions
	width  int
	height int

	// State
	ready    bool
	quitting bool
}

// NewModel creates a new TUI model.
func NewModel(manager ServiceManager) *Model {
	services := manager.ServiceNames()

	return &Model{
		tabBar:    NewTabBar(services),
		logView:   NewLogView(services),
		searchBar: NewSearchBar(),
		manager:   manager,
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("ss run"),
		m.tick(),
	)
}

// tick returns a command that sends a tick message periodically.
func (m *Model) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys first
		if !m.searchBar.IsFocused() {
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit

			case "/":
				cmds = append(cmds, m.searchBar.ClearAndFocus())
				return m, tea.Batch(cmds...)

			case "tab":
				m.tabBar.Next()
				m.updateServiceFilter()
				return m, nil

			case "shift+tab":
				m.tabBar.Prev()
				m.updateServiceFilter()
				return m, nil

			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				idx := int(msg.String()[0] - '1')
				if idx < m.tabBar.Count() {
					m.tabBar.SetActive(idx)
					m.updateServiceFilter()
				}
				return m, nil

			case "a":
				m.logView.ToggleAutoScroll()
				return m, nil

			case "c":
				m.logView.Clear()
				return m, nil

			case "r":
				// Restart current service
				if name := m.tabBar.ActiveName(); name != "" && name != "All" {
					go func() {
						_ = m.manager.RestartService(context.Background(), name, true)
					}()
				}
				return m, nil
			}
		}

		// Handle search bar input
		if m.searchBar.IsFocused() {
			cmd, handled := m.searchBar.Update(msg)
			if handled {
				m.logView.SetSearchTerm(m.searchBar.Value())
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Update search in real-time
			m.logView.SetSearchTerm(m.searchBar.Value())
			return m, tea.Batch(cmds...)
		}

		// Forward to log view
		cmd := m.logView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.ready = true

	case TickMsg:
		// Update service states
		for name, state := range m.manager.ServiceStates() {
			m.tabBar.SetState(name, string(state))
		}
		cmds = append(cmds, m.tick())

	case ProcessEventMsg:
		// Handle process events
		m.tabBar.SetState(msg.Service, string(msg.State))

		// Add log entry for state changes
		level := LogLevelInfo
		if msg.Error != nil {
			level = LogLevelError
		} else if msg.State == StateRunning {
			level = LogLevelSuccess
		}

		m.logView.AddLog(LogEntry{
			Time:    time.Now(),
			Service: msg.Service,
			Level:   level,
			Message: msg.Message,
		})

	case LogMsg:
		// Handle log messages from services
		m.logView.AddLog(LogEntry{
			Time:    msg.Time,
			Service: msg.Service,
			Level:   LogLevelInfo,
			Message: msg.Message,
		})

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

// updateLayout updates component sizes based on window size.
func (m *Model) updateLayout() {
	// Tab bar takes 1 line
	tabHeight := 1

	// Dividers take 2 lines (one after tabs, one before status)
	dividerHeight := 2

	// Search/status bar takes 1 line
	statusHeight := 1

	// Log view gets the rest
	logHeight := m.height - tabHeight - dividerHeight - statusHeight

	m.tabBar.SetWidth(m.width)
	m.logView.SetSize(m.width, logHeight)
	m.searchBar.SetWidth(m.width)
}

// updateServiceFilter updates the log view filter based on active tab.
func (m *Model) updateServiceFilter() {
	m.logView.SetServiceFilter(m.tabBar.ActiveName())
}

// View renders the TUI.
func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.quitting {
		return "\n  Shutting down...\n"
	}

	divider := DividerStyle.Render(strings.Repeat("─", m.width))

	var parts []string

	// Tabs
	parts = append(parts, m.tabBar.Render())

	// Divider after tabs
	parts = append(parts, divider)

	// Log view
	parts = append(parts, m.logView.View())

	// Divider before status
	parts = append(parts, divider)

	// Search bar or status
	if m.searchBar.IsFocused() || m.searchBar.Value() != "" {
		parts = append(parts, m.searchBar.View())
	} else {
		parts = append(parts, m.renderStatusBar())
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderStatusBar renders the bottom status bar.
func (m *Model) renderStatusBar() string {
	// Build status items
	items := []string{
		StatusKeyStyle.Render("Tab") + StatusDescStyle.Render(": switch"),
		StatusKeyStyle.Render("/") + StatusDescStyle.Render(": search"),
		StatusKeyStyle.Render("r") + StatusDescStyle.Render(": restart"),
		StatusKeyStyle.Render("a") + StatusDescStyle.Render(": auto-scroll"),
		StatusKeyStyle.Render("c") + StatusDescStyle.Render(": clear"),
		StatusKeyStyle.Render("q") + StatusDescStyle.Render(": quit"),
	}

	left := strings.Join(items, "  ")

	// Build right side with scroll info
	scrollInfo := ""
	if !m.logView.IsAutoScrolling() {
		scrollInfo = fmt.Sprintf("%.0f%%", m.logView.ScrollPercent()*100)
	} else {
		scrollInfo = "AUTO"
	}

	logCount := fmt.Sprintf("%d/%d logs", m.logView.LogCount(), m.logView.TotalLogCount())
	right := HelpStyle.Render(logCount + " │ " + scrollInfo)

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacing := m.width - leftWidth - rightWidth - 2
	if spacing < 0 {
		spacing = 0
	}

	content := left + strings.Repeat(" ", spacing) + right
	return StatusBarStyle.Width(m.width).Render(content)
}
