// Package tui provides the terminal user interface for ss-plugin-run.
package tui

import (
	"time"
)

// QuitMsg signals that the TUI should quit.
type QuitMsg struct{}

// ProcessState represents the current state of a process.
type ProcessState string

const (
	StateIdle     ProcessState = "idle"
	StateBuilding ProcessState = "building"
	StateStarting ProcessState = "starting"
	StateRunning  ProcessState = "running"
	StateStopping ProcessState = "stopping"
	StateStopped  ProcessState = "stopped"
	StateError    ProcessState = "error"
)

// ProcessEventMsg represents a process state change event.
type ProcessEventMsg struct {
	Service string
	State   ProcessState
	Message string
	Error   error
}

// FileChangedMsg signals a file change event.
type FileChangedMsg struct {
	Service string
	Files   []string
}

// LogMsg represents a log message from a service.
type LogMsg struct {
	Service string
	Message string
	Time    time.Time
}

// LogEntry represents a single log line.
type LogEntry struct {
	Time    time.Time
	Service string
	Level   LogLevel
	Message string
}

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelSuccess
	LogLevelWarning
	LogLevelError
	LogLevelDebug
)

// TickMsg is sent periodically to update the TUI.
type TickMsg time.Time

// WindowSizeMsg is sent when the terminal window size changes.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// FocusSearchMsg signals that the search bar should be focused.
type FocusSearchMsg struct{}

// ClearSearchMsg signals that the search should be cleared.
type ClearSearchMsg struct{}

// SwitchTabMsg signals a tab switch.
type SwitchTabMsg struct {
	Index int
}

// NewLogEntry creates a new log entry.
func NewLogEntry(service, message string, level LogLevel) LogEntry {
	return LogEntry{
		Time:    time.Now(),
		Service: service,
		Level:   level,
		Message: message,
	}
}
