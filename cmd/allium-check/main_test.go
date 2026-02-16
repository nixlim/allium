package main

import (
	"os"
	"path/filepath"
	"testing"
)

var refExample = filepath.Join("..", "..", "schemas", "v1", "examples", "password-auth.allium.json")

func TestRunValidFile(t *testing.T) {
	code := run([]string{refExample})
	if code != 0 {
		t.Errorf("run(valid file) = %d, want 0", code)
	}
}

func TestRunValidFileSchemaOnly(t *testing.T) {
	code := run([]string{"--schema-only", refExample})
	if code != 0 {
		t.Errorf("run(--schema-only valid) = %d, want 0", code)
	}
}

func TestRunValidFileCoreRulesOnly(t *testing.T) {
	// Run only references/uniqueness/expressions/sumtypes rules — no surfaces.
	// These should all pass on the reference example.
	code := run([]string{"--rules", "1,3,6,10-14,16-19,22,23,26-28,30,31,35", refExample})
	if code != 0 {
		t.Errorf("run(core rules only) = %d, want 0", code)
	}
}

func TestRunNonexistentFile(t *testing.T) {
	code := run([]string{"/nonexistent/file.allium.json"})
	if code != 2 {
		t.Errorf("run(nonexistent) = %d, want 2", code)
	}
}

func TestRunInvalidSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.allium.json")
	if err := os.WriteFile(path, []byte(`{"file": "test.allium"}`), 0644); err != nil {
		t.Fatal(err)
	}
	code := run([]string{path})
	if code != 1 {
		t.Errorf("run(invalid schema) = %d, want 1", code)
	}
}

func TestRunNoArgs(t *testing.T) {
	code := run([]string{})
	if code != 2 {
		t.Errorf("run(no args) = %d, want 2", code)
	}
}

func TestRunVersion(t *testing.T) {
	code := run([]string{"--version"})
	if code != 0 {
		t.Errorf("run(--version) = %d, want 0", code)
	}
}

func TestRunInvalidFormat(t *testing.T) {
	code := run([]string{"--format", "xml", refExample})
	if code != 2 {
		t.Errorf("run(--format xml) = %d, want 2", code)
	}
}

func TestRunInvalidRules(t *testing.T) {
	code := run([]string{"--rules", "abc", refExample})
	if code != 2 {
		t.Errorf("run(--rules abc) = %d, want 2", code)
	}
}

func TestRunQuiet(t *testing.T) {
	// --quiet should still return the correct exit code.
	code := run([]string{"--quiet", "--schema-only", refExample})
	if code != 0 {
		t.Errorf("run(--quiet --schema-only valid) = %d, want 0", code)
	}
}

func TestRunJSONFormat(t *testing.T) {
	code := run([]string{"--format", "json", "--schema-only", refExample})
	if code != 0 {
		t.Errorf("run(--format json --schema-only valid) = %d, want 0", code)
	}
}

func TestRunStrict(t *testing.T) {
	// The reference example is now clean (0 errors, 0 warnings).
	code := run([]string{"--strict", "--schema-only", refExample})
	if code != 0 {
		t.Errorf("run(--strict --schema-only) = %d, want 0", code)
	}

	// Without --schema-only, --strict should still pass since no warnings.
	code = run([]string{"--strict", refExample})
	if code != 0 {
		t.Errorf("run(--strict, all passes) = %d, want 0", code)
	}
}

func TestRunMultipleFiles(t *testing.T) {
	// One valid (schema-only), one nonexistent. Should return exit code 2 (max).
	code := run([]string{"--schema-only", refExample, "/nonexistent.json"})
	if code != 2 {
		t.Errorf("run(valid + nonexistent) = %d, want 2", code)
	}
}

func TestRunMultipleFilesAllValid(t *testing.T) {
	// Same file twice, schema-only.
	code := run([]string{"--schema-only", refExample, refExample})
	if code != 0 {
		t.Errorf("run(valid + valid, schema-only) = %d, want 0", code)
	}
}

func TestParseRuleFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"single", "7", []int{7}, false},
		{"multiple", "1,3,22", []int{1, 3, 22}, false},
		{"range", "7-9", []int{7, 8, 9}, false},
		{"mixed", "1,7-9,22", []int{1, 7, 8, 9, 22}, false},
		{"spaces", " 1 , 3 ", []int{1, 3}, false},
		{"invalid number", "abc", nil, true},
		{"invalid range start", "abc-5", nil, true},
		{"invalid range end", "5-abc", nil, true},
		{"reversed range", "9-7", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRuleFilter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRuleFilter(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalSlice(got, tt.want) {
				t.Errorf("parseRuleFilter(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func equalSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
