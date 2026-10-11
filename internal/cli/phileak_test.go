package cli

import (
	"bytes"
	"io"
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
		`error: ` + errorAlternatives + `)$`)

// errorAlternatives lists every post-parse error message shape. Table messages
// name a line number and a codepoint; the internal-error shape carries a Go
// runtime message (indices and lengths) or a panic value's type, never input.
const errorAlternatives = `(unmappable character U\+[0-9A-F]{4,6} \(strict mode\)|` +
	`UTF-16 input has an odd byte count|` +
	`invalid base64 input|cannot read (stdin|input file)|input exceeds 64 MiB|` +
	`cannot write output file|table (line \d+: [^\n]*|file is not valid UTF-8)|cannot read table file|` +
	`internal conformance failure: [^\n]*|internal error: [^\n]*)`

var errorShapes = regexp.MustCompile(`^` + errorAlternatives + `$`)

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

// TestCanaryNeverLeaks plants a marker in the input, the table and the file
// path of every failure path and asserts no diagnostic, in either format,
// carries it. (The unknown-flag message does echo the flag by design: flags
// come from the caller's script, not from the report.)
func TestCanaryNeverLeaks(t *testing.T) {
	const canary = "ZQXCANARY7"
	dir := t.TempDir()
	badTable := filepath.Join(dir, canary+".tsv")
	if err := os.WriteFile(badTable, []byte("U+00E9\t"+canary+"|\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		stdin io.Reader
		args  []string
		code  int
	}{
		{"strict unmappable", strings.NewReader(canary + " \u2603"), []string{"-", "--strict"}, 1},
		{"odd-byte utf16", strings.NewReader("\xff\xfe" + canary + "\x00"), []string{"-"}, 1},
		{"table delimiter", strings.NewReader(canary), []string{"-", "--table", badTable}, 2},
		{"table missing", strings.NewReader(canary), []string{"-", "--table", filepath.Join(dir, canary+"-none.tsv")}, 2},
		{"input file missing", strings.NewReader(""), []string{filepath.Join(dir, canary+"-none.txt")}, 2},
		{"invalid base64", strings.NewReader(canary + "!"), []string{"-", "--base64-input"}, 2},
		{"oversize", strings.NewReader(canary + strings.Repeat("a", maxInput)), []string{"-"}, 2},
		{"panic", panicReader{}, []string{"-"}, 1},
		{"unwritable output", strings.NewReader(canary), []string{"-", "-o", filepath.Join(dir, canary+"-none", "out.txt")}, 2},
	}
	for _, c := range cases {
		for _, format := range []string{"text", "json"} {
			var out, errb bytes.Buffer
			args := append([]string{"auFTsanitiser"}, c.args...)
			args = append(args, "--format", format)
			if s, ok := c.stdin.(*strings.Reader); ok {
				s.Seek(0, io.SeekStart)
			}
			code := Run(args, c.stdin, &out, &errb)
			if code != c.code {
				t.Errorf("%s/%s: exit %d, want %d (stderr %q)", c.name, format, code, c.code, errb.String())
			}
			if strings.Contains(out.String(), canary) || strings.Contains(errb.String(), canary) {
				t.Errorf("%s/%s: canary leaked: stdout %q stderr %q", c.name, format, out.String(), errb.String())
			}
			if format == "json" && errb.Len() != 0 {
				t.Errorf("%s/json: stderr must be empty, got %q", c.name, errb.String())
			}
			if format == "json" && !strings.Contains(out.String(), `"error":"`) {
				t.Errorf("%s/json: no error envelope: %q", c.name, out.String())
			}
			if format == "text" {
				for _, line := range strings.Split(strings.TrimRight(errb.String(), "\n"), "\n") {
					if line != "" && !allowedStderr.MatchString(line) {
						t.Errorf("%s/text: stderr line outside allowed shapes: %q", c.name, line)
					}
				}
			}
		}
	}
}
