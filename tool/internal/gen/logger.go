package gen

import (
	"fmt"
	"io"
	"os"
)

// Logger provides logging functionality for generators.
type Logger interface {
	// Printf prints a formatted message.
	Printf(format string, args ...any)

	// Verbosef prints a formatted message only in verbose mode.
	Verbosef(format string, args ...any)

	// Errorf prints a formatted error message.
	Errorf(format string, args ...any)
}

// StdLogger implements Logger using standard output.
type StdLogger struct {
	verbose bool
	out     io.Writer
	errOut  io.Writer
}

// NewStdLogger creates a new StdLogger.
func NewStdLogger(verbose bool) *StdLogger {
	return &StdLogger{
		verbose: verbose,
		out:     os.Stdout,
		errOut:  os.Stderr,
	}
}

// NewStdLoggerWithWriter creates a StdLogger with custom writers.
func NewStdLoggerWithWriter(verbose bool, out, errOut io.Writer) *StdLogger {
	return &StdLogger{
		verbose: verbose,
		out:     out,
		errOut:  errOut,
	}
}

func (l *StdLogger) Printf(format string, args ...any) {
	fmt.Fprintf(l.out, format+"\n", args...)
}

func (l *StdLogger) Verbosef(format string, args ...any) {
	if l.verbose {
		fmt.Fprintf(l.out, format+"\n", args...)
	}
}

func (l *StdLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(l.errOut, format+"\n", args...)
}

// NopLogger is a no-op logger for testing.
type NopLogger struct{}

func (NopLogger) Printf(format string, args ...any)   {}
func (NopLogger) Verbosef(format string, args ...any) {}
func (NopLogger) Errorf(format string, args ...any)   {}
