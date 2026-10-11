package table

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTable(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "override.tsv")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOverrideHappyPath(t *testing.T) {
	p := writeTable(t, "# site overrides\n\nU+2191\t(up)\nU+2193\t(down)\n")
	m, err := LoadOverrides(p)
	if err != nil {
		t.Fatal(err)
	}
	if m[0x2191] != "(up)" || m[0x2193] != "(down)" {
		t.Errorf("got %v", m)
	}
}

func TestOverrideEmptyReplacementMeansRemove(t *testing.T) {
	p := writeTable(t, "U+00AD\t\n")
	m, err := LoadOverrides(p)
	if err != nil {
		t.Fatal(err)
	}
	rep, ok := m[0x00AD]
	if !ok || rep != "" {
		t.Errorf("got %v", m)
	}
}

func TestOverrideLastWins(t *testing.T) {
	p := writeTable(t, "U+2191\t(up)\nU+2191\tincreased\n")
	m, err := LoadOverrides(p)
	if err != nil {
		t.Fatal(err)
	}
	if m[0x2191] != "increased" {
		t.Errorf("last entry must win, got %q", m[0x2191])
	}
}

func TestOverrideTrailingSpaceSignificant(t *testing.T) {
	p := writeTable(t, "U+20AC\tEUR \n")
	m, err := LoadOverrides(p)
	if err != nil {
		t.Fatal(err)
	}
	if m[0x20AC] != "EUR " {
		t.Errorf("trailing space lost: %q", m[0x20AC])
	}
}

func TestOverrideValidationRejections(t *testing.T) {
	bad := map[string]string{
		"non-ascii replacement": "U+2191\télevated\n",
		"delimiter tilde":       "U+2248\t~\n",
		"delimiter pipe":        "U+2191\tup|down\n",
		"delimiter backslash":   "U+2191\tup\\down\n",
		"bad codepoint":         "2191\t(up)\n",
		"short hex":             "U+21\tx\n",
		"ascii-range codepoint": "U+0041\tA\n",
		"bad separator":         "U+2191 (up)\n",
		"garbage line":          "not a mapping\n",
	}
	for name, content := range bad {
		p := writeTable(t, content)
		if _, err := LoadOverrides(p); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestOverrideMissingFile(t *testing.T) {
	if _, err := LoadOverrides("/nonexistent/override.tsv"); err == nil {
		t.Error("missing file must error")
	}
}

func TestOverrideErrorsAreMetadataOnly(t *testing.T) {
	p := writeTable(t, "U+2191\tup|PATIENT NAME\n")
	_, err := LoadOverrides(p)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if msg := err.Error(); len(msg) > 0 && (contains(msg, "PATIENT") || contains(msg, "NAME")) {
		t.Errorf("error leaks file content: %q", msg)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Review C2: a table saved with CRLF line endings (Windows editors) loads
// the same as LF.
func TestOverrideCRLFLineEndings(t *testing.T) {
	p := writeTable(t, "# site\r\nU+00DF\tbeta\r\nU+00AD\t\r\n")
	m, err := LoadOverrides(p)
	if err != nil {
		t.Fatal(err)
	}
	if m[0x00DF] != "beta" {
		t.Errorf("got %q, want beta", m[0x00DF])
	}
	if rep, ok := m[0x00AD]; !ok || rep != "" {
		t.Errorf("CRLF removal entry: got %q, %v", rep, ok)
	}
}
