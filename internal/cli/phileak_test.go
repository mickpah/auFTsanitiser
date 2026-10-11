package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every diagnostic line the tool can emit must match one of these shapes —
// codepoints and counts only, never report text (SC-005, NFR-003).
var allowedStderr = regexp.MustCompile(
	`^auFTsanitiser: (warning: (unmappable character U\+[0-9A-F]{4,6} replaced \(x\d+\)|` +
		`removed character U\+[0-9A-F]{4,6} \(x\d+\)|` +
		`tab replaced with space \(x\d+\)|` +
		`input decoded as Windows-1252 \(fallback\))|` +
		`error: .*)$`)

var errorShapes = regexp.MustCompile(
	`^(unmappable character U\+[0-9A-F]{4,6} \(strict mode\)|` +
		`UTF-16 input has an odd byte count|` +
		`invalid base64 input|cannot read (stdin|input file)|` +
		`cannot write output file|table (line \d+: .*|file .*))$`)

// TestNoFixtureTextInDiagnostics runs the whole corpus through the CLI and
// asserts stderr carries only the allowed metadata shapes.
func TestNoFixtureTextInDiagnostics(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "*.input.txt"))
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	for _, in := range inputs {
		raw, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		for _, extra := range [][]string{{"-"}, {"-", "--strict"}, {"-", "--hl7-escape"}} {
			_, _, errs := runCLI(t, string(raw), extra...)
			for _, line := range strings.Split(strings.TrimRight(errs, "\n"), "\n") {
				if line == "" {
					continue
				}
				if !allowedStderr.MatchString(line) {
					t.Errorf("%s %v: stderr line outside allowed shapes: %q",
						filepath.Base(in), extra, line)
				}
			}
		}
	}
}

// TestStrictErrorShape pins the strict-mode error to its metadata-only form.
func TestStrictErrorShape(t *testing.T) {
	_, _, errs := runCLI(t, "text with ☃ inside\n", "-", "--strict")
	line := strings.TrimPrefix(strings.TrimRight(errs, "\n"), "auFTsanitiser: error: ")
	if !errorShapes.MatchString(line) {
		t.Errorf("strict error %q does not match the allowed shapes", line)
	}
}
