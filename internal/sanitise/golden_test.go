package sanitise

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"auFTsanitiser/internal/conform"
)

// Golden-corpus harness (T012): every testdata pair must byte-match under
// default options, and every golden output must pass the executable
// conformance check — the same oracle the CLI emission gate runs.
func TestGoldenCorpus(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("..", "..", "testdata", "*.input.txt"))
	if err != nil || len(inputs) == 0 {
		t.Fatalf("no golden fixtures found: %v", err)
	}
	params := conform.Params{Charset: "ascii", LineBreak: "lf", OutputEncoding: "utf8"}
	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".input.txt")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			golden, err := os.ReadFile(strings.TrimSuffix(in, ".input.txt") + ".golden.txt")
			if err != nil {
				t.Fatal(err)
			}
			res, err := Sanitise(string(raw), Options{Charset: "ascii"})
			if err != nil {
				t.Fatalf("Sanitise failed: %v", err)
			}
			if !bytes.Equal([]byte(res.Text), golden) {
				t.Errorf("output does not byte-match golden\n got: %q\nwant: %q", res.Text, golden)
			}
			if err := conform.Check([]byte(res.Text), params); err != nil {
				t.Errorf("golden output violates conformance: %v", err)
			}
			// Idempotence over the whole corpus (SC-007).
			again, err := Sanitise(res.Text, Options{Charset: "ascii"})
			if err != nil {
				t.Fatalf("re-sanitise failed: %v", err)
			}
			if again.Text != res.Text {
				t.Errorf("sanitise is not idempotent for %s", name)
			}
			if again.Substituted != 0 || again.Unmappable != 0 || len(again.Warnings) != 0 {
				t.Errorf("re-sanitise of clean text reported work: %+v", again)
			}
		})
	}
}
