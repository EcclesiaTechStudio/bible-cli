package shell

import (
	"strings"
	"testing"

	"github.com/EcclesiaTechStudio/bible-cli/internal/model"
	"github.com/EcclesiaTechStudio/bible-cli/internal/testutils"
)

func getMockDB() *model.Bible {
	return &model.Bible{
		OT: model.Testament{
			"Genesis": model.Book{
				"1": model.Chapter{"1": "In the beginning..."},
			},
			"Exodus": model.Book{
				"1": model.Chapter{"1": "Now these are the names..."},
			},
		},
		NT: model.Testament{
			"Matthew": model.Book{
				"1": model.Chapter{"1": "The book of the generation..."},
			},
			"John": model.Book{
				"3": model.Chapter{"16": "For God so loved..."},
			},
			"1 John": model.Book{
				"1": model.Chapter{"1": "That which was from the beginning..."},
			},
		},
	}
}

func TestNavigation(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	tests := []struct {
		name         string
		startPath    []string
		command      string
		expectedPath string
	}{
		{"Enter OT", []string{}, "cd ot", "/OT"},
		{"Enter NT", []string{}, "cd nt", "/NT"},
		{"Enter Book (Case Insensitive)", []string{"OT"}, "cd gen", "/OT/Genesis"},
		{"Enter Chapter", []string{"OT", "Genesis"}, "cd 1", "/OT/Genesis/1"},
		{"Go Back", []string{"OT", "Genesis"}, "cd ..", "/OT"},
		{"Go Root", []string{"OT", "Genesis", "1"}, "cd /", ""},
		{"Teleport to John", []string{"OT"}, "cd john", "/NT/John"},
		{"Teleport with Number", []string{}, "cd 1john", "/NT/1 John"},
		{"Read moves context", []string{}, "cat john 3:16", "/NT/John"},
		{"Read maintains context", []string{"NT", "Matthew"}, "cat 1", "/NT/Matthew"},
		{"Invalid Book", []string{"OT"}, "cd fakebook", "/OT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine.Path = tt.startPath
			engine.RunCommand(tt.command)
			resultPath := engine.GetPathString()

			if len(engine.Path) == 0 {
				resultPath = ""
			}

			if resultPath != tt.expectedPath {
				if tt.expectedPath == "" && resultPath == "/" {
					return
				}
				t.Errorf("Command '%s': expected path %v, got %v",
					tt.command, tt.expectedPath, resultPath)
			}
		})
	}
}

func TestGrepSanity(t *testing.T) {
	db := getMockDB()
	engine := New(db)
	engine.RunCommand("grep God")
}

func TestReadingLogic(t *testing.T) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"NT", "John"}

	tests := []struct {
		name string
		args string
	}{
		{"Read Whole Chapter", ""},
		{"Read Single Verse", "3:16"},
		{"Read Range", "3:16-17"},
		{"Read Comma Separated", "3:16,18"},
		{"Read Invalid Verse", "3:99"},
		{"Read Invalid Chapter", "99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine.handleSmartCat(tt.args)
		})
	}
}

func TestBookmarksLogic(t *testing.T) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"OT", "Genesis"}
	engine.RunCommand("mark testgen")
	engine.RunCommand("cd /")
	engine.RunCommand("jump testgen")

	if engine.GetPathString() != "/OT/Genesis" {
		t.Errorf("Bookmark jump failed. Got %s", engine.GetPathString())
	}
}

func TestCat_ContentVerification(t *testing.T) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"NT", "John"}

	tests := []struct {
		name           string
		commandArg     string
		expectedPhrase string
	}{
		{
			name:           "Read 3:16",
			commandArg:     "3:16",
			expectedPhrase: "For God so loved",
		},
		{
			name:           "Read Range 3:16-17",
			commandArg:     "3:16-17",
			expectedPhrase: "For God so loved",
		},
		{
			name:           "Read Invalid Verse",
			commandArg:     "3:99",
			expectedPhrase: "Verse 99 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// CORRECTED: Using testutils.CaptureOutput
			output := testutils.CaptureOutput(func() {
				engine.handleSmartCat(tt.commandArg)
			})

			if !strings.Contains(output, tt.expectedPhrase) {
				t.Errorf("Expected output to contain '%s', but got:\n%s", tt.expectedPhrase, output)
			}
		})
	}
}

func TestLS_DirectoryListing(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	// 1. Test Root Listing
	output := testutils.CaptureOutput(func() {
		engine.doLS()
	})

	if !strings.Contains(output, "OT") || !strings.Contains(output, "NT") {
		t.Error("Root LS should list OT and NT")
	}

	// 2. Test Book Listing
	engine.Path = []string{"OT"}
	// CORRECTED: Using testutils.CaptureOutput (was captureOutput)
	output = testutils.CaptureOutput(func() {
		engine.doLS()
	})

	if !strings.Contains(output, "Genesis") {
		t.Error("OT Listing should contain Genesis")
	}
}

func TestMultiReferenceRead(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	// We use testutils to capture the output of the whole command flow
	output := testutils.CaptureOutput(func() {
		// Try reading from two different books at once using the "+" separator
		// Note: Our mock DB has Genesis 1:1 and John 3:16
		engine.RunCommand("cat Genesis 1:1 + John 3:16")
	})

	// Assertions
	if !strings.Contains(output, "In the beginning") {
		t.Error("Multi-ref failed: Did not find Genesis text")
	}
	if !strings.Contains(output, "For God so loved") {
		t.Error("Multi-ref failed: Did not find John text")
	}
}

func TestGrepScope(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	query := "the"

	// 1. Test Root Search (Should find in OT and NT)
	outputRoot := testutils.CaptureOutput(func() {
		engine.Path = []string{} // Root
		engine.doGrep(query)
	})

	if !strings.Contains(outputRoot, "[Genesis 1:1]") || !strings.Contains(outputRoot, "[Matthew 1:1]") {
		t.Errorf("Root search should find both testaments. Got:\n%s", outputRoot)
	}

	// 2. Test OT Only Scope
	outputOT := testutils.CaptureOutput(func() {
		engine.Path = []string{"OT"} // Enter Old Testament
		engine.doGrep(query)
	})

	if !strings.Contains(outputOT, "[Genesis 1:1]") {
		t.Error("OT search should find Genesis")
	}
	if strings.Contains(outputOT, "[Matthew 1:1]") {
		t.Error("OT search should NOT find Matthew (Leaked context)")
	}

	// 3. Test Book Only Scope
	// We search for "book", which causes doGrep to insert color codes around that word.
	// So we check for the Verse ID and a word that ISN'T highlighted ("generation")
	outputBook := testutils.CaptureOutput(func() {
		engine.Path = []string{"NT", "Matthew"}
		engine.doGrep("book")
	})

	if !strings.Contains(outputBook, "[Matthew 1:1]") {
		t.Error("Book search failed to find the correct verse reference")
	}
	if !strings.Contains(outputBook, "generation") {
		t.Error("Book search failed to find the text content")
	}
}

func TestInvalidRange(t *testing.T) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"NT", "John"}

	output := testutils.CaptureOutput(func() {
		// "16-bad" will fail the Atoi check
		engine.handleSmartCat("3:16-bad")
	})

	if !strings.Contains(output, "Invalid range") {
		t.Errorf("Expected 'Invalid range' error, got:\n%s", output)
	}
}

func TestEdgeCases(t *testing.T) {
	db := getMockDB()
	engine := New(db)

	t.Run("Empty command", func(t *testing.T) {
		// Empty command should do nothing (not crash)
		engine.RunCommand("")
		engine.RunCommand("   ")
	})

	t.Run("Empty bookmark name", func(t *testing.T) {
		engine.Path = []string{"NT", "John"}
		output := testutils.CaptureOutput(func() {
			engine.saveBookmark("")
		})
		if !strings.Contains(output, "cannot be empty") {
			t.Error("Should reject empty bookmark name")
		}
	})

	t.Run("Whitespace-only bookmark name", func(t *testing.T) {
		engine.Path = []string{"NT", "John"}
		output := testutils.CaptureOutput(func() {
			engine.saveBookmark("   ")
		})
		if !strings.Contains(output, "cannot be empty") {
			t.Error("Should reject whitespace-only bookmark name")
		}
	})

	t.Run("Reserved bookmark name", func(t *testing.T) {
		engine.Path = []string{"NT", "John"}
		output := testutils.CaptureOutput(func() {
			engine.saveBookmark("cd")
		})
		if !strings.Contains(output, "reserved command name") {
			t.Error("Should reject reserved command name as bookmark")
		}
	})

	t.Run("Long bookmark name", func(t *testing.T) {
		engine.Path = []string{"NT", "John"}
		longName := strings.Repeat("a", 51)
		output := testutils.CaptureOutput(func() {
			engine.saveBookmark(longName)
		})
		if !strings.Contains(output, "too long") {
			t.Error("Should reject bookmark name > 50 chars")
		}
	})

	t.Run("Empty search query", func(t *testing.T) {
		output := testutils.CaptureOutput(func() {
			engine.doGrep("")
		})
		if !strings.Contains(output, "cannot be empty") {
			t.Error("Should reject empty search query")
		}
	})

	t.Run("Single character search warning", func(t *testing.T) {
		output := testutils.CaptureOutput(func() {
			engine.doGrep("a")
		})
		if !strings.Contains(output, "Warning") {
			t.Error("Should warn about single character search")
		}
	})

	t.Run("Long search query", func(t *testing.T) {
		longQuery := strings.Repeat("search", 20)
		output := testutils.CaptureOutput(func() {
			engine.doGrep(longQuery)
		})
		if !strings.Contains(output, "too long") {
			t.Error("Should reject search query > 100 chars")
		}
	})

	t.Run("Goto non-existent bookmark", func(t *testing.T) {
		output := testutils.CaptureOutput(func() {
			engine.goToBookmark("nonexistent")
		})
		if !strings.Contains(output, "not found") {
			t.Error("Should show error for non-existent bookmark")
		}
	})

	t.Run("Unknown command", func(t *testing.T) {
		output := testutils.CaptureOutput(func() {
			engine.RunCommand("unknowncmd")
		})
		if !strings.Contains(output, "not found") {
			t.Error("Should show error for unknown command")
		}
	})
}

func TestRangeEdgeCases(t *testing.T) {
	db := getMockDB()
	engine := New(db)
	engine.Path = []string{"NT", "John"}

	tests := []struct {
		name          string
		rangeArg      string
		shouldContain string
	}{
		{
			name:          "Valid single verse",
			rangeArg:      "3:16",
			shouldContain: "For God so loved",
		},
		{
			name:          "Reversed range",
			rangeArg:      "3:20-10",
			shouldContain: "invalid range",
		},
		{
			name:          "Negative verse number",
			rangeArg:      "3:-1",
			shouldContain: "Invalid range",
		},
		{
			name:          "Zero verse number",
			rangeArg:      "3:0",
			shouldContain: "Verse 0 not found",
		},
		{
			name:          "Too many range parts",
			rangeArg:      "3:16-17-18",
			shouldContain: "Invalid range format",
		},
		{
			name:          "Non-numeric range",
			rangeArg:      "3:abc",
			shouldContain: "Verse abc not found",
		},
		{
			name:          "Large range",
			rangeArg:      "3:1-200",
			shouldContain: "range too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := testutils.CaptureOutput(func() {
				engine.handleSmartCat(tt.rangeArg)
			})

			if !strings.Contains(output, tt.shouldContain) {
				t.Errorf("Expected output to contain '%s', got:\n%s", tt.shouldContain, output)
			}
		})
	}
}
