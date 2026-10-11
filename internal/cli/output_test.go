package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWriteSuccess(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out.txt")
	code, out, _ := runCLI(t, "µg\n", "-", "-o", dest)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "" {
		t.Errorf("stdout must be empty with -o, got %q", out)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "ug\n" {
		t.Errorf("file %q, %v", data, err)
	}
}

func TestAtomicWriteLeavesNoTempOnSuccess(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")
	runCLI(t, "x\n", "-", "-o", dest)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "out.txt" {
		t.Errorf("directory should contain only out.txt: %v", entries)
	}
}

func TestUnwritableDestinationIsUsageError(t *testing.T) {
	code, _, errs := runCLI(t, "x\n", "-", "-o", "/nonexistent-dir/deep/out.txt")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if errs == "" {
		t.Error("expected diagnostic")
	}
}

func TestNoPartialFileOnConversionFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")
	code, _, _ := runCLI(t, "bad ☃\n", "-", "--strict", "-o", dest)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("no file may exist after failure: %v", entries)
	}
}

func TestLineBreakCR(t *testing.T) {
	code, out, _ := runCLI(t, "a\nb\n", "-", "--line-break", "cr")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "a\rb\r" {
		t.Errorf("got %q, want CR line breaks", out)
	}
	if strings.Contains(out, "\n") {
		t.Error("LF must not appear in cr mode")
	}
}

func TestLatin1OutputEncoding(t *testing.T) {
	// latin1 charset + latin1 encoding: µ emitted as single byte 0xB5.
	code, out, _ := runCLI(t, "µg café\n", "-",
		"--charset", "latin1", "--output-encoding", "latin1")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := string([]byte{0xB5, 'g', ' ', 'c', 'a', 'f', 0xE9, '\n'})
	if out != want {
		t.Errorf("got %x, want %x", out, want)
	}
}

func TestLatin1CharsetUTF8Encoding(t *testing.T) {
	// Default utf8 encoding keeps multi-byte sequences; codepoints must be
	// in the latin1 permitted set (C1 fix from analysis).
	code, out, _ := runCLI(t, "µg – café\n", "-", "--charset", "latin1")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out != "µg - café\n" {
		t.Errorf("got %q", out)
	}
}

// Review C1: latin1 charset with --hl7-escape succeeds end to end.
func TestLatin1CharsetHL7Escape(t *testing.T) {
	code, out, errs := runCLI(t, "café a|b\n", "-", "--charset", "latin1", "--hl7-escape")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if out != `café a\F\b\.br\` {
		t.Errorf("got %q", out)
	}
}

// Review C3: -o creates a world-readable file (0644, not CreateTemp's
// 0600) and keeps the mode of a file it replaces.
func TestAtomicWriteMode(t *testing.T) {
	dir := t.TempDir()
	fresh := filepath.Join(dir, "new.txt")
	runCLI(t, "x\n", "-", "-o", fresh)
	if fi, err := os.Stat(fresh); err != nil {
		t.Fatal(err)
	} else if fi.Mode().Perm() != 0o644 {
		t.Errorf("new file mode %v, want 0644", fi.Mode().Perm())
	}
	existing := filepath.Join(dir, "old.txt")
	if err := os.WriteFile(existing, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	os.Chmod(existing, 0o640) // defeat umask
	runCLI(t, "x\n", "-", "-o", existing)
	if fi, err := os.Stat(existing); err != nil {
		t.Fatal(err)
	} else if fi.Mode().Perm() != 0o640 {
		t.Errorf("replaced file mode %v, want 0640 kept", fi.Mode().Perm())
	}
}
