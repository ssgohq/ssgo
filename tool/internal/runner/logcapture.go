package runner

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// ansiRegex matches ANSI escape codes
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// LogEntry represents a captured log line.
type LogEntry struct {
	Time    time.Time
	Service string
	Message string
}

// LogCapture captures log output and sends it to a channel.
type LogCapture struct {
	mu      sync.Mutex
	service string
	buffer  []byte
	logs    chan<- LogEntry
}

// NewLogCapture creates a new log capture writer.
func NewLogCapture(service string, logs chan<- LogEntry) *LogCapture {
	return &LogCapture{
		service: service,
		buffer:  make([]byte, 0, 1024),
		logs:    logs,
	}
}

// Write implements io.Writer.
func (lc *LogCapture) Write(p []byte) (n int, err error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	n = len(p)
	lc.buffer = append(lc.buffer, p...)

	// Process complete lines
	for {
		idx := strings.IndexByte(string(lc.buffer), '\n')
		if idx == -1 {
			break
		}

		line := string(lc.buffer[:idx])
		lc.buffer = lc.buffer[idx+1:]

		// Sanitize and skip empty lines
		line = sanitizeLogLine(line)
		if line == "" {
			continue
		}

		// Send log entry
		if lc.logs != nil {
			lc.logs <- LogEntry{
				Time:    time.Now(),
				Service: lc.service,
				Message: line,
			}
		}
	}

	return n, nil
}

// Flush writes any remaining buffered content.
func (lc *LogCapture) Flush() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.buffer) > 0 {
		line := string(lc.buffer)
		lc.buffer = lc.buffer[:0]

		line = sanitizeLogLine(line)
		if line != "" && lc.logs != nil {
			lc.logs <- LogEntry{
				Time:    time.Now(),
				Service: lc.service,
				Message: line,
			}
		}
	}
}

// sanitizeLogLine cleans up a log line for display.
func sanitizeLogLine(line string) string {
	// Remove ANSI escape codes
	line = ansiRegex.ReplaceAllString(line, "")

	// Remove carriage returns (used for progress bars, etc.)
	line = strings.ReplaceAll(line, "\r", "")

	// Remove other control characters except tab
	var clean strings.Builder
	for _, r := range line {
		if r == '\t' || r >= 32 {
			clean.WriteRune(r)
		}
	}

	return strings.TrimSpace(clean.String())
}
