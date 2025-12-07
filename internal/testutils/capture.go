package testutils

import (
	"bytes"
	"io"
	"os"
)

// CaptureOutput runs a function and returns what it printed to stdout
func CaptureOutput(f func()) string {
	// 1. Keep backup of the real stdout
	old := os.Stdout

	// 2. Create a pipe (reader, writer)
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 3. Run the function
	f()

	// 4. Close writer so we can read
	w.Close()
	os.Stdout = old // Restore real stdout

	// 5. Read the output from the pipe
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
