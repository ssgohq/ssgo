// Package main provides version information for ssgo CLI.
package main

import "runtime"

// Version information - these values are set during build time via ldflags
var (
	// Version is the semantic version of ssgo
	Version = "0.1.0-dev"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the build timestamp
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)
