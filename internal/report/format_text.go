package report

import (
	"fmt"
	"strings"
)

// FormatText returns a human-readable string representation of the report.
// Each finding is on its own line with rule ID, severity, message, and location.
// A summary line is appended at the end.
func FormatText(r *Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "File: %s\n", r.File)

	for _, f := range r.Errors {
		writeFinding(&b, f)
	}
	for _, f := range r.Warnings {
		writeFinding(&b, f)
	}

	fmt.Fprintf(&b, "\n%d errors, %d warnings\n", r.Summary.ErrorCount, r.Summary.WarningCount)
	return b.String()
}

func writeFinding(b *strings.Builder, f Finding) {
	loc := f.Location.Path
	if f.Location.Line > 0 {
		loc = fmt.Sprintf("%s (line %d)", loc, f.Location.Line)
	}
	fmt.Fprintf(b, "  [%s] %s: %s at %s\n", f.Rule, f.Severity, f.Message, loc)
}
