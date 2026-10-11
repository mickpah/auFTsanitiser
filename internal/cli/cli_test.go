package cli

import (
	"bytes"
	"strings"
	"testing"
)

// runCLI executes Run with in-memory streams.
func runCLI(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(append([]string{"auFTsanitiser"}, args...), strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

// Quickstart Scenario 1 / SC-002: the PRD acceptance line converts exactly.
func TestAcceptanceLine(t *testing.T) {
	in := "37.5°C, µg/L, β-hCG, 5–10 mm, “normal”, m², ±2\n"
	want := "37.5 deg C, ug/L, beta-hCG, 5-10 mm, \"normal\", m2, +/-2\n"
	code, out, errs := runCLI(t, in, "-")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if out != want {
		t.Errorf("stdout %q, want %q", out, want)
	}
	if errs != "" {
		t.Errorf("stderr should be empty, got %q", errs)
	}
}

func TestOutputCharsetInvariant(t *testing.T) {
	code, out, _ := runCLI(t, "mixed µ – ” input ↑\n", "-")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for i := 0; i < len(out); i++ {
		if out[i] != '\n' && (out[i] < 0x20 || out[i] > 0x7E) {
			t.Fatalf("out-of-set byte 0x%02X at %d", out[i], i)
		}
	}
}

func TestWarningsGoToStderrOnly(t *testing.T) {
	code, out, errs := runCLI(t, "tab\there\n", "-")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(out, "tab replaced") {
		t.Error("warning leaked to stdout")
	}
	if !strings.Contains(errs, "tab replaced with space (x1)") {
		t.Errorf("expected tab warning on stderr, got %q", errs)
	}
}

func TestHelpAndVersion(t *testing.T) {
	code, out, _ := runCLI(t, "", "--help")
	if code != 0 || !strings.Contains(out, "usage:") {
		t.Errorf("--help: code %d out %q", code, out)
	}
	code, out, _ = runCLI(t, "", "--version")
	if code != 0 || !strings.HasPrefix(out, "auFTsanitiser ") {
		t.Errorf("--version: code %d out %q", code, out)
	}
}

// FR-021: without --hl7-escape reserved delimiters pass through; with it
// (quickstart Scenario 4) the output is delimiter-clean.
func TestHL7EscapeWiring(t *testing.T) {
	in := "A|B^C~D&E\\F\nline2\n"
	code, out, _ := runCLI(t, in, "-")
	if code != 0 || out != in {
		t.Errorf("passthrough: code %d out %q", code, out)
	}
	code, out, _ = runCLI(t, in, "-", "--hl7-escape")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := `A\F\B\S\C\R\D\T\E\E\F\.br\line2\.br\`
	if out != want {
		t.Errorf("escaped: %q, want %q", out, want)
	}
}

func TestMissingInputIsUsageError(t *testing.T) {
	code, _, errs := runCLI(t, "")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if errs == "" {
		t.Error("expected usage diagnostic on stderr")
	}
}

func TestUnreadableFileIsUsageError(t *testing.T) {
	code, _, _ := runCLI(t, "", "/nonexistent/input.txt")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}
