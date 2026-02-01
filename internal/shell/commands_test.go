package shell

import (
	"strings"
	"testing"

	"github.com/EcclesiaTechStudio/bible-cli/internal/testutils"
)

func TestMarksCommand(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	// Save a couple of bookmarks
	engine.Path = []string{"OT", "Genesis"}
	engine.saveBookmark("genesis1")

	engine.Path = []string{"NT", "John"}
	engine.saveBookmark("john3")

	// Test listing bookmarks
	output := testutils.CaptureOutput(func() {
		engine.listBookmarks()
	})

	if !strings.Contains(output, "genesis1") {
		t.Error("Marks command should list 'genesis1' bookmark")
	}
	if !strings.Contains(output, "john3") {
		t.Error("Marks command should list 'john3' bookmark")
	}
	if !strings.Contains(output, "/OT/Genesis") {
		t.Error("Marks command should show genesis1 path")
	}
	if !strings.Contains(output, "/NT/John") {
		t.Error("Marks command should show john3 path")
	}

	// Test empty bookmarks
	engine2 := New(db)
	engine2.Bookmarks = make(map[string]string) // Clear any loaded bookmarks
	output2 := testutils.CaptureOutput(func() {
		engine2.listBookmarks()
	})

	if !strings.Contains(output2, "(No bookmarks yet)") {
		t.Error("Should show '(No bookmarks yet)' message when empty")
	}
}

func TestMannaCommand(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	// Test manna command without speech
	output := testutils.CaptureOutput(func() {
		engine.doRandom(false)
	})

	// Check that output contains [Random] tag
	if !strings.Contains(output, "[Random]") {
		t.Error("Manna command should show [Random] tag")
	}

	// The output should contain at least one of our mock verses
	hasVerse := strings.Contains(output, "In the beginning") ||
		strings.Contains(output, "For God so loved") ||
		strings.Contains(output, "The book of the generation") ||
		strings.Contains(output, "That which was from the beginning") ||
		strings.Contains(output, "Now these are the names")

	if !hasVerse {
		t.Errorf("Manna command should display a random verse, got:\n%s", output)
	}
}

func TestStatsCommand(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	output := testutils.CaptureOutput(func() {
		engine.doStats()
	})

	// Check for expected stats output sections
	if !strings.Contains(output, "BIBLE STATISTICS") {
		t.Error("Stats should show 'BIBLE STATISTICS' header")
	}
	if !strings.Contains(output, "Books:") {
		t.Error("Stats should show Books count")
	}
	if !strings.Contains(output, "Chapters:") {
		t.Error("Stats should show Chapters count")
	}
	if !strings.Contains(output, "Verses:") {
		t.Error("Stats should show Verses count")
	}
	if !strings.Contains(output, "Version:") {
		t.Error("Stats should show Version")
	}
	if !strings.Contains(output, "KJV") {
		t.Error("Stats should mention KJV version")
	}

	// Mock DB has 2 OT books and 3 NT books = 5 total
	if !strings.Contains(output, "5") {
		t.Error("Stats should show correct book count for mock DB")
	}
}

func TestVersionCommand(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	output := testutils.CaptureOutput(func() {
		engine.RunCommand("version")
	})

	if !strings.Contains(output, "Bible CLI") {
		t.Error("Version command should show 'Bible CLI'")
	}
	if !strings.Contains(output, "v1.3.1") {
		t.Error("Version command should show version number")
	}

	// Test alternate version flags
	output2 := testutils.CaptureOutput(func() {
		engine.RunCommand("--version")
	})
	if !strings.Contains(output2, "Bible CLI") {
		t.Error("--version flag should work")
	}

	output3 := testutils.CaptureOutput(func() {
		engine.RunCommand("-v")
	})
	if !strings.Contains(output3, "Bible CLI") {
		t.Error("-v flag should work")
	}
}

func TestHelpCommand(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	output := testutils.CaptureOutput(func() {
		engine.printHelp()
	})

	// Check for major help sections
	sections := []string{
		"BIBLE CLI — HELP",
		"NAVIGATION",
		"READING",
		"SEARCH",
		"BOOKMARKS",
		"OTHER",
	}

	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("Help should contain '%s' section", section)
		}
	}

	// Check for key commands
	commands := []string{"ls", "cd", "cat", "grep", "mark", "goto", "marks", "manna", "stats", "help", "exit"}
	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Help should mention '%s' command", cmd)
		}
	}
}

func TestClearCommand(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	output := testutils.CaptureOutput(func() {
		engine.RunCommand("clear")
	})

	// Check for ANSI escape codes for clearing screen
	// \033[H\033[2J is the clear screen sequence
	if !strings.Contains(output, "\033[H\033[2J") {
		t.Error("Clear command should output ANSI escape codes")
	}

	// Test cls alias
	output2 := testutils.CaptureOutput(func() {
		engine.RunCommand("cls")
	})

	if !strings.Contains(output2, "\033[H\033[2J") {
		t.Error("cls command should work as alias for clear")
	}
}
