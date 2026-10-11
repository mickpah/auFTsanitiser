package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"auFTsanitiser/internal/conform"
	"auFTsanitiser/internal/decode"
	"auFTsanitiser/internal/sanitise"
)

// Local-only real-report harness (PRD section 9, research.md R10): runs over
// the PHI corpus in /Sample_reports when present, skips when absent. Asserts
// the charset invariant, zero unmappables (every real character maps), and
// metadata-only diagnostics. Never prints corpus content.
func TestRealReportCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "Sample_reports")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("Sample_reports/ not present (local-only corpus)")
	}
	var files []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".hl7") {
			files = append(files, p)
		}
		return nil
	})
	if len(files) == 0 {
		t.Skip("Sample_reports/ contains no .hl7 files")
	}
	params := conform.Params{Charset: "ascii", LineBreak: "lf", OutputEncoding: "utf8"}
	unmappable := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: unreadable", filepath.Base(f))
		}
		dec, err := decode.Decode(raw)
		if err != nil {
			t.Errorf("%s: decode failed: %v", filepath.Base(f), err)
			continue
		}
		res, err := sanitise.Sanitise(dec.Text, sanitise.Options{Charset: "ascii"})
		if err != nil {
			t.Errorf("%s: sanitise failed: %v", filepath.Base(f), err)
			continue
		}
		if cerr := conform.Check([]byte(res.Text), params); cerr != nil {
			t.Errorf("%s: charset invariant violated: %v", filepath.Base(f), cerr)
		}
		unmappable += res.Unmappable
	}
	if unmappable != 0 {
		t.Errorf("real corpus contains %d unmappable character occurrences — extend the table", unmappable)
	}
	t.Logf("real corpus: %d files, all conformant, zero unmappables", len(files))
}
