package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SearchBar is a search input component.
type SearchBar struct {
	input   textinput.Model
	focused bool
	width   int
}

// NewSearchBar creates a new search bar.
func NewSearchBar() *SearchBar {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = 100
	ti.Width = 40
	ti.Prompt = ""
	ti.TextStyle = SearchInputStyle
	ti.PlaceholderStyle = HelpStyle

	return &SearchBar{
		input:   ti,
		focused: false,
	}
}

// SetWidth sets the search bar width.
func (s *SearchBar) SetWidth(width int) {
	s.width = width
	s.input.Width = width - 12 // Account for prompt and padding
}

// Focus focuses the search bar and selects all text for easy replacement.
func (s *SearchBar) Focus() tea.Cmd {
	s.focused = true
	// Move cursor to end and select all (effectively allows overwriting)
	s.input.CursorEnd()
	return s.input.Focus()
}

// ClearAndFocus clears the search bar and focuses it.
func (s *SearchBar) ClearAndFocus() tea.Cmd {
	s.Clear()
	s.focused = true
	return s.input.Focus()
}

// Blur unfocuses the search bar.
func (s *SearchBar) Blur() {
	s.focused = false
	s.input.Blur()
}

// IsFocused returns whether the search bar is focused.
func (s *SearchBar) IsFocused() bool {
	return s.focused
}

// Value returns the current search value.
func (s *SearchBar) Value() string {
	return s.input.Value()
}

// SetValue sets the search value.
func (s *SearchBar) SetValue(value string) {
	s.input.SetValue(value)
}

// Clear clears the search input.
func (s *SearchBar) Clear() {
	s.input.SetValue("")
}

// Update handles input events.
func (s *SearchBar) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !s.focused {
		return nil, false
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			s.Clear()
			s.Blur()
			return nil, true
		case "enter":
			s.Blur()
			return nil, true
		}
	}

	s.input, cmd = s.input.Update(msg)
	return cmd, false
}

// View renders the search bar.
func (s *SearchBar) View() string {
	prompt := SearchPromptStyle.Render("Search: ")
	input := s.input.View()

	content := prompt + input
	return SearchBarStyle.Width(s.width).Render(content)
}

// HelpView renders the help text for the status bar.
func (s *SearchBar) HelpView() string {
	if s.focused {
		return HelpStyle.Render("ESC: cancel │ Enter: apply")
	}
	return ""
}
