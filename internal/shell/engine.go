// Package shell implements the interactive shell engine for the Bible CLI.
// It provides a Unix-like navigation system for browsing the Bible with commands
// like cd, ls, cat, grep, and bookmark management. The shell maintains navigation
// state and handles all user commands.
package shell

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/EcclesiaTechStudio/bible-cli/internal/model"
	"github.com/EcclesiaTechStudio/bible-cli/internal/ui"
)

// Engine is the core shell engine that manages navigation state and command execution.
// It maintains the current path within the Bible hierarchy, bookmark state, and
// provides an index for fast book lookups.
type Engine struct {
	DB        *model.Bible          // The complete Bible database
	Path      []string              // Current navigation path (e.g., ["NT", "John", "3"])
	PrevPath  []string              // Previous path for "cd -" functionality
	BookIndex map[string]string     // Fuzzy lookup index mapping abbreviations to full paths
	Bookmarks map[string]string     // User-saved bookmarks mapping names to paths
}

// New creates and initializes a new shell Engine with the provided Bible database.
// It builds the book index for fuzzy matching and loads any saved bookmarks from disk.
//
// Parameters:
//   - db: The Bible database to use for navigation and search
//
// Returns:
//   - *Engine: A fully initialized engine ready to process commands
func New(db *model.Bible) *Engine {
	e := &Engine{
		DB:        db,
		Path:      []string{},
		BookIndex: make(map[string]string),
		Bookmarks: make(map[string]string),
	}
	e.buildIndex()
	e.loadBookmarks()
	return e
}

// GetPathString returns the current path as a formatted string for display in the prompt.
// The path is formatted as a Unix-style path starting with "/".
//
// Examples:
//   - Empty path → "/"
//   - ["OT"] → "/OT"
//   - ["NT", "John", "3"] → "/NT/John/3"
func (e *Engine) GetPathString() string {
	return "/" + strings.Join(e.Path, "/")
}

// --- COMMAND ROUTING ---

// RunCommand processes a user command and executes the appropriate action.
// It handles all shell commands including navigation (cd, ls), reading (cat),
// searching (grep), bookmarks (mark, goto, marks), and utility commands
// (help, version, stats, clear, exit).
//
// The function parses the input into command and arguments, then dispatches
// to the appropriate handler. Empty or whitespace-only input is ignored.
//
// Parameters:
//   - input: The raw user input string to parse and execute
//
// Supported commands:
//   - exit, quit: Exit the application
//   - ls, ll: List contents at current path
//   - cd: Change directory/navigate
//   - cat, read: Read verses or chapters
//   - grep, search: Search for text
//   - mark: Save a bookmark
//   - goto, jump: Navigate to a bookmark
//   - marks: List all bookmarks
//   - manna, random: Display random verse
//   - stats: Show Bible statistics
//   - version, --version, -v: Show version
//   - help: Show help text
//   - clear, cls: Clear screen
func (e *Engine) RunCommand(input string) {
	if input == "" {
		return
	}
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	// Farewell messages for exit
	farewells := []string{
		"God bless! 🙏",
		"Go in peace! ✝️",
		"The Lord be with you! 📖",
		"Grace and peace! 🕊️",
		"Walk in the light! ✨",
	}

	switch cmd {
	case "exit", "quit":
		fmt.Printf("\n%s%s%s\n\n", ui.ColorCyan, farewells[rand.Intn(len(farewells))], ui.ColorReset)
		os.Exit(0)
	case "ls", "ll":
		e.doLS()
	case "cd":
		e.doCD(args)
	case "cat", "read":
		if args == "" {
			e.doCat("")
			return
		}

		// --- MULTI-REF SUPPORT ---
		// 1. Normalize separators
		// Allows: "john 3:16 + rom 8:28" OR "john 3:16 and rom 8:28"
		// We pad with spaces to ensure we don't accidentally split words (though unlikely in Bible books)
		normalized := strings.ReplaceAll(args, " + ", " |BREAK| ")
		normalized = strings.ReplaceAll(normalized, " and ", " |BREAK| ")
		normalized = strings.ReplaceAll(normalized, " AND ", " |BREAK| ") // Case insensitive check

		// 2. Split by our special token
		segments := strings.Split(normalized, "|BREAK|")

		for _, seg := range segments {
			cleanSeg := strings.TrimSpace(seg)
			if cleanSeg == "" {
				continue
			}
			e.handleSmartCat(cleanSeg)
		}
	case "grep", "search":
		if args == "" {
			fmt.Println("Usage: grep <word>")
		} else {
			e.doGrep(args)
		}
	case "mark":
		if args == "" {
			fmt.Println("Usage: mark <name>")
		} else {
			e.saveBookmark(args)
		}
	case "goto", "jump":
		e.goToBookmark(args)
	case "marks":
		e.listBookmarks()
	case "manna", "random":
		shouldSpeak := strings.Contains(args, "-s") || strings.Contains(args, "--speak")
		e.doRandom(shouldSpeak)
	case "stats":
		e.doStats()
	case "version", "--version", "-v":
		fmt.Printf("Bible CLI %sv1.3.0%s\n", ui.ColorGreen, ui.ColorReset)
	case "help":
		e.printHelp()
	case "clear", "cls":
		fmt.Print("\033[H\033[2J")
	default:
		if isNumeric(cmd) {
			e.handleSmartCat(input)
		} else {
			fmt.Printf("Command '%s' not found.\n", cmd)
		}
	}
}

// --- INITIALIZATION ---

func (e *Engine) buildIndex() {
	indexTestament := func(tName string, tMap model.Testament) {
		e.BookIndex[strings.ToLower(tName)] = "/" + tName

		// Use UI helper for sorting
		names := ui.GetSortedKeys(tMap)

		for _, name := range names {
			lower := strings.ToLower(name)
			fullPath := "/" + tName + "/" + name
			cleanKey := strings.ReplaceAll(lower, " ", "")

			for i := 1; i <= len(cleanKey); i++ {
				prefix := cleanKey[:i]
				if _, exists := e.BookIndex[prefix]; !exists {
					e.BookIndex[prefix] = fullPath
				}
			}
			// Manual overrides
			if cleanKey == "matthew" {
				e.BookIndex["mt"] = fullPath
			}
			if cleanKey == "mark" {
				e.BookIndex["mk"] = fullPath
			}
			if cleanKey == "luke" {
				e.BookIndex["lk"] = fullPath
			}
			if cleanKey == "john" {
				e.BookIndex["jn"] = fullPath
			}
			if cleanKey == "philippians" {
				e.BookIndex["php"] = fullPath
			}
		}
	}

	indexTestament("OT", e.DB.OT)
	indexTestament("NT", e.DB.NT)
}

// --- NAVIGATION ---

func (e *Engine) doCD(arg string) {
	if arg == "" || arg == "/" {
		e.Path = []string{}
		return
	}

	if arg == "-" {
		if len(e.PrevPath) == 0 {
			fmt.Println(ui.ColorRed + "No history." + ui.ColorReset)
			return
		}
		e.Path, e.PrevPath = e.PrevPath, e.Path
		return
	}
	if arg == ".." {
		if len(e.Path) > 0 {
			e.Path = e.Path[:len(e.Path)-1]
		}
		return
	}

	e.saveHistory()

	if strings.HasPrefix(arg, "/") {
		e.Path = []string{}
		arg = strings.TrimPrefix(arg, "/")
		for part := range strings.SplitSeq(arg, "/") {
			if part == "" {
				continue
			}
			if !e.tryLocalStep(part) {
				fmt.Printf("%s❌ Path element '%s' not found.%s\n", ui.ColorRed, part, ui.ColorReset)
				return
			}
		}
		return
	}

	if e.tryLocalStep(arg) {
		return
	}
	if e.tryTeleport(arg) {
		return
	}

	fmt.Printf("%s❌ Path '%s' not found.%s\n", ui.ColorRed, arg, ui.ColorReset)
}

func (e *Engine) tryLocalStep(target string) bool {
	switch len(e.Path) {
	case 0:
		return e.enterTestament(target)
	case 1:
		return e.enterBook(target)
	case 2:
		return e.enterChapter(target)
	default:
		return false
	}
}

func (e *Engine) enterTestament(target string) bool {
	clean := strings.ToLower(target)
	if clean == "ot" {
		e.Path = append(e.Path, "OT")
		return true
	}
	if clean == "nt" {
		e.Path = append(e.Path, "NT")
		return true
	}
	return false
}

func (e *Engine) enterBook(target string) bool {
	cleanTarget := strings.ToLower(strings.ReplaceAll(target, " ", ""))

	var tMap model.Testament
	if e.Path[0] == "OT" {
		tMap = e.DB.OT
	} else {
		tMap = e.DB.NT
	}

	for k := range tMap {
		if strings.EqualFold(strings.ReplaceAll(k, " ", ""), cleanTarget) {
			e.Path = append(e.Path, k)
			return true
		}
	}
	return false
}

func (e *Engine) enterChapter(target string) bool {
	book := e.getBook(e.Path[0], e.Path[1])
	if book == nil {
		return false
	}
	if _, ok := book[target]; ok {
		e.Path = append(e.Path, target)
		return true
	}
	return false
}

func (e *Engine) tryTeleport(target string) bool {
	cleanTarget := strings.ToLower(strings.ReplaceAll(target, " ", ""))
	if targetPath, found := e.BookIndex[cleanTarget]; found {
		e.Path = strings.Split(strings.TrimPrefix(targetPath, "/"), "/")
		return true
	}
	return false
}

// --- READING (CAT) ---

// validateVerseRange checks if a verse range is valid.
// It ensures start and end are positive, properly ordered, and not too large.
//
// Validation rules:
//   - Start must be >= 1
//   - End must be >= start
//   - Range size must not exceed 100 verses
//
// Parameters:
//   - start: The starting verse number
//   - end: The ending verse number
//
// Returns:
//   - error: Non-nil if validation fails, describing the specific issue
func validateVerseRange(start, end int) error {
	if start < 1 {
		return fmt.Errorf("verse numbers must be >= 1")
	}
	if end < start {
		return fmt.Errorf("invalid range: end (%d) must be >= start (%d)", end, start)
	}
	if end-start+1 > 100 {
		return fmt.Errorf("range too large (max 100 verses), requested %d verses", end-start+1)
	}
	return nil
}

func (e *Engine) handleSmartCat(args string) {
	if args == "" {
		e.doCat("")
		return
	}

	parts := strings.Fields(args)

	// --- GREEDY BOOK MATCHER ---
	var bestMatchPath string
	var argsAfterMatch []string
	var tokensConsumed int

	currentKey := ""
	for i, part := range parts {
		currentKey += strings.ToLower(part)

		if targetPath, ok := e.BookIndex[currentKey]; ok {
			bestMatchPath = targetPath
			tokensConsumed = i + 1
			if i+1 < len(parts) {
				argsAfterMatch = parts[i+1:]
			} else {
				argsAfterMatch = []string{}
			}
		}
	}

	if bestMatchPath != "" {
		isLocalChapter := false
		if len(e.Path) == 2 {
			book := e.getBook(e.Path[0], e.Path[1])
			if book != nil {
				if _, ok := book[strings.ToLower(parts[0])]; ok {
					isLocalChapter = true
				}
			}
		}

		if tokensConsumed > 1 || !isLocalChapter {
			e.Path = strings.Split(strings.TrimPrefix(bestMatchPath, "/"), "/")
			if len(argsAfterMatch) > 0 {
				newArgs := strings.Join(argsAfterMatch, " ")
				e.doCat(newArgs)
			} else {
				e.doCat("")
			}
			return
		}
	}

	e.doCat(args)
}

func (e *Engine) doCat(arg string) {
	if len(e.Path) < 2 {
		fmt.Printf("%sError: Select a book first.%s\n", ui.ColorRed, ui.ColorReset)
		return
	}

	tName, bName := e.Path[0], e.Path[1]
	book := e.getBook(tName, bName)
	if book == nil {
		return
	}

	if arg == "" && len(e.Path) == 2 {
		e.renderBook(book)
		return
	}
	if len(e.Path) == 3 && arg == "" {
		e.renderChapter(e.Path[2], book[e.Path[2]])
		return
	}

	var chapNum string
	var verseArgs string

	if len(e.Path) == 2 {
		tokens := strings.Fields(strings.ReplaceAll(arg, ":", " "))
		chapNum = tokens[0]
		if len(tokens) > 1 {
			verseArgs = tokens[1]
		}
	} else {
		chapNum = e.Path[2]
		verseArgs = arg
	}

	chapter, ok := book[chapNum]
	if !ok {
		fmt.Printf("%sChapter %s not found.%s\n", ui.ColorRed, chapNum, ui.ColorReset)
		return
	}

	if verseArgs == "" {
		e.renderChapter(chapNum, chapter)
		return
	}

	fmt.Printf("\n%sReading %s %s:%s%s\n", ui.ColorCyan, bName, chapNum, verseArgs, ui.ColorReset)

	segments := strings.Split(verseArgs, ",")
	for _, rawSeg := range segments {
		seg := strings.TrimSpace(rawSeg)
		if seg == "" {
			continue
		}

		if strings.Contains(seg, "-") {
			rangeParts := strings.Split(seg, "-")
			if len(rangeParts) != 2 {
				fmt.Printf("%sInvalid range format: %s (use format like '16-18')%s\n", ui.ColorRed, seg, ui.ColorReset)
				continue
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])

			if err1 != nil || err2 != nil {
				fmt.Printf("%sInvalid range: %s (numbers required)%s\n", ui.ColorRed, seg, ui.ColorReset)
				continue
			}

			if err := validateVerseRange(start, end); err != nil {
				fmt.Printf("%s%v%s\n", ui.ColorRed, err, ui.ColorReset)
				continue
			}

			for i := start; i <= end; i++ {
				vKey := strconv.Itoa(i)
				if text, ok := chapter[vKey]; ok {
					fmt.Printf("%s%3d: %s%v\n", ui.ColorYellow, i, ui.ColorReset, text)
				} else {
					fmt.Printf("%s     (End of chapter)%s\n", ui.ColorGray, ui.ColorReset)
					break
				}
			}
			continue
		}

		if text, ok := chapter[seg]; ok {
			i, _ := strconv.Atoi(seg)
			fmt.Printf("%s%3d: %s%v\n", ui.ColorYellow, i, ui.ColorReset, text)
		} else {
			fmt.Printf("%sVerse %s not found.%s\n", ui.ColorRed, seg, ui.ColorReset)
		}
	}
	fmt.Println()
}

// --- RENDERING ---

func (e *Engine) doLS() {
	if len(e.Path) == 0 {
		fmt.Println(ui.ColorGray + "── Bible Root ──" + ui.ColorReset)
		fmt.Println(ui.ColorBlue + "OT  " + ui.ColorReset + "(Old Testament)")
		fmt.Println(ui.ColorBlue + "NT  " + ui.ColorReset + "(New Testament)")
		return
	}
	if len(e.Path) == 1 {
		tMap := e.DB.OT
		if e.Path[0] == "NT" {
			tMap = e.DB.NT
		}
		e.renderTestament(tMap)
		return
	}
	if len(e.Path) == 2 {
		book := e.getBook(e.Path[0], e.Path[1])
		e.renderBook(book)
		return
	}
	if len(e.Path) == 3 {
		book := e.getBook(e.Path[0], e.Path[1])
		chap := book[e.Path[2]]
		e.renderChapter(e.Path[2], chap)
	}
}

func (e *Engine) renderTestament(t model.Testament) {
	keys := ui.GetSortedKeys(t)
	fmt.Println(ui.ColorGray + "── Books ──" + ui.ColorReset)
	for _, k := range keys {
		fmt.Printf("%sDIR  %s%s\n", ui.ColorBlue, k, ui.ColorReset)
	}
}

func (e *Engine) renderBook(bk model.Book) {
	keys := ui.GetSortedKeys(bk)
	fmt.Println(ui.ColorGray + "── Chapters ──" + ui.ColorReset)
	for _, k := range keys {
		fmt.Printf("%sDIR  %s%s\n", ui.ColorBlue, k, ui.ColorReset)
	}
}

func (e *Engine) renderChapter(cNum string, ch model.Chapter) {
	keys := ui.GetSortedKeys(ch)
	fmt.Println(ui.ColorGray + "── Reading " + cNum + " ──" + ui.ColorReset)
	for _, k := range keys {
		fmt.Printf("%s%3s: %s%v\n", ui.ColorYellow, k, ui.ColorReset, ch[k])
	}
}

// --- SEARCH ---

// doGrep performs a context-aware search for the given query string.
// The search scope depends on the current path:
//   - At root: searches entire Bible
//   - In OT/NT: searches that testament
//   - In a book: searches only that book
//   - In a chapter: searches only that chapter
//
// The search is case-insensitive and highlights all matching occurrences.
//
// Parameters:
//   - query: The text to search for (validated for length and emptiness)
func (e *Engine) doGrep(query string) {
	// Validate query
	query = strings.TrimSpace(query)
	if query == "" {
		fmt.Printf("%sError: Search query cannot be empty%s\n", ui.ColorRed, ui.ColorReset)
		return
	}
	if len(query) > 100 {
		fmt.Printf("%sError: Search query too long (max 100 characters)%s\n", ui.ColorRed, ui.ColorReset)
		return
	}
	if len(query) == 1 {
		fmt.Printf("%s⚠️  Warning: Single character search may return many results%s\n", ui.ColorYellow, ui.ColorReset)
	}

	query = strings.ToLower(strings.ReplaceAll(query, "\"", ""))
	fmt.Printf("%sSearching for '%s'...%s\n", ui.ColorGray, query, ui.ColorReset)
	count := 0

	searchChapter := func(bName, cName string, ch model.Chapter) {
		keys := ui.GetSortedKeys(ch)
		for _, vKey := range keys {
			text := ch[vKey]
			if strings.Contains(strings.ToLower(text), query) {
				count++
				// Highlight ALL occurrences (case-insensitive)
				highlighted := highlightAll(text, query)
				fmt.Printf("%s[%s %s:%s] %s%s\n", ui.ColorCyan, bName, cName, vKey, ui.ColorReset, highlighted)
			}
		}
	}

	// Helper to search a whole testament
	searchTestament := func(t model.Testament) {
		for bName, book := range t {
			for cName, chapter := range book {
				searchChapter(bName, cName, chapter)
			}
		}
	}

	// Context Aware Search
	if len(e.Path) == 0 {
		searchTestament(e.DB.OT)
		searchTestament(e.DB.NT)
	} else if len(e.Path) == 1 {
		tMap := e.DB.OT
		if e.Path[0] == "NT" {
			tMap = e.DB.NT
		}
		searchTestament(tMap)
	} else if len(e.Path) == 2 {
		bk := e.getBook(e.Path[0], e.Path[1])
		for ck, cv := range bk {
			searchChapter(e.Path[1], ck, cv)
		}
	} else if len(e.Path) == 3 {
		bk := e.getBook(e.Path[0], e.Path[1])
		searchChapter(e.Path[1], e.Path[2], bk[e.Path[2]])
	}

	if count == 0 {
		fmt.Println("No matches.")
	} else {
		fmt.Printf("%sFound %d matches.%s\n", ui.ColorGray, count, ui.ColorReset)
	}
}

// --- BOOKMARKS ---

// saveBookmark saves the current path as a bookmark with the given name.
// Bookmarks are persisted to disk in ~/.bible_bookmarks JSON file.
//
// Validation rules:
//   - Name cannot be empty (after trimming whitespace)
//   - Name cannot exceed 50 characters
//   - Name cannot be a reserved command name
//
// Parameters:
//   - name: The name to assign to this bookmark
func (e *Engine) saveBookmark(name string) {
	// Validate bookmark name
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Printf("%sError: Bookmark name cannot be empty%s\n", ui.ColorRed, ui.ColorReset)
		return
	}
	if len(name) > 50 {
		fmt.Printf("%sError: Bookmark name too long (max 50 characters)%s\n", ui.ColorRed, ui.ColorReset)
		return
	}

	// Check for reserved command names
	reservedNames := []string{
		"ls", "ll", "cd", "cat", "read", "grep", "search",
		"mark", "goto", "jump", "marks", "manna", "random",
		"stats", "version", "help", "clear", "cls", "exit", "quit",
	}
	lowerName := strings.ToLower(name)
	for _, reserved := range reservedNames {
		if lowerName == reserved {
			fmt.Printf("%sError: '%s' is a reserved command name, choose a different bookmark name%s\n", ui.ColorRed, name, ui.ColorReset)
			return
		}
	}

	pathStr := "/" + strings.Join(e.Path, "/")
	e.Bookmarks[name] = pathStr
	e.persistBookmarks()
	fmt.Printf("%sMarked '%s' at %s%s\n", ui.ColorGreen, name, pathStr, ui.ColorReset)
}

// goToBookmark navigates to a previously saved bookmark location.
// If the bookmark doesn't exist, displays an error message.
//
// Parameters:
//   - name: The name of the bookmark to navigate to
func (e *Engine) goToBookmark(name string) {
	if target, ok := e.Bookmarks[name]; ok {
		e.saveHistory()
		cleanTarget := strings.TrimPrefix(target, "/")
		if cleanTarget == "" {
			e.Path = []string{}
		} else {
			e.Path = strings.Split(cleanTarget, "/")
		}
	} else {
		fmt.Printf("%sBookmark '%s' not found.%s\n", ui.ColorRed, name, ui.ColorReset)
	}
}

// listBookmarks displays all saved bookmarks with their paths.
// Shows a message if no bookmarks have been saved yet.
func (e *Engine) listBookmarks() {
	fmt.Println(ui.ColorCyan + "══ Saved Bookmarks ══" + ui.ColorReset)
	if len(e.Bookmarks) == 0 {
		fmt.Println("  (No bookmarks yet)")
	}
	for name, path := range e.Bookmarks {
		fmt.Printf("  %s%-10s%s -> %s\n", ui.ColorYellow, name, ui.ColorReset, path)
	}
}

func (e *Engine) persistBookmarks() {
	data, err := json.MarshalIndent(e.Bookmarks, "", "  ")
	if err != nil {
		fmt.Printf("%s⚠️  Warning: Failed to encode bookmarks: %v%s\n", ui.ColorYellow, err, ui.ColorReset)
		return
	}
	if err := os.WriteFile(e.getBookmarkFile(), data, 0644); err != nil {
		fmt.Printf("%s⚠️  Warning: Failed to save bookmarks: %v%s\n", ui.ColorYellow, err, ui.ColorReset)
	}
}

func (e *Engine) getBookmarkFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bible_bookmarks"
	}
	return home + "/.bible_bookmarks"
}

func (e *Engine) loadBookmarks() {
	e.Bookmarks = make(map[string]string)
	data, err := os.ReadFile(e.getBookmarkFile())
	if err == nil {
		if err := json.Unmarshal(data, &e.Bookmarks); err != nil {
			fmt.Printf("%s⚠️  Warning: Bookmark file corrupted, starting fresh: %v%s\n", ui.ColorYellow, err, ui.ColorReset)
			e.Bookmarks = make(map[string]string)
		}
	}
}

// --- UTILS ---

func (e *Engine) getBook(tName, bName string) model.Book {
	var tMap model.Testament
	if tName == "OT" {
		tMap = e.DB.OT
	} else {
		tMap = e.DB.NT
	}
	for k, v := range tMap {
		if strings.EqualFold(k, bName) {
			return v
		}
	}
	return nil
}

func (e *Engine) saveHistory() {
	e.PrevPath = make([]string, len(e.Path))
	copy(e.PrevPath, e.Path)
}

// doRandom displays a random verse from anywhere in the Bible.
// Optionally narrates the verse using text-to-speech if speak is true.
//
// Parameters:
//   - speak: If true, narrates the verse using platform-specific TTS
func (e *Engine) doRandom(speak bool) {
	testaments := []string{"OT", "NT"}
	tKey := testaments[rand.Intn(2)]
	var tMap model.Testament
	if tKey == "OT" {
		tMap = e.DB.OT
	} else {
		tMap = e.DB.NT
	}

	books := ui.GetSortedKeys(tMap)
	bKey := books[rand.Intn(len(books))]
	bk := tMap[bKey]

	chaps := ui.GetSortedKeys(bk)
	cKey := chaps[rand.Intn(len(chaps))]
	ch := bk[cKey]

	vs := ui.GetSortedKeys(ch)
	vKey := vs[rand.Intn(len(vs))]

	verseText := ch[vKey]
	fmt.Printf("\n%s[Random] %s %s:%s%s\n%s%s%s\n\n", ui.ColorCyan, bKey, cKey, vKey, ui.ColorReset, ui.ColorBold, verseText, ui.ColorReset)

	if speak {
		narrate := fmt.Sprintf("%s chapter %s verse %s. %s", bKey, cKey, vKey, verseText)
		e.speak(narrate)
	}
}

func (e *Engine) speak(text string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("say", text)
	case "windows":
		// PowerShell speech synthesis
		psCmd := fmt.Sprintf("Add-Type -AssemblyName System.Speech; (New-Object System.Speech.Synthesis.SpeechSynthesizer).Speak('%s')", strings.ReplaceAll(text, "'", "''"))
		cmd = exec.Command("powershell", "-Command", psCmd)
	case "linux":
		// Try espeak if available
		cmd = exec.Command("espeak", text)
	default:
		return
	}

	if cmd != nil {
		_ = cmd.Start() // Start asynchronously so it doesn't block the shell
	}
}

// doStats displays statistics about the Bible database including
// total number of books, chapters, verses, and version information.
func (e *Engine) doStats() {
	otBooks := len(e.DB.OT)
	ntBooks := len(e.DB.NT)

	totalChapters := 0
	totalVerses := 0

	countStats := func(t model.Testament) {
		for _, book := range t {
			totalChapters += len(book)
			for _, chapter := range book {
				totalVerses += len(chapter)
			}
		}
	}

	countStats(e.DB.OT)
	countStats(e.DB.NT)

	fmt.Println()
	fmt.Println(ui.ColorCyan + "📊 BIBLE STATISTICS" + ui.ColorReset)
	fmt.Printf("  %sBooks:%s       %d (39 OT, 27 NT)\n", ui.ColorBlue, ui.ColorReset, otBooks+ntBooks)
	fmt.Printf("  %sChapters:%s    %d\n", ui.ColorBlue, ui.ColorReset, totalChapters)
	fmt.Printf("  %sVerses:%s      %d\n", ui.ColorBlue, ui.ColorReset, totalVerses)
	fmt.Printf("  %sVersion:%s     KJV (King James Version)\n", ui.ColorBlue, ui.ColorReset)
	fmt.Println()
}

// printHelp displays comprehensive help documentation for all available commands.
// Shows command syntax, descriptions, and usage examples organized by category.
func (e *Engine) printHelp() {
	fmt.Println()
	fmt.Println(ui.ColorCyan + "╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    BIBLE CLI — HELP                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝" + ui.ColorReset)

	fmt.Println(ui.ColorBlue + "\n📂 NAVIGATION" + ui.ColorReset)
	fmt.Printf("  %sls%s                  List books or chapters in current location\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scd <book>%s           Jump to a book %s(fuzzy: 'cd jn' → John)%s\n", ui.ColorGreen, ui.ColorReset, ui.ColorGray, ui.ColorReset)
	fmt.Printf("  %scd <chapter>%s        Enter a chapter %s(e.g. 'cd 3')%s\n", ui.ColorGreen, ui.ColorReset, ui.ColorGray, ui.ColorReset)
	fmt.Printf("  %scd ..%s               Go up one level\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scd /%s                Return to root\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scd -%s                Go to previous location (undo)\n", ui.ColorGreen, ui.ColorReset)

	fmt.Println(ui.ColorBlue + "\n📖 READING" + ui.ColorReset)
	fmt.Printf("  %scat%s                 Read current chapter\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scat <ref>%s           Read verses %s(e.g. 'cat 3:16', 'cat 3:16-18')%s\n", ui.ColorGreen, ui.ColorReset, ui.ColorGray, ui.ColorReset)
	fmt.Printf("  %scat <book> <ref>%s    Quick read without navigating\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("                       %s(e.g. 'cat ps 23', 'cat rom 8:28')%s\n", ui.ColorGray, ui.ColorReset)
	fmt.Printf("  %scat ... + ...%s       Read multiple references\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("                       %s(e.g. 'cat gen 1:1 + jn 1:1')%s\n", ui.ColorGray, ui.ColorReset)

	fmt.Println(ui.ColorBlue + "\n🔍 SEARCH" + ui.ColorReset)
	fmt.Printf("  %sgrep <word>%s         Search in current scope\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("                       %s• At root: searches entire Bible%s\n", ui.ColorGray, ui.ColorReset)
	fmt.Printf("                       %s• In OT/NT: searches that testament%s\n", ui.ColorGray, ui.ColorReset)
	fmt.Printf("                       %s• In a book: searches only that book%s\n", ui.ColorGray, ui.ColorReset)

	fmt.Println(ui.ColorBlue + "\n🔖 BOOKMARKS" + ui.ColorReset)
	fmt.Printf("  %smark <name>%s         Save current location as a bookmark\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sgoto <name>%s         Jump to a saved bookmark\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %smarks%s               List all saved bookmarks\n", ui.ColorGreen, ui.ColorReset)

	fmt.Println(ui.ColorBlue + "\n✨ OTHER" + ui.ColorReset)
	fmt.Printf("  %smanna%s               Display a random verse %s(-s to hear it)%s\n", ui.ColorGreen, ui.ColorReset, ui.ColorGray, ui.ColorReset)
	fmt.Printf("  %sstats%s               Show Bible statistics\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sversion%s             Show version information\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sclear%s               Clear the terminal screen\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %shelp%s                Show this help message\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sexit%s                Quit the application\n", ui.ColorGreen, ui.ColorReset)

	fmt.Println()
	fmt.Println(ui.ColorGray + "─────────────────────────────────────────────────────────────────")
	fmt.Println("  Tip: You can also run commands directly: ./bible cat john 3:16")
	fmt.Println("─────────────────────────────────────────────────────────────────" + ui.ColorReset)
	fmt.Println()
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(string(s[0]))
	return err == nil
}

// highlightAll highlights all case-insensitive occurrences of query in text
func highlightAll(text, query string) string {
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	var result strings.Builder
	lastEnd := 0

	for {
		idx := strings.Index(lowerText[lastEnd:], lowerQuery)
		if idx == -1 {
			result.WriteString(text[lastEnd:])
			break
		}

		// Absolute index in original string
		absIdx := lastEnd + idx

		// Write text before match
		result.WriteString(text[lastEnd:absIdx])

		// Write highlighted match (preserving original case)
		result.WriteString(ui.ColorYellow + ui.ColorBold)
		result.WriteString(text[absIdx : absIdx+len(query)])
		result.WriteString(ui.ColorReset)

		lastEnd = absIdx + len(query)
	}

	return result.String()
}
