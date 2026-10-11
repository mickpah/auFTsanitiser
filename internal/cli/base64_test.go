package cli

import (
	"encoding/base64"
	"strings"
	"testing"
)

// FR-004: --base64-input decodes before everything else; invalid base64 is
// a usage error (exit 2).
func TestBase64InputRoundTrip(t *testing.T) {
	plain := "37.5°C report\n"
	wrapped := base64.StdEncoding.EncodeToString([]byte(plain))
	code, out, _ := runCLI(t, wrapped, "-", "--base64-input")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "37.5 deg C report\n" {
		t.Errorf("got %q", out)
	}
}

func TestInvalidBase64InputIsUsageError(t *testing.T) {
	code, out, errs := runCLI(t, "!!!not-base64!!!", "-", "--base64-input")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if out != "" {
		t.Errorf("stdout must be empty, got %q", out)
	}
	if errs == "" {
		t.Error("expected diagnostic on stderr")
	}
}

// US3 CLI parity: UTF-16LE input through the full pipeline equals UTF-8.
func TestCLIDecodesUTF16(t *testing.T) {
	le := []byte{0xFF, 0xFE} // BOM
	for _, r := range "5–10 µg\n" {
		u := uint16(r)
		le = append(le, byte(u), byte(u>>8))
	}
	code, out, _ := runCLI(t, string(le), "-")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "5-10 ug\n" {
		t.Errorf("got %q", out)
	}
}

// Windows-1252 bytes through the CLI engage the fallback with a warning.
func TestCLIDecodesCP1252WithWarning(t *testing.T) {
	code, out, errs := runCLI(t, "caf\xe9 \x96 end\n", "-")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "cafe - end\n" {
		t.Errorf("got %q", out)
	}
	if !strings.Contains(errs, "Windows-1252") {
		t.Errorf("expected fallback warning, got %q", errs)
	}
}
