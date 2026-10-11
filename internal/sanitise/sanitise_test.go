package sanitise

import (
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, in string) Result {
	t.Helper()
	res, err := Sanitise(in, Options{Charset: "ascii"})
	if err != nil {
		t.Fatalf("Sanitise(%q) failed: %v", in, err)
	}
	return res
}

func TestTableSubstitutionAndCounter(t *testing.T) {
	res := run(t, "µg and β-hCG – ok")
	if res.Text != "ug and beta-hCG - ok" {
		t.Errorf("got %q", res.Text)
	}
	if res.Substituted != 3 {
		t.Errorf("substituted = %d, want 3", res.Substituted)
	}
	if res.Unmappable != 0 || len(res.Warnings) != 0 {
		t.Errorf("unexpected unmappable/warnings: %+v", res)
	}
}

func TestFoldFallback(t *testing.T) {
	res := run(t, "café Zürich straße")
	if res.Text != "cafe Zurich strasse" {
		t.Errorf("got %q", res.Text)
	}
}

func TestNFDCombiningParity(t *testing.T) {
	precomposed := run(t, "café")
	decomposed := run(t, "cafe\u0301")
	if precomposed.Text != decomposed.Text || precomposed.Text != "cafe" {
		t.Errorf("NFC %q vs NFD %q, want both \"cafe\"", precomposed.Text, decomposed.Text)
	}
}

func TestTabReplacedWithWarning(t *testing.T) {
	res := run(t, "a\tb\tc")
	if res.Text != "a b c" {
		t.Errorf("got %q", res.Text)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Category != CategoryTabReplaced || res.Warnings[0].Count != 2 {
		t.Errorf("want one tab warning x2, got %+v", res.Warnings)
	}
}

func TestControlsAndInvisiblesRemovedWithWarning(t *testing.T) {
	res := run(t, "a\x01b\u200Bc\u00ADd") // C0 control, zero-width space, soft hyphen
	if res.Text != "abcd" {
		t.Errorf("got %q", res.Text)
	}
	if len(res.Warnings) != 3 {
		t.Fatalf("want 3 removal warnings (distinct codepoints), got %+v", res.Warnings)
	}
	for _, w := range res.Warnings {
		if w.Category != CategoryRemoved {
			t.Errorf("warning %+v should be CategoryRemoved", w)
		}
	}
}

func TestDegreeSpaceWrapAndCollapse(t *testing.T) {
	cases := map[string]string{
		"37.5°C":    "37.5 deg C",  // insert both sides
		"37.5 °C":   "37.5 deg C",  // collapse before
		"37.5° C":   "37.5 deg C",  // collapse after
		"37.5 ° C":  "37.5 deg C",  // collapse both
		"°C start":  "deg C start", // no leading space at start
		"tilt 20°":  "tilt 20 deg", // no trailing space at end
		"20°\nnext": "20 deg\nnext",
	}
	for in, want := range cases {
		if got := run(t, in).Text; got != want {
			t.Errorf("Sanitise(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCollapseOnlyAtSubstitutionSites(t *testing.T) {
	// Legitimate doubled spaces in clean text must pass byte-identical (A2).
	in := "Findings  are  stable.  Next line."
	if got := run(t, in).Text; got != in {
		t.Errorf("clean doubled spaces were modified: %q", got)
	}
}

func TestIdempotence(t *testing.T) {
	inputs := []string{
		"37.5°C, µg/L, β-hCG",
		"plain ASCII stays",
		"40 deg C unchanged",
	}
	for _, in := range inputs {
		once := run(t, in).Text
		twice := run(t, once)
		if twice.Text != once {
			t.Errorf("not idempotent: %q -> %q -> %q", in, once, twice.Text)
		}
		if twice.Substituted != 0 {
			t.Errorf("second pass reported substitutions for %q", in)
		}
	}
}

func TestLatin1ModePassthrough(t *testing.T) {
	res, err := Sanitise("café µg 5–10", Options{Charset: "latin1"})
	if err != nil {
		t.Fatal(err)
	}
	// Latin-1 printables pass through; non-latin1 chars still substituted.
	if res.Text != "café µg 5-10" {
		t.Errorf("got %q", res.Text)
	}
	if res.Substituted != 1 {
		t.Errorf("substituted = %d, want 1 (en dash only)", res.Substituted)
	}
}

func TestReservedDelimitersPassThrough(t *testing.T) {
	// FR-021: without --hl7-escape, input delimiters are untouched.
	in := `A|B^C~D&E\F`
	if got := run(t, in).Text; got != in {
		t.Errorf("delimiters modified: %q", got)
	}
}

func TestNoDelimiterEverIntroduced(t *testing.T) {
	// FR-012 spot check at the pass level.
	res := run(t, "≈ « » ↑ ↓")
	if strings.ContainsAny(res.Text, `|^~&\`) {
		t.Errorf("substitution introduced a reserved delimiter: %q", res.Text)
	}
}

// Review C4: a site override for a Latin-1 codepoint applies in latin1
// mode (explicit site intent beats passthrough); other Latin-1 still passes.
func TestLatin1ModeOverrideWins(t *testing.T) {
	res, err := Sanitise("ß-hCG café", Options{Charset: "latin1", Overrides: map[rune]string{0x00DF: "beta"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "beta-hCG café" {
		t.Errorf("got %q", res.Text)
	}
	if res.Substituted != 1 {
		t.Errorf("substituted = %d, want 1", res.Substituted)
	}
}

// Review C5: combining marks from the supplementary combining blocks are
// dropped like U+0300-U+036F, not reported unmappable.
func TestSupplementaryCombiningMarksDropped(t *testing.T) {
	res, err := Sanitise("a⃗ b᪰ c᷀ d︠", Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "a b c d" || res.Unmappable != 0 {
		t.Errorf("got %q unmappable=%d", res.Text, res.Unmappable)
	}
}

// AU localisation 4.10 (SPIA units): superscripts must not appear in units and
// a power of ten is written 10^12. FR-012 forbids the caret, so a superscript
// run after a base of 10 becomes the x10E9/L convention; any other numeric base
// gets "exp"; after a unit letter the UCUM form (m2, L-1) is kept.
func TestSuperscriptExponents(t *testing.T) {
	cases := map[string]string{
		"6.1×10¹²/L":       "6.1x10E12/L",
		"4.5x10⁹/L":        "4.5x10E9/L",
		"10⁻⁶ mol":         "10E-6 mol",
		"1.73m² and cm³":   "1.73m2 and cm3",
		"mmol L⁻¹":         "mmol L-1",
		"2³ = 8":           "2exp3 = 8",
		"Ca²⁺ Cl⁻":         "Ca2+ Cl-",
		"x⁰ y⁴⁵⁶⁷⁸⁹ zⁱ wⁿ": "x0 y456789 zi wn",
		"e⁼ ⁽a⁾ ₊₋₌₍₎ H₂O": "e= (a) +-=() H2O",
		"10 ⁹":             "10 9",
	}
	for in, want := range cases {
		if got := run(t, in).Text; got != want {
			t.Errorf("Sanitise(%q) = %q, want %q", in, got, want)
		}
	}
}

// Warning aggregation must stay linear: input with many distinct unmappable
// codepoints used to scan the whole warning list per character (finding 6).
func TestManyDistinctUnmappablesIsLinear(t *testing.T) {
	const n = 300000
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteRune(rune(0x40000 + i)) // planes 4-8 are unassigned: never mapped
	}
	start := time.Now()
	res := run(t, sb.String())
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("took %v for %d distinct unmappables", d, n)
	}
	if len(res.Warnings) != n || res.Warnings[0].Codepoint != 0x40000 || res.Warnings[n-1].Codepoint != 0x40000+n-1 {
		t.Errorf("warnings %d, first %U last %U", len(res.Warnings), res.Warnings[0].Codepoint, res.Warnings[len(res.Warnings)-1].Codepoint)
	}
	if res.Unmappable != n {
		t.Errorf("unmappable = %d", res.Unmappable)
	}
}
