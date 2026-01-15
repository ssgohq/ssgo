// Package log provides color-coded multi-service logging for the SDK.
package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// ColorName represents a color name.
type ColorName string

// Available colors for service logging.
const (
	ColorRed     ColorName = "red"
	ColorGreen   ColorName = "green"
	ColorYellow  ColorName = "yellow"
	ColorBlue    ColorName = "blue"
	ColorMagenta ColorName = "magenta"
	ColorCyan    ColorName = "cyan"
	ColorWhite   ColorName = "white"
)

// supportedColors is the list of supported color names for rotation.
var supportedColors = []ColorName{
	ColorCyan, ColorGreen, ColorYellow, ColorBlue, ColorMagenta, ColorRed, ColorWhite,
}

// colorFuncs maps color names to color functions.
var colorFuncs = map[ColorName]func(format string, a ...interface{}) string{
	ColorRed:     color.RedString,
	ColorGreen:   color.GreenString,
	ColorYellow:  color.YellowString,
	ColorBlue:    color.BlueString,
	ColorMagenta: color.MagentaString,
	ColorCyan:    color.CyanString,
	ColorWhite:   color.WhiteString,
}

// boldColorFuncs maps color names to bold color functions.
var boldColorFuncs = map[ColorName]func(format string, a ...interface{}) string{
	ColorRed:     color.New(color.FgRed, color.Bold).SprintfFunc(),
	ColorGreen:   color.New(color.FgGreen, color.Bold).SprintfFunc(),
	ColorYellow:  color.New(color.FgYellow, color.Bold).SprintfFunc(),
	ColorBlue:    color.New(color.FgBlue, color.Bold).SprintfFunc(),
	ColorMagenta: color.New(color.FgMagenta, color.Bold).SprintfFunc(),
	ColorCyan:    color.New(color.FgCyan, color.Bold).SprintfFunc(),
	ColorWhite:   color.New(color.FgWhite, color.Bold).SprintfFunc(),
}

// Logger provides color-coded logging for multiple services.
type Logger struct {
	mu           sync.Mutex
	output       io.Writer
	writers      map[string]*ServiceLogger
	colorIndex   int
	maxNameWidth int
}

// New creates a new Logger instance.
func New() *Logger {
	return &Logger{
		output:       os.Stdout,
		writers:      make(map[string]*ServiceLogger),
		colorIndex:   0,
		maxNameWidth: 0,
	}
}

// SetOutput sets the output writer for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// ForService returns a ServiceLogger for the given service name.
// If colorName is empty, a color will be assigned automatically.
func (l *Logger) ForService(name, colorName string) *ServiceLogger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already exists
	if existing, ok := l.writers[name]; ok {
		return existing
	}

	// Update max name width for alignment
	if len(name) > l.maxNameWidth {
		l.maxNameWidth = len(name)
	}

	// Determine color
	var c ColorName
	if colorName != "" {
		c = ColorName(colorName)
		// Validate color name
		if _, ok := colorFuncs[c]; !ok {
			c = supportedColors[l.colorIndex%len(supportedColors)]
			l.colorIndex++
		}
	} else {
		c = supportedColors[l.colorIndex%len(supportedColors)]
		l.colorIndex++
	}

	sl := &ServiceLogger{
		logger:  l,
		name:    name,
		color:   c,
		lineBuf: make([]byte, 0, 1024),
	}
	l.writers[name] = sl

	// Update prefixes for all writers to align
	l.updatePrefixes()

	return sl
}

// GetService returns the ServiceLogger for a service, or nil if not found.
func (l *Logger) GetService(name string) *ServiceLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.writers[name]
}

// updatePrefixes updates all service prefixes for alignment.
func (l *Logger) updatePrefixes() {
	for _, sl := range l.writers {
		paddedName := fmt.Sprintf("%-*s", l.maxNameWidth, sl.name)
		colorFn := boldColorFuncs[sl.color]
		if colorFn == nil {
			colorFn = boldColorFuncs[ColorWhite]
		}
		sl.prefix = colorFn("[%s] ", paddedName)
	}
}

// Info logs an info message at the global level.
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	info := color.CyanString("ℹ")
	fmt.Fprintf(l.output, "%s %s\n", info, msg)
}

// Success logs a success message at the global level.
func (l *Logger) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	checkmark := color.GreenString("✓")
	fmt.Fprintf(l.output, "%s %s\n", checkmark, msg)
}

// Error logs an error message at the global level.
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	xmark := color.RedString("✗")
	fmt.Fprintf(l.output, "%s %s\n", xmark, msg)
}

// Warn logs a warning message at the global level.
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	exclaim := color.YellowString("!")
	fmt.Fprintf(l.output, "%s %s\n", exclaim, msg)
}

// ServiceLogger is a prefixed writer for a single service.
type ServiceLogger struct {
	logger       *Logger
	name         string
	color        ColorName
	prefix       string
	lineBuf      []byte
	needsModTidy bool
	mu           sync.Mutex
}

// Name returns the service name.
func (sl *ServiceLogger) Name() string {
	return sl.name
}

// Write implements io.Writer for ServiceLogger.
// It buffers input and writes complete lines with prefix.
// Also detects "go mod tidy" messages for auto-recovery.
func (sl *ServiceLogger) Write(p []byte) (n int, err error) {
	sl.logger.mu.Lock()
	defer sl.logger.mu.Unlock()

	n = len(p)
	sl.lineBuf = append(sl.lineBuf, p...)

	// Process complete lines
	for {
		idx := strings.IndexByte(string(sl.lineBuf), '\n')
		if idx == -1 {
			break
		}

		line := string(sl.lineBuf[:idx])
		sl.lineBuf = sl.lineBuf[idx+1:]

		// Detect "go mod tidy" requirement
		if strings.Contains(line, "updates to go.mod needed") ||
			strings.Contains(line, "go mod tidy") {
			sl.mu.Lock()
			sl.needsModTidy = true
			sl.mu.Unlock()
		}

		// Write prefixed line
		fmt.Fprintf(sl.logger.output, "%s%s\n", sl.prefix, line)
	}

	return n, nil
}

// Flush writes any remaining buffered content.
func (sl *ServiceLogger) Flush() {
	sl.logger.mu.Lock()
	defer sl.logger.mu.Unlock()

	if len(sl.lineBuf) > 0 {
		fmt.Fprintf(sl.logger.output, "%s%s\n", sl.prefix, string(sl.lineBuf))
		sl.lineBuf = sl.lineBuf[:0]
	}
}

// NeedsModTidy returns true if "go mod tidy" was detected in the output.
func (sl *ServiceLogger) NeedsModTidy() bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.needsModTidy
}

// ResetModTidy resets the mod tidy detection flag.
func (sl *ServiceLogger) ResetModTidy() {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.needsModTidy = false
}

// Info logs an info message for the service.
func (sl *ServiceLogger) Info(format string, args ...interface{}) {
	sl.logger.mu.Lock()
	defer sl.logger.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(sl.logger.output, "%s%s\n", sl.prefix, msg)
}

// Success logs a success message (green checkmark) for the service.
func (sl *ServiceLogger) Success(format string, args ...interface{}) {
	sl.logger.mu.Lock()
	defer sl.logger.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	checkmark := color.GreenString("✓")
	fmt.Fprintf(sl.logger.output, "%s%s %s\n", sl.prefix, checkmark, msg)
}

// Error logs an error message (red X) for the service.
func (sl *ServiceLogger) Error(format string, args ...interface{}) {
	sl.logger.mu.Lock()
	defer sl.logger.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	xmark := color.RedString("✗")
	fmt.Fprintf(sl.logger.output, "%s%s %s\n", sl.prefix, xmark, msg)
}

// Warn logs a warning message (yellow exclamation) for the service.
func (sl *ServiceLogger) Warn(format string, args ...interface{}) {
	sl.logger.mu.Lock()
	defer sl.logger.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	exclaim := color.YellowString("!")
	fmt.Fprintf(sl.logger.output, "%s%s %s\n", sl.prefix, exclaim, msg)
}

// PrintBanner prints the ssgo run banner.
func PrintBanner() {
	banner := `
  ___  ___   __ _   ___    _ __ _   _ _ __  
 / __|/ __| / _` + "`" + ` | / _ \  | '__| | | | '_ \ 
 \__ \\__ \| (_| || (_) | | |  | |_| | | | |
 |___/|___/ \__, | \___/  |_|   \__,_|_| |_|
            |___/                           
`
	fmt.Println(color.MagentaString(banner))
}

// Global output helper functions

// Info prints an info message to stdout.
func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	info := color.CyanString("ℹ")
	fmt.Fprintf(os.Stdout, "%s %s\n", info, msg)
}

// Success prints a success message to stdout.
func Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	checkmark := color.GreenString("✓")
	fmt.Fprintf(os.Stdout, "%s %s\n", checkmark, msg)
}

// Warning prints a warning message to stdout.
func Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	exclaim := color.YellowString("!")
	fmt.Fprintf(os.Stdout, "%s %s\n", exclaim, msg)
}

// GlobalError prints an error message to stderr (named to avoid conflict with Error method).
func GlobalError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	xmark := color.RedString("✗")
	fmt.Fprintf(os.Stderr, "%s %s\n", xmark, msg)
}

// FilterCompletions filters completions by prefix.
func FilterCompletions(completions []string, prefix string) []string {
	if prefix == "" {
		return completions
	}
	var filtered []string
	for _, c := range completions {
		if len(c) >= len(prefix) && c[:len(prefix)] == prefix {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// PrintCompletions prints a list of completions to stdout.
func PrintCompletions(completions []string) {
	for _, c := range completions {
		fmt.Println(c)
	}
}
