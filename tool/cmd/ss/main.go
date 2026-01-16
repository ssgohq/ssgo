// Package main provides the entry point for the ssgo CLI tool.
// ssgo is an all-in-one CLI tool for generating Go services, APIs, and database models.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
