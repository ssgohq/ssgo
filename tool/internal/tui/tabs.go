package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab represents a service tab.
type Tab struct {
	Name   string
	State  string
	Color  lipgloss.Color
	Active bool
}

// TabBar renders the service tabs.
type TabBar struct {
	tabs     []Tab
	active   int
	width    int
	allLabel string
}

// NewTabBar creates a new tab bar.
func NewTabBar(services []string) *TabBar {
	tabs := make([]Tab, len(services)+1)

	// First tab is "All"
	tabs[0] = Tab{
		Name:   "All",
		State:  "running",
		Color:  ColorPrimary,
		Active: true,
	}

	// Add service tabs
	for i, name := range services {
		tabs[i+1] = Tab{
			Name:  name,
			State: "idle",
			Color: GetServiceColor(i),
		}
	}

	return &TabBar{
		tabs:     tabs,
		active:   0,
		allLabel: "All",
	}
}

// SetWidth sets the tab bar width.
func (t *TabBar) SetWidth(width int) {
	t.width = width
}

// SetActive sets the active tab index.
func (t *TabBar) SetActive(index int) {
	if index < 0 || index >= len(t.tabs) {
		return
	}

	for i := range t.tabs {
		t.tabs[i].Active = (i == index)
	}
	t.active = index
}

// Active returns the active tab index.
func (t *TabBar) Active() int {
	return t.active
}

// ActiveName returns the name of the active tab.
func (t *TabBar) ActiveName() string {
	if t.active >= 0 && t.active < len(t.tabs) {
		return t.tabs[t.active].Name
	}
	return ""
}

// Next switches to the next tab.
func (t *TabBar) Next() {
	next := (t.active + 1) % len(t.tabs)
	t.SetActive(next)
}

// Prev switches to the previous tab.
func (t *TabBar) Prev() {
	prev := t.active - 1
	if prev < 0 {
		prev = len(t.tabs) - 1
	}
	t.SetActive(prev)
}

// SetState updates the state of a service tab.
func (t *TabBar) SetState(name, state string) {
	for i := range t.tabs {
		if t.tabs[i].Name == name {
			t.tabs[i].State = state
			break
		}
	}
}

// Render renders the tab bar.
func (t *TabBar) Render() string {
	var parts []string

	for i, tab := range t.tabs {
		// Build tab label with state indicator
		var label string
		if tab.Name == "All" {
			label = tab.Name
		} else {
			icon := StateIcon(tab.State)
			iconStyle := StateStyle(tab.State)
			label = iconStyle.Render(icon) + " " + tab.Name
		}

		// Apply tab style
		var style lipgloss.Style
		if tab.Active {
			style = ActiveTabStyle.Foreground(tab.Color)
		} else {
			style = InactiveTabStyle
		}

		parts = append(parts, style.Render(label))

		// Add separator except for last tab
		if i < len(t.tabs)-1 {
			parts = append(parts, TabSeparatorStyle.Render(" │ "))
		}
	}

	// Join and render
	content := strings.Join(parts, "")
	return TabBarStyle.Width(t.width).Render(content)
}

// Count returns the number of tabs.
func (t *TabBar) Count() int {
	return len(t.tabs)
}

// ServiceIndex returns the service index for a tab index.
// Returns -1 for the "All" tab.
func (t *TabBar) ServiceIndex() int {
	if t.active == 0 {
		return -1
	}
	return t.active - 1
}
