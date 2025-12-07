package model

import (
	"testing"
)

func TestBibleStructure(t *testing.T) {
	jsonData := []byte(`{
		"OT": {
			"Genesis": {
				"1": { "1": "In the beginning" }
			}
		},
		"NT": {}
	}`)

	db, err := ParseDatabase(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse valid JSON: %v", err)
	}

	// Verify Hierarchy: Testament -> Book -> Chapter -> Verse
	book, ok := db.OT["Genesis"]
	if !ok {
		t.Fatal("Failed to map 'Genesis' inside OT")
	}

	chapter, ok := book["1"]
	if !ok {
		t.Fatal("Failed to map Chapter 1 inside Genesis")
	}

	text, ok := chapter["1"]
	if !ok {
		t.Fatal("Failed to map Verse 1 inside Chapter 1")
	}

	if text != "In the beginning" {
		t.Errorf("Verse content mismatch. Got '%s'", text)
	}
}

func TestMalformedJSON(t *testing.T) {
	badData := []byte(`{ "OT": "This should be a map, not a string" }`)

	_, err := ParseDatabase(badData)

	// Expect an error
	if err == nil {
		t.Error("Decoder should have failed on type mismatch, but passed")
	}
}

func TestEmptyData(t *testing.T) {
	// Test the specific check for empty files
	_, err := ParseDatabase([]byte{})
	if err == nil {
		t.Error("Should have failed on empty data")
	}
}
