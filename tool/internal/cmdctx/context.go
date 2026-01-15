// Package cmdctx provides the command execution context for ssgo commands.
package cmdctx

import (
	"fmt"
)

// Context holds the execution context for a command.
type Context struct {
	// Command-line arguments (after flags are parsed)
	Args []string

	// Flags parsed from command-line
	Flags map[string]interface{}

	// Working directory
	WorkingDir string

	// Debug mode
	Debug bool
}

// New creates a new Context with defaults.
func New() *Context {
	return &Context{
		Args:       make([]string, 0),
		Flags:      make(map[string]interface{}),
		WorkingDir: ".",
	}
}

// ParseArgs parses command-line arguments and populates the Flags map.
func (c *Context) ParseArgs() {
	var newArgs []string
	args := c.Args

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check for --flag=value or -f=value format
		if len(arg) > 1 && (arg[0] == '-') {
			var name, value string
			var hasValue bool

			if idx := findChar(arg, '='); idx != -1 {
				// --flag=value or -f=value
				name = trimDashes(arg[:idx])
				value = arg[idx+1:]
				hasValue = true
			} else {
				// --flag or -f (value might be next arg)
				name = trimDashes(arg)

				// Check if next arg is a value (not another flag)
				if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
					value = args[i+1]
					hasValue = true
					i++ // skip next arg
				} else {
					// Flag without value, treat as boolean true
					c.Flags[name] = true
					continue
				}
			}

			if hasValue {
				// Store the flag value
				c.Flags[name] = value

				// Also store short/long alias
				switch name {
				case "a", "api":
					c.Flags["api"] = value
					c.Flags["a"] = value
				case "m", "module":
					c.Flags["module"] = value
					c.Flags["m"] = value
				case "o", "dir":
					c.Flags["dir"] = value
					c.Flags["o"] = value
				case "p", "proto":
					c.Flags["proto"] = value
					c.Flags["p"] = value
				case "s", "service":
					c.Flags["service"] = value
					c.Flags["s"] = value
				case "f", "file":
					c.Flags["file"] = value
					c.Flags["f"] = value
				case "v", "verbose":
					c.Flags["verbose"] = true
					c.Flags["v"] = true
				}
			}
		} else {
			// Not a flag, keep as positional arg
			newArgs = append(newArgs, arg)
		}
	}

	c.Args = newArgs
}

// GetFlag returns a flag value as string.
func (c *Context) GetFlag(name string) string {
	if v, ok := c.Flags[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// GetFlagBool returns a flag value as bool.
func (c *Context) GetFlagBool(name string) bool {
	if v, ok := c.Flags[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// HasFlag returns true if a flag exists in the context.
func (c *Context) HasFlag(name string) bool {
	_, ok := c.Flags[name]
	return ok
}

// GetCompletionToComplete returns the current word being completed (for shell completion).
func (c *Context) GetCompletionToComplete() string {
	if len(c.Args) > 0 {
		return c.Args[len(c.Args)-1]
	}
	return ""
}

// PrintCompletions prints shell completion suggestions (map version).
func (c *Context) PrintCompletions(completions map[string]string) {
	for name, desc := range completions {
		if desc != "" {
			fmt.Printf("%s\t%s\n", name, desc)
		} else {
			fmt.Println(name)
		}
	}
}

// Helper functions
func findChar(s string, char byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == char {
			return i
		}
	}
	return -1
}

func trimDashes(s string) string {
	for len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	return s
}
