// Package ui provides user interface utilities for the Bible CLI application.
// It handles console output formatting, ANSI color codes, and display helpers.
package ui

import (
	"fmt"
	"sort"
	"strconv"
)

// ANSI color codes for terminal output formatting.
// These constants are used throughout the application to provide
// colorized output for better readability and user experience.
const (
	ColorReset  = "\033[0m" // Reset all formatting
	ColorGreen  = "\033[32m" // Used for prompts and success messages
	ColorBlue   = "\033[34m" // Used for directories and paths
	ColorYellow = "\033[33m" // Used for verse numbers and highlights
	ColorCyan   = "\033[36m" // Used for headers and section titles
	ColorRed    = "\033[31m" // Used for error messages
	ColorGray   = "\033[90m" // Used for secondary text and hints
	ColorBold   = "\033[1m"  // Used for emphasis
)

// PrintHeader clears the screen and displays the welcome banner for the Bible CLI.
// This is called once when the interactive shell starts.
func PrintHeader() {
	fmt.Print("\033[H\033[2J")
	fmt.Println(ColorCyan + "╔══════════════════════════════════════╗")
	fmt.Println("║          BIBLE CLI v1.3.1            ║")
	fmt.Println("╚══════════════════════════════════════╝" + ColorReset)
	fmt.Println(ColorGray + "Type " + ColorGreen + "help" + ColorGray + " to see all commands.")
	fmt.Println(ColorGray + "Type " + ColorGreen + "manna" + ColorGray + " for a random verse.")
	fmt.Println()
}

// GetSortedKeys returns a sorted slice of keys from a map.
// It uses numeric sorting for numeric keys (e.g., chapter/verse numbers)
// and alphabetical sorting for non-numeric keys (e.g., book names).
//
// For example:
//   - Chapter numbers: "1", "2", "10" → sorted as [1, 2, 10] not ["1", "10", "2"]
//   - Book names: "Genesis", "Matthew" → sorted alphabetically
func GetSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		n1, err1 := strconv.Atoi(keys[i])
		n2, err2 := strconv.Atoi(keys[j])
		if err1 == nil && err2 == nil {
			return n1 < n2
		}
		return keys[i] < keys[j]
	})
	return keys
}
