// Command allium-check validates Allium specification files (.allium.json)
// against the JSON Schema and semantic analysis rules.
//
// Usage:
//
//	allium-check [flags] file1.allium.json [file2.allium.json ...]
//
// Exit codes:
//
//	0  All files are valid (no errors; warnings may be present unless --strict)
//	1  One or more files have validation errors (or warnings with --strict)
//	2  Input or parse error (missing file, invalid JSON, bad flags)
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/foundry-zero/allium/internal/checker"
	"github.com/foundry-zero/allium/internal/report"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("allium-check", flag.ContinueOnError)

	formatFlag := fs.String("format", "text", "Output format: text or json")
	quiet := fs.Bool("quiet", false, "Suppress output (exit code only)")
	strict := fs.Bool("strict", false, "Treat warnings as errors")
	schemaOnly := fs.Bool("schema-only", false, "Run schema validation only, skip semantic passes")
	rulesFlag := fs.String("rules", "", "Comma-separated rule numbers or range (e.g., 7,8,9 or 7-9)")
	showVersion := fs.Bool("version", false, "Print version and exit")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	if *showVersion {
		fmt.Printf("allium-check %s\n", version)
		return 0
	}

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files specified")
		fs.Usage()
		return 2
	}

	// Validate format flag
	if *formatFlag != "text" && *formatFlag != "json" {
		fmt.Fprintf(os.Stderr, "Error: invalid format %q (use text or json)\n", *formatFlag)
		return 2
	}

	// Parse rule filter
	ruleFilter, err := parseRuleFilter(*rulesFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid --rules value: %v\n", err)
		return 2
	}

	// Create checker
	c, err := checker.NewChecker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	opts := checker.CheckOptions{
		SchemaOnly: *schemaOnly,
		RuleFilter: ruleFilter,
		Strict:     *strict,
	}

	exitCode := 0
	for _, path := range files {
		r := c.Check(path, opts)

		// Determine exit code for this file
		if hasInputError(r) {
			exitCode = max(exitCode, 2)
		} else if r.HasErrors() {
			exitCode = max(exitCode, 1)
		} else if *strict && r.HasWarnings() {
			exitCode = max(exitCode, 1)
		}

		// Output unless quiet
		if !*quiet {
			if err := printReport(r, *formatFlag); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 2
			}
		}
	}

	return exitCode
}

// hasInputError returns true if the report contains an INPUT error.
func hasInputError(r *report.Report) bool {
	for _, e := range r.Errors {
		if e.Rule == "INPUT" {
			return true
		}
	}
	return false
}

// printReport outputs the report in the specified format.
func printReport(r *report.Report, format string) error {
	switch format {
	case "json":
		data, err := report.FormatJSON(r)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "text":
		fmt.Print(report.FormatText(r))
	}
	return nil
}

// parseRuleFilter parses a comma-separated list of rule numbers or ranges.
// Examples: "7,8,9", "7-9", "1,3,7-9,22"
func parseRuleFilter(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}

	var rules []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", bounds[0])
			}
			hi, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", bounds[1])
			}
			if lo > hi {
				return nil, fmt.Errorf("invalid range %d-%d", lo, hi)
			}
			for i := lo; i <= hi; i++ {
				rules = append(rules, i)
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid rule number %q", part)
			}
			rules = append(rules, n)
		}
	}
	return rules, nil
}
