package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
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

// panicReader panics on the first Read with a value that carries a canary, as
// a buggy decoder holding report text would.
type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("ZQXCANARY7 report text") }

// A crash must be a hard exit 1 with a metadata-only message in the selected
// format, not a Go stack trace with the usage-error code (finding 2).
func TestPanicIsInternalErrorExitOne(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"auFTsanitiser", "-"}, panicReader{}, &out, &errb)
	if code != 1 || out.Len() != 0 {
		t.Fatalf("text mode: exit %d, stdout %q", code, out.String())
	}
	if errb.String() != "auFTsanitiser: error: internal error: string\n" {
		t.Errorf("stderr %q", errb.String())
	}
	out.Reset()
	errb.Reset()
	code = Run([]string{"auFTsanitiser", "-", "--format", "json"}, panicReader{}, &out, &errb)
	if code != 1 || errb.Len() != 0 {
		t.Fatalf("json mode: exit %d, stderr %q", code, errb.String())
	}
	if !strings.Contains(out.String(), `"error":"internal error: string"`) || strings.Contains(out.String(), "ZQXCANARY7") {
		t.Errorf("envelope %q", out.String())
	}
}

// A runtime error (index, nil) names only indices and lengths, so its message
// is kept; any other panic value is reduced to its type.
func TestInternalErrorMessage(t *testing.T) {
	var runtimeErr interface{}
	func() {
		defer func() { runtimeErr = recover() }()
		var s []int
		_ = s[3]
	}()
	if got := internalErrorMessage(runtimeErr); !strings.HasPrefix(got, "internal error: runtime error: index out of range") {
		t.Errorf("runtime error: %q", got)
	}
	if got := internalErrorMessage("ZQXCANARY7"); got != "internal error: string" {
		t.Errorf("string panic: %q", got)
	}
	if got := internalErrorMessage(errors.New("ZQXCANARY7")); got != "internal error: *errors.errorString" {
		t.Errorf("error panic: %q", got)
	}
}

// Input is capped so a runaway or hostile stream fails fast with exit 2 instead
// of exhausting memory (finding 5). Both stdin and file inputs are guarded.
func TestInputCap(t *testing.T) {
	big := strings.Repeat("a", maxInput+1)
	code, out, errs := runCLI(t, big, "-")
	if code != 2 || out != "" || errs != "auFTsanitiser: error: input exceeds 64 MiB\n" {
		t.Errorf("stdin: exit %d, stdout %d bytes, stderr %q", code, len(out), errs)
	}
	code, _, _ = runCLI(t, big[:maxInput], "-")
	if code != 0 {
		t.Errorf("exactly the cap should pass, exit %d", code)
	}
	f := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(f, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(f, maxInput+1); err != nil { // sparse: no 64 MiB written
		t.Fatal(err)
	}
	code, _, errs = runCLI(t, "", f, "--format", "json")
	if code != 2 || errs != "" {
		t.Errorf("file: exit %d, stderr %q", code, errs)
	}
}

// eofReader records whether the caller consumed it to EOF.
type eofReader struct {
	r    io.Reader
	done bool
}

func (e *eofReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF {
		e.done = true
	}
	return n, err
}

// The tool drains stdin before failing on the --table so a caller writing a
// large report never sees a broken pipe instead of the table error (finding 3).
func TestTableErrorAfterInputConsumed(t *testing.T) {
	in := &eofReader{r: strings.NewReader("report\n")}
	var out, errb bytes.Buffer
	code := Run([]string{"auFTsanitiser", "-", "--format", "json", "--table", filepath.Join(t.TempDir(), "missing.tsv")}, in, &out, &errb)
	if code != 2 || !strings.Contains(out.String(), `"error":"cannot read table file"`) {
		t.Fatalf("exit %d, stdout %q", code, out.String())
	}
	if !in.done {
		t.Error("stdin was not drained before the table error")
	}
}
