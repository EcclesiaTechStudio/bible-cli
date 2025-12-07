package shell

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/EcclesiaTechStudio/bible-cli/internal/model"
	"github.com/EcclesiaTechStudio/bible-cli/internal/ui"
)

type Engine struct {
	DB        *model.Bible
	Path      []string
	PrevPath  []string
	BookIndex map[string]string
	Bookmarks map[string]string
}

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

// GetPathString returns the string for the prompt
func (e *Engine) GetPathString() string {
	return "/" + strings.Join(e.Path, "/")
}

// --- COMMAND ROUTING ---

func (e *Engine) RunCommand(input string) {
	if input == "" {
		return
	}
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	switch cmd {
	case "exit", "quit":
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
		e.doRandom()
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
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])

			if err1 != nil || err2 != nil {
				fmt.Printf("%sInvalid range: %s%s\n", ui.ColorRed, seg, ui.ColorReset)
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

func (e *Engine) doGrep(query string) {
	query = strings.ToLower(strings.ReplaceAll(query, "\"", ""))
	fmt.Printf("%sSearching for '%s'...%s\n", ui.ColorGray, query, ui.ColorReset)
	count := 0

	searchChapter := func(bName, cName string, ch model.Chapter) {
		keys := ui.GetSortedKeys(ch)
		for _, vKey := range keys {
			text := ch[vKey]
			if strings.Contains(strings.ToLower(text), query) {
				count++
				lowerText := strings.ToLower(text)
				idx := strings.Index(lowerText, query)
				highlighted := text[:idx] + ui.ColorRed + text[idx:idx+len(query)] + ui.ColorReset + text[idx+len(query):]

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

func (e *Engine) saveBookmark(name string) {
	pathStr := "/" + strings.Join(e.Path, "/")
	e.Bookmarks[name] = pathStr
	e.persistBookmarks()
	fmt.Printf("%sMarked '%s' at %s%s\n", ui.ColorGreen, name, pathStr, ui.ColorReset)
}

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
	data, _ := json.MarshalIndent(e.Bookmarks, "", "  ")
	os.WriteFile(e.getBookmarkFile(), data, 0644)
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
		json.Unmarshal(data, &e.Bookmarks)
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

func (e *Engine) doRandom() {
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

	fmt.Printf("\n%s[Random] %s %s:%s%s\n%s%s%s\n\n", ui.ColorCyan, bKey, cKey, vKey, ui.ColorReset, ui.ColorBold, ch[vKey], ui.ColorReset)
}

func (e *Engine) printHelp() {
	fmt.Println()
	fmt.Println(ui.ColorCyan + "═══ BIBLE SHELL MANUAL v1.0 ═══" + ui.ColorReset)
	fmt.Println(ui.ColorBlue + "\n[ NAVIGATION ]" + ui.ColorReset)
	fmt.Printf("  %scd <book>%s        Teleport (e.g. 'cd rom', 'cd 1 cor')\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scd <chapter>%s     Enter chapter (e.g. 'cd 1')\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scd ..%s            Go back one level\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scd -%s             Jump to previous location (Undo)\n", ui.ColorGreen, ui.ColorReset)
	fmt.Println(ui.ColorBlue + "\n[ READING ]" + ui.ColorReset)
	fmt.Printf("  %scat <ref>%s        Read (e.g. 'cat 3:16', '3:16-18')\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %scat <book...>%s    Quick read (e.g. 'cat john 3:16')\n", ui.ColorGreen, ui.ColorReset)
	fmt.Println(ui.ColorBlue + "\n[ MEMORY ]" + ui.ColorReset)
	fmt.Printf("  %smark <name>%s      Save current spot\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sgoto <name>%s      Jump to saved spot\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %smarks%s            List all bookmarks\n", ui.ColorGreen, ui.ColorReset)
	fmt.Println(ui.ColorBlue + "\n[ TOOLS ]" + ui.ColorReset)
	fmt.Printf("  %sgrep <word>%s      Search contextually\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %smanna%s            Random verse\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sclear%s            Clear screen\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("  %sexit%s             Quit\n", ui.ColorGreen, ui.ColorReset)
	fmt.Println()
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(string(s[0]))
	return err == nil
}
