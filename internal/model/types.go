// Package model provides the data model and loading functionality for the Bible database.
// It defines the hierarchical structure: Bible → Testament → Book → Chapter → Verses
// and handles loading the embedded KJV Bible JSON data.
package model

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data.json
var embeddedData []byte

// Bible represents the complete Bible database with Old and New Testaments.
type Bible struct {
	OT Testament `json:"OT"` // Old Testament (39 books in KJV)
	NT Testament `json:"NT"` // New Testament (27 books in KJV)
}

// Testament is a map of book names to Book structures.
// Example keys: "Genesis", "Matthew", "1 John"
type Testament map[string]Book

// Book is a map of chapter numbers to Chapter structures.
// Keys are string representations of chapter numbers: "1", "2", "3", etc.
type Book map[string]Chapter

// Chapter is a map of verse numbers to verse text.
// Keys are string representations of verse numbers: "1", "2", "3", etc.
// Values are the complete text of each verse.
type Chapter map[string]string

// ParseDatabase parses Bible data from JSON format into a Bible structure.
// It validates that the data is not empty and returns an error if JSON parsing fails.
//
// Parameters:
//   - data: Raw JSON bytes containing the Bible database
//
// Returns:
//   - *Bible: Parsed Bible structure
//   - error: Non-nil if data is empty or JSON is malformed
func ParseDatabase(data []byte) (*Bible, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	var db Bible
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("JSON structure mismatch: %v", err)
	}
	return &db, nil
}

// LoadDatabase loads the embedded Bible database from the compiled binary.
// This is the primary entry point for loading Bible data in the application.
//
// Returns:
//   - *Bible: The loaded and parsed Bible structure
//   - error: Non-nil if loading or parsing fails
func LoadDatabase() (*Bible, error) {
	return ParseDatabase(embeddedData)
}

// ValidateDatabase checks the structural integrity of a Bible database.
// It verifies that the database conforms to KJV standards:
//   - Old Testament contains exactly 39 books
//   - New Testament contains exactly 27 books
//   - No books are empty (all books have at least one chapter)
//
// This validation helps catch data corruption or incomplete database files.
//
// Parameters:
//   - db: The Bible structure to validate
//
// Returns:
//   - error: Non-nil if validation fails, describing the specific issue
func ValidateDatabase(db *Bible) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	// Validate OT has 39 books (KJV standard)
	otCount := len(db.OT)
	if otCount != 39 {
		return fmt.Errorf("Old Testament should have 39 books, found %d", otCount)
	}

	// Validate NT has 27 books (KJV standard)
	ntCount := len(db.NT)
	if ntCount != 27 {
		return fmt.Errorf("New Testament should have 27 books, found %d", ntCount)
	}

	// Validate no empty books in OT
	for bookName, book := range db.OT {
		if len(book) == 0 {
			return fmt.Errorf("OT book '%s' has no chapters", bookName)
		}
	}

	// Validate no empty books in NT
	for bookName, book := range db.NT {
		if len(book) == 0 {
			return fmt.Errorf("NT book '%s' has no chapters", bookName)
		}
	}

	return nil
}
