package cli

import (
	"strings"
	"testing"
)

func TestFlagsAnyOrder(t *testing.T) {
	// FR-033: identical behaviour regardless of flag position.
	in := "5–10 µg\n"
	want := "5-10 ug\n"
	argSets := [][]string{
		{"-", "--strict"},
		{"--strict", "-"},
		{"--charset", "ascii", "-", "--line-break", "lf"},
		{"-", "--format", "text"},
	}
	for _, args := range argSets {
		code, out, _ := runCLI(t, in, args...)
		if code != 0 || out != want {
			t.Errorf("args %v: code %d out %q", args, code, out)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	// One case per parseOptions failure message; the message text is part of
	// the diagnostics contract and is pinned here.
	cases := []struct {
		args    []string
		wantMsg string
	}{
		{[]string{"-", "--nonsense"}, "unknown flag --nonsense"},
		{[]string{"--format"}, "flag --format requires a value"},
		{[]string{"-", "--format", "xml"}, "invalid --format (text|json)"},
		{[]string{"-", "--charset", "big5"}, "invalid --charset (ascii|latin1)"},
		{[]string{"-", "--line-break", "crlf"}, "invalid --line-break (lf|cr)"},
		{[]string{"-", "--output-encoding", "utf16"}, "invalid --output-encoding (utf8|latin1)"},
		{[]string{"a.txt", "b.txt"}, "multiple input arguments"},
		{[]string{}, "missing input (file path or - for stdin)"},
	}
	for _, c := range cases {
		code, out, errs := runCLI(t, "x", c.args...)
		if code != 2 {
			t.Errorf("args %v: exit %d, want 2", c.args, code)
		}
		if out != "" {
			t.Errorf("args %v: stdout must be empty on usage error, got %q", c.args, out)
		}
		if !strings.Contains(errs, "usage:") {
			t.Errorf("args %v: stderr should carry usage text", c.args)
		}
		if !strings.Contains(errs, c.wantMsg) {
			t.Errorf("args %v: stderr missing %q, got %q", c.args, c.wantMsg, errs)
		}
	}
}

// A1 usage-error boundary: a flag-parse failure is text-mode even when
// --format json appears on the command line.
func TestUsageErrorBoundaryWithJSONFlag(t *testing.T) {
	code, out, errs := runCLI(t, "x", "--format", "json", "--nonsense", "-")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if out != "" {
		t.Errorf("no JSON envelope on parse failure, got %q", out)
	}
	if !strings.Contains(errs, "usage:") {
		t.Errorf("expected text usage on stderr, got %q", errs)
	}
}

func TestUsageTextMatchesContract(t *testing.T) {
	_, out, _ := runCLI(t, "", "--help")
	for _, flag := range []string{"-o <file>", "--format", "--charset",
		"--output-encoding", "--line-break", "--hl7-escape", "--strict",
		"--table", "--base64", "--base64-input", "--help", "--version"} {
		if !strings.Contains(out, flag) {
			t.Errorf("usage text missing %q", flag)
		}
	}
	if !strings.Contains(out, "0 success; 1 conversion failure; 2 usage/input error") {
		t.Error("usage text missing exit-code contract")
	}
}
