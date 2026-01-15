package runner

import (
	"testing"
	"time"
)

func TestSanitizeLogLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "with ANSI colors",
			input:    "\x1b[32mGreen text\x1b[0m",
			expected: "Green text",
		},
		{
			name:     "with carriage return",
			input:    "Progress: 50%\rProgress: 100%",
			expected: "Progress: 50%Progress: 100%",
		},
		{
			name:     "with control characters",
			input:    "Text\x00with\x01control\x02chars",
			expected: "Textwithcontrolchars",
		},
		{
			name:     "preserves tabs",
			input:    "Column1\tColumn2\tColumn3",
			expected: "Column1\tColumn2\tColumn3",
		},
		{
			name:     "complex ANSI sequence",
			input:    "\x1b[1;31;40mBold red on black\x1b[0m",
			expected: "Bold red on black",
		},
		{
			name:     "with leading/trailing spaces",
			input:    "  trimmed  ",
			expected: "trimmed",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only ANSI codes",
			input:    "\x1b[32m\x1b[0m",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeLogLine(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeLogLine(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLogCaptureWrite(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("test-service", logs)

	// Write a complete line
	n, err := capture.Write([]byte("First line\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 11 {
		t.Errorf("Write returned %d, expected 11", n)
	}

	// Should receive log entry
	select {
	case entry := <-logs:
		if entry.Service != "test-service" {
			t.Errorf("Service = %q, want %q", entry.Service, "test-service")
		}
		if entry.Message != "First line" {
			t.Errorf("Message = %q, want %q", entry.Message, "First line")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected log entry not received")
	}
}

func TestLogCaptureMultipleLines(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("svc", logs)

	// Write multiple lines at once
	_, _ = capture.Write([]byte("Line 1\nLine 2\nLine 3\n"))

	// Should receive 3 entries
	messages := []string{}
	for i := 0; i < 3; i++ {
		select {
		case entry := <-logs:
			messages = append(messages, entry.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("expected 3 log entries, got %d", len(messages))
		}
	}

	expected := []string{"Line 1", "Line 2", "Line 3"}
	for i, msg := range messages {
		if msg != expected[i] {
			t.Errorf("message[%d] = %q, want %q", i, msg, expected[i])
		}
	}
}

func TestLogCapturePartialLine(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("svc", logs)

	// Write partial line
	_, _ = capture.Write([]byte("Partial"))

	// Should not receive anything yet
	select {
	case entry := <-logs:
		t.Errorf("unexpected log entry: %v", entry)
	case <-time.After(50 * time.Millisecond):
		// Expected - no entry yet
	}

	// Complete the line
	_, _ = capture.Write([]byte(" line\n"))

	// Now should receive
	select {
	case entry := <-logs:
		if entry.Message != "Partial line" {
			t.Errorf("Message = %q, want %q", entry.Message, "Partial line")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected log entry not received")
	}
}

func TestLogCaptureFlush(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("svc", logs)

	// Write partial line without newline
	_, _ = capture.Write([]byte("Incomplete"))

	// Should not receive anything yet
	select {
	case <-logs:
		t.Error("should not receive log before flush")
	case <-time.After(50 * time.Millisecond):
	}

	// Flush should send remaining content
	capture.Flush()

	select {
	case entry := <-logs:
		if entry.Message != "Incomplete" {
			t.Errorf("Message = %q, want %q", entry.Message, "Incomplete")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected log entry after flush")
	}
}

func TestLogCaptureEmptyLines(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("svc", logs)

	// Write lines with empty ones
	_, _ = capture.Write([]byte("Line 1\n\n\nLine 2\n"))

	// Should receive only non-empty lines
	messages := []string{}
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case entry := <-logs:
			messages = append(messages, entry.Message)
		case <-timeout:
			break loop
		}
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d: %v", len(messages), messages)
	}
}

func TestLogCaptureSanitizesOutput(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("svc", logs)

	// Write line with ANSI codes
	_, _ = capture.Write([]byte("\x1b[32m[INFO]\x1b[0m Server started\n"))

	select {
	case entry := <-logs:
		if entry.Message != "[INFO] Server started" {
			t.Errorf("Message = %q, want %q", entry.Message, "[INFO] Server started")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected log entry not received")
	}
}

func TestLogEntryHasTimestamp(t *testing.T) {
	logs := make(chan LogEntry, 10)
	capture := NewLogCapture("svc", logs)

	before := time.Now()
	_, _ = capture.Write([]byte("Test\n"))
	after := time.Now()

	select {
	case entry := <-logs:
		if entry.Time.Before(before) || entry.Time.After(after) {
			t.Errorf("Time should be between %v and %v, got %v", before, after, entry.Time)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected log entry not received")
	}
}
