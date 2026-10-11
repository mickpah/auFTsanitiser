package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzRun drives the whole CLI: the exit code is always 0, 1 or 2, a panic
// surfaces as the internal-error shape (so the fuzzer reports it), and every
// stderr line stays inside the metadata-only shapes.
func FuzzRun(f *testing.F) {
	inputs, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "*.input.txt"))
	for _, in := range inputs {
		raw, err := os.ReadFile(in)
		if err == nil {
			f.Add(raw, false, false)
		}
	}
	f.Add([]byte("\xff\xfea\x00b"), true, false)
	f.Add([]byte("\x81\x8d plain"), false, true)
	f.Fuzz(func(t *testing.T, data []byte, strict, hl7 bool) {
		args := []string{"auFTsanitiser", "-"}
		if strict {
			args = append(args, "--strict")
		}
		if hl7 {
			args = append(args, "--hl7-escape")
		}
		var out, errb bytes.Buffer
		code := Run(args, bytes.NewReader(data), &out, &errb)
		if code < 0 || code > 2 {
			t.Fatalf("exit %d", code)
		}
		for _, line := range strings.Split(strings.TrimRight(errb.String(), "\n"), "\n") {
			if line == "" {
				continue
			}
			if strings.Contains(line, "internal error") {
				t.Fatalf("panic recovered: %q", line)
			}
			if !allowedStderr.MatchString(line) {
				t.Fatalf("stderr line outside allowed shapes: %q", line)
			}
		}
	})
}
