package table

import (
	"strings"
	"testing"
)

// FR-012: no replacement may contain an HL7 reserved delimiter.
const reservedDelims = `|^~&\`

func TestEveryReplacementIsPrintableASCIIAndDelimiterFree(t *testing.T) {
	for _, r := range All() {
		rep, ok := Lookup(r)
		if !ok {
			t.Fatalf("All() returned U+%04X but Lookup misses it", r)
		}
		for i := 0; i < len(rep); i++ {
			b := rep[i]
			if b < 0x20 || b > 0x7E {
				t.Errorf("U+%04X replacement contains non-printable byte 0x%02X", r, b)
			}
		}
		if strings.ContainsAny(rep, reservedDelims) {
			t.Errorf("U+%04X replacement %q contains a reserved delimiter (FR-012)", r, rep)
		}
	}
}

func TestNoPermittedASCIIKeys(t *testing.T) {
	// Entries for already-permitted ASCII would break idempotence.
	for _, r := range All() {
		if r >= 0x20 && r <= 0x7E {
			t.Errorf("table contains permitted-ASCII key U+%04X", r)
		}
	}
}

func TestAllIsSortedAndMatchesLookup(t *testing.T) {
	keys := All()
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Fatalf("All() not strictly sorted at %d: U+%04X >= U+%04X", i, keys[i-1], keys[i])
		}
	}
	if len(keys) < 100 {
		t.Errorf("table suspiciously small: %d entries", len(keys))
	}
}

func TestPRDEntries(t *testing.T) {
	want := map[rune]string{
		0x00A0: " ",        // no-break space
		0x202F: " ",        // narrow NBSP
		0x200B: "",         // zero-width space (remove)
		0x200C: "",         // zero-width non-joiner
		0x200D: "",         // zero-width joiner
		0xFEFF: "",         // BOM / zero-width no-break
		0x00AD: "",         // soft hyphen
		0x2018: "'",        // left single quote
		0x2019: "'",        // right single quote
		0x2032: "'",        // prime
		0x201C: `"`,        // left double quote
		0x201D: `"`,        // right double quote
		0x2033: `"`,        // double prime
		0x201A: "'",        // single low-9
		0x201E: `"`,        // double low-9
		0x2013: "-",        // en dash
		0x2014: "-",        // em dash
		0x2015: "-",        // horizontal bar
		0x2212: "-",        // minus sign
		0x2026: "...",      // ellipsis
		0x2022: "*",        // bullet
		0x25CF: "*",        // black circle
		0x00B7: "*",        // middle dot
		0x00AB: "<<",       // left guillemet
		0x00BB: ">>",       // right guillemet
		0x2020: "+",        // dagger
		0x2021: "++",       // double dagger
		0x00B0: "deg",      // degree
		0x2103: "deg C",    // degree celsius
		0x2109: "deg F",    // degree fahrenheit
		0x00B5: "u",        // micro sign
		0x03BC: "u",        // greek small mu (PRD signs table)
		0x00B1: "+/-",      // plus-minus
		0x00D7: "x",        // multiplication
		0x00F7: "/",        // division
		0x2044: "/",        // fraction slash
		0x2264: "<=",       // less-or-equal
		0x2265: ">=",       // greater-or-equal
		0x2260: "not =",    // not equal
		0x2248: "approx.",  // almost equal — NEVER "~"
		0x2030: "per 1000", // per mille
		0x00B9: "1",
		0x00B2: "2",
		0x00B3: "3",
		0x207F: "n",
		0x2080: "0", // subscript digits
		0x2089: "9",
		0x00BD: "1/2",
		0x00BC: "1/4",
		0x00BE: "3/4",
		0x2153: "1/3",
		0x2154: "2/3",
		0x215B: "1/8",
		0x215C: "3/8",
		0x215D: "5/8",
		0x215E: "7/8",
		0x2116: "No.",
		0x2190: "<-",
		0x2192: "->",
		0x2191: "increased",
		0x2193: "decreased",
		0x03B1: "alpha",
		0x0391: "alpha",
		0x03B2: "beta",
		0x0392: "beta",
		0x03B3: "gamma",
		0x03B4: "delta",
		0x0394: "delta",
		0x03BA: "kappa",
		0x03BB: "lambda",
		0x03C3: "sigma",
		0x03C2: "sigma", // final sigma
		0x03C4: "tau",
		0x03C6: "phi",
		0x03C9: "omega",
		0x03A9: "omega",
		0x2122: "(TM)",
		0x00AE: "(R)",
		0x00A9: "(C)",
		0x20AC: "EUR ",
		0x00A3: "GBP ",
		0x00A5: "JPY ",
		0x00A2: "c",
		0x00C6: "AE",
		0x00E6: "ae",
		0x0152: "OE",
		0x0153: "oe",
		0x00DF: "ss",
		0x00D0: "D",
		0x00F0: "d",
		0x00DE: "Th",
		0x00FE: "th",
		0x00D8: "O",
		0x00F8: "o",
		0x0141: "L",
		0x0142: "l",
		0x0131: "i",
	}
	for r, exp := range want {
		got, ok := Lookup(r)
		if !ok {
			t.Errorf("U+%04X missing from table", r)
			continue
		}
		if got != exp {
			t.Errorf("U+%04X: got %q want %q", r, got, exp)
		}
	}
}

func TestSuperscriptAndSubscriptEntriesComplete(t *testing.T) {
	// AU 4.10: superscripts and subscripts must not be used in units, so every
	// superscript/subscript digit and sign maps to its plain form.
	want := map[rune]string{
		0x2070: "0", 0x2071: "i", 0x2074: "4", 0x2075: "5", 0x2076: "6", 0x2077: "7", 0x2078: "8", 0x2079: "9",
		0x207A: "+", 0x207B: "-", 0x207C: "=", 0x207D: "(", 0x207E: ")",
		0x208A: "+", 0x208B: "-", 0x208C: "=", 0x208D: "(", 0x208E: ")",
	}
	for r, rep := range want {
		if got, ok := Lookup(r); !ok || got != rep {
			t.Errorf("U+%04X: got %q,%v want %q", r, got, ok, rep)
		}
	}
}

func TestSpaceWrappedEntries(t *testing.T) {
	for _, r := range []rune{0x00B0, 0x2103, 0x2109} {
		if !SpaceWrapped(r) {
			t.Errorf("U+%04X should be space-wrapped (PRD Q4)", r)
		}
	}
	for _, r := range []rune{0x00B1, 0x03B2, 0x00B5} {
		if SpaceWrapped(r) {
			t.Errorf("U+%04X must not be space-wrapped", r)
		}
	}
}

func TestGreekAlphabetComplete(t *testing.T) {
	// Every Greek letter in the standard ranges must be present (PRD: full table).
	for r := rune(0x0391); r <= 0x03A9; r++ {
		if r == 0x03A2 { // reserved, no capital final sigma
			continue
		}
		if _, ok := Lookup(r); !ok {
			t.Errorf("capital Greek U+%04X missing", r)
		}
	}
	for r := rune(0x03B1); r <= 0x03C9; r++ {
		if _, ok := Lookup(r); !ok {
			t.Errorf("small Greek U+%04X missing", r)
		}
	}
}
