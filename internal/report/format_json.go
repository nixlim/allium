package report

import "encoding/json"

// FormatJSON returns the report as indented JSON bytes.
func FormatJSON(r *Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
