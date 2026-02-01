//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type SourceBook struct {
	Name     string     `json:"name"`
	Abbrev   string     `json:"abbrev"`
	Chapters [][]string `json:"chapters"`
}

func main() {
	url := "https://raw.githubusercontent.com/thiagobodruk/bible/master/json/en_kjv.json"
	fmt.Println("Downloading KJV Bible from GitHub...")

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read response body: %v", err))
	}

	// Remove invisible start bytes
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))

	var sourceData []SourceBook
	fmt.Println("Parsing source JSON...")
	if err := json.Unmarshal(body, &sourceData); err != nil {
		panic(err)
	}

	targetData := make(map[string]map[string]map[string]map[string]string)
	targetData["OT"] = make(map[string]map[string]map[string]string)
	targetData["NT"] = make(map[string]map[string]map[string]string)

	// COMPILE REGEX: Matches footnotes like {Note...: Heb...}
	// Uses [^{}]* instead of .*? to prevent matching across multiple brace groups
	footnoteRegex := regexp.MustCompile(`\{[^{}]*:[^{}]*\}`)

	fmt.Println("Cleaning and reformatting data...")
	for i, book := range sourceData {
		testament := "OT"
		if i >= 39 {
			testament = "NT"
		}

		chaptersMap := make(map[string]map[string]string)

		for cIndex, verses := range book.Chapters {
			chapNum := strconv.Itoa(cIndex + 1)
			verseMap := make(map[string]string)

			for vIndex, rawText := range verses {
				// --- CLEANING LOGIC ---
				// Step A: Remove footnotes (things with colons inside curly braces)
				cleanText := footnoteRegex.ReplaceAllString(rawText, "")

				// Step B: Remove remaining braces but KEEP the text (e.g. {was} -> was)
				cleanText = strings.ReplaceAll(cleanText, "{", "")
				cleanText = strings.ReplaceAll(cleanText, "}", "")

				// Step C: Trim extra spaces caused by removal
				cleanText = strings.TrimSpace(cleanText)

				verseNum := strconv.Itoa(vIndex + 1)
				verseMap[verseNum] = cleanText
			}
			chaptersMap[chapNum] = verseMap
		}

		targetData[testament][book.Name] = chaptersMap
	}

	fmt.Println("Saving to clean data.json...")
	file, err := json.MarshalIndent(targetData, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}
	if err := os.WriteFile("data.json", file, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write data.json: %v", err))
	}

	fmt.Println("Done! Your bible text is now scrubbed and clean.")
}
