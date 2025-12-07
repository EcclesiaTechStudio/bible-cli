package model

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data.json
var embeddedData []byte

type Bible struct {
	OT Testament `json:"OT"`
	NT Testament `json:"NT"`
}
type Testament map[string]Book
type Book map[string]Chapter
type Chapter map[string]string

// ParseDatabase parses the embedded JSON
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

func LoadDatabase() (*Bible, error) {
	return ParseDatabase(embeddedData)
}
