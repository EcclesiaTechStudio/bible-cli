package ui

import (
	"reflect"
	"testing"
)

func TestGetSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name: "Numeric Sorting (Chapters/Verses)",
			input: map[string]string{
				"10": "Text",
				"1":  "Text",
				"2":  "Text",
			},
			expected: []string{"1", "2", "10"}, // Standard string sort would be 1, 10, 2
		},
		{
			name: "Alphabetical Sorting (Books)",
			input: map[string]string{
				"Genesis": "Book",
				"Exodus":  "Book",
				"Lev":     "Book",
			},
			expected: []string{"Exodus", "Genesis", "Lev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSortedKeys(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetSortedKeys() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVisuals(t *testing.T) {
	// This ensures the function runs without crashing
	PrintHeader()
}
