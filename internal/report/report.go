// Package report defines types for validation findings (errors and warnings)
// and the report structure used to collect and present validation results.
package report

import "fmt"

// Severity indicates whether a finding is an error or a warning.
type Severity int

const (
	SeverityError   Severity = iota
	SeverityWarning
)

// String returns "error" or "warning".
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// MarshalText implements encoding.TextMarshaler so JSON output uses the string form.
func (s Severity) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON round-tripping.
func (s *Severity) UnmarshalText(text []byte) error {
	switch string(text) {
	case "error":
		*s = SeverityError
	case "warning":
		*s = SeverityWarning
	default:
		return fmt.Errorf("unknown severity %q", text)
	}
	return nil
}

// Location identifies where in a source file a finding occurred.
type Location struct {
	File string `json:"file"`
	Path string `json:"path"` // JSON path like "$.entities[0].fields[1]"
	Line int    `json:"line,omitempty"`
}

// Finding represents a single validation error or warning.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Location Location `json:"location"`
}

// NewFinding creates a Finding with the given parameters.
func NewFinding(rule string, severity Severity, message string, loc Location) Finding {
	return Finding{
		Rule:     rule,
		Severity: severity,
		Message:  message,
		Location: loc,
	}
}

// NewError creates an error-severity Finding.
func NewError(rule string, message string, loc Location) Finding {
	return NewFinding(rule, SeverityError, message, loc)
}

// NewWarning creates a warning-severity Finding.
func NewWarning(rule string, message string, loc Location) Finding {
	return NewFinding(rule, SeverityWarning, message, loc)
}

// Summary holds aggregate counts for a report.
type Summary struct {
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
}

// Report collects all validation findings for a single file.
type Report struct {
	File        string    `json:"file"`
	SchemaValid bool      `json:"schema_valid"`
	Errors      []Finding `json:"errors"`
	Warnings    []Finding `json:"warnings"`
	Summary     Summary   `json:"summary"`
}

// NewReport creates a Report for the given file with empty finding slices.
func NewReport(file string) *Report {
	return &Report{
		File:     file,
		Errors:   []Finding{},
		Warnings: []Finding{},
	}
}

// AddFinding appends a finding to the appropriate slice (Errors or Warnings)
// and updates the summary counts.
func (r *Report) AddFinding(f Finding) {
	switch f.Severity {
	case SeverityError:
		r.Errors = append(r.Errors, f)
		r.Summary.ErrorCount++
	case SeverityWarning:
		r.Warnings = append(r.Warnings, f)
		r.Summary.WarningCount++
	}
}

// HasErrors returns true if the report contains any error-severity findings.
func (r *Report) HasErrors() bool {
	return r.Summary.ErrorCount > 0
}

// HasWarnings returns true if the report contains any warning-severity findings.
func (r *Report) HasWarnings() bool {
	return r.Summary.WarningCount > 0
}
