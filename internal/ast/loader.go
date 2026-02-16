package ast

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadSpec reads and parses an Allium specification JSON file into a Spec.
func LoadSpec(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec JSON: %w", err)
	}

	return &spec, nil
}
