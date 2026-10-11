package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Quickstart Scenario 2: strict mode fails with exit 1 and produces no
// output payload anywhere (FR-013, FR-030).
func TestStrictExitOneNoStdout(t *testing.T) {
	code, out, errs := runCLI(t, "snow ☃ end\n", "-", "--strict")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if out != "" {
		t.Errorf("stdout must be empty, got %q", out)
	}
	if !strings.Contains(errs, "U+2603") {
		t.Errorf("stderr should name the codepoint, got %q", errs)
	}
	if strings.Contains(errs, "snow") || strings.Contains(errs, "end") {
		t.Errorf("stderr leaks report text: %q", errs)
	}
}

func TestStrictNoOutputFileCreated(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.txt")
	code, _, _ := runCLI(t, "snow ☃\n", "-", "--strict", "-o", dest)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("output file must not exist after strict failure")
	}
	// No temp litter either (atomic write discipline).
	entries, _ := os.ReadDir(filepath.Dir(dest))
	if len(entries) != 0 {
		t.Errorf("unexpected files left in output dir: %v", entries)
	}
}

func TestDefaultModeReplacesAndExitsZero(t *testing.T) {
	code, out, errs := runCLI(t, "snow ☃☃☃ end\n", "-")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if out != "snow ??? end\n" {
		t.Errorf("stdout %q", out)
	}
	if !strings.Contains(errs, "unmappable character U+2603 replaced (x3)") {
		t.Errorf("stderr %q", errs)
	}
}
