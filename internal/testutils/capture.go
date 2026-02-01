// Package testutils provides testing utilities for the Bible CLI application.
// It includes helpers for capturing stdout output during tests, which is
// essential for testing CLI applications that print to the console.
package testutils

import (
	"bytes"
	"io"
	"os"
)

// CaptureOutput runs a function and returns what it printed to stdout.
// This is useful for testing CLI output without actually displaying it.
//
// The function creates a pipe to intercept stdout, runs the provided function,
// then restores stdout to its original state and returns the captured output.
//
// Parameters:
//   - f: A function to execute while capturing its stdout output
//
// Returns:
//   - string: All text that was written to stdout during function execution.
//     Returns empty string if pipe creation fails, or partial output if
//     the copy operation fails.
//
// Example usage:
//
//	output := CaptureOutput(func() {
//	    fmt.Println("Hello, World!")
//	})
//	// output will be "Hello, World!\n"
func CaptureOutput(f func()) string {
	// 1. Keep backup of the real stdout
	old := os.Stdout

	// 2. Create a pipe (reader, writer)
	r, w, err := os.Pipe()
	if err != nil {
		// If we can't create a pipe, restore stdout and return empty
		os.Stdout = old
		return ""
	}
	os.Stdout = w

	// 3. Run the function
	f()

	// 4. Close writer so we can read
	w.Close()
	os.Stdout = old // Restore real stdout

	// 5. Read the output from the pipe
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		// Return partial capture if copy fails
		return buf.String()
	}
	return buf.String()
}
