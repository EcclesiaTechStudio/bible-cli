package ui

import (
	"fmt"
	"sort"
	"strconv"
)

// ANSI Colors
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorBlue   = "\033[34m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorRed    = "\033[31m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

func PrintHeader() {
	fmt.Print("\033[H\033[2J")
	fmt.Println(ColorCyan + "╔══════════════════════════════════════╗")
	fmt.Println("║            BIBLE CLI v1.0            ║")
	fmt.Println("╚══════════════════════════════════════╝" + ColorReset)
	fmt.Println(ColorGray + "Type " + ColorGreen + "help" + ColorGray + " to see all commands.")
	fmt.Println(ColorGray + "Type " + ColorGreen + "manna" + ColorGray + " for a random verse.")
	fmt.Println()
}

// GetSortedKeys is a generic helper used by UI and Logic
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
