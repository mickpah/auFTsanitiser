package decode

import (
	"strings"
	"testing"
	"unicode/utf16"
)

// encode helpers for the parity matrix.
func utf16Bytes(s string, bigEndian bool, bom bool) []byte {
	units := utf16.Encode([]rune(s))
	var b []byte
	if bom {
		units = append([]uint16{0xFEFF}, units...)
	}
	for _, u := range units {
		if bigEndian {
			b = append(b, byte(u>>8), byte(u))
		} else {
			b = append(b, byte(u), byte(u>>8))
		}
	}
	return b
}

func cp1252Bytes() []byte {
	// "37.5°C café – µg" in CP1252: ° 0xB0, é 0xE9, – 0x96 (dash in 0x80-0x9F
	// range where CP1252 differs from Latin-1), µ 0xB5.
	return []byte("37.5\xb0C caf\xe9 \x96 \xb5g")
}

func TestFiveEncodingParity(t *testing.T) {
	logical := "37.5°C café – µg"
	utf8Plain := []byte(logical)
	utf8BOM := append([]byte{0xEF, 0xBB, 0xBF}, utf8Plain...)

	variants := map[string][]byte{
		"utf8":          utf8Plain,
		"utf8-bom":      utf8BOM,
		"utf16le":       utf16Bytes(logical, false, true),
		"utf16be":       utf16Bytes(logical, true, true),
		"utf16le-nobom": utf16Bytes(logical, false, false),
		"utf16be-nobom": utf16Bytes(logical, true, false),
		"cp1252":        cp1252Bytes(),
	}
	var first string
	for name, raw := range variants {
		got, err := Decode(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if first == "" {
			first = got.Text
		}
		if got.Text != logical {
			t.Errorf("%s: decoded %q, want %q", name, got.Text, logical)
		}
		// SourceEncoding pin: the BOM-less heuristic paths report the plain
		// utf16le/utf16be names.
		if want := strings.TrimSuffix(name, "-nobom"); got.SourceEncoding != want {
			t.Errorf("%s: SourceEncoding %q, want %q", name, got.SourceEncoding, want)
		}
	}
}

func TestSourceEncodingAndFallbackWarning(t *testing.T) {
	got, err := Decode(cp1252Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceEncoding != "cp1252" {
		t.Errorf("SourceEncoding %q, want cp1252", got.SourceEncoding)
	}
	if len(got.Warnings) == 0 {
		t.Error("cp1252 fallback must record a warning")
	}

	clean, err := Decode([]byte("plain ascii"))
	if err != nil {
		t.Fatal(err)
	}
	if clean.SourceEncoding != "utf8" || len(clean.Warnings) != 0 {
		t.Errorf("clean ASCII: %q %v", clean.SourceEncoding, clean.Warnings)
	}
}

func TestCP1252UndefinedBytesBecomeC1Controls(t *testing.T) {
	// 0x81 0x8D 0x8F 0x90 0x9D are undefined in CP1252: decode to the
	// corresponding C1 codepoints so the sanitiser removes them with a
	// warning (research.md R2).
	got, err := Decode([]byte{'a', 0x81, 'b'})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "a\u0081b" {
		t.Errorf("got %q", got.Text)
	}
}

func TestCP1252PunctuationRange(t *testing.T) {
	// Spot-check the 0x80-0x9F difference table.
	cases := map[byte]rune{
		0x80: 0x20AC, // euro
		0x91: 0x2018, // left single quote
		0x92: 0x2019,
		0x93: 0x201C,
		0x94: 0x201D,
		0x95: 0x2022, // bullet
		0x96: 0x2013, // en dash
		0x97: 0x2014, // em dash
		0x99: 0x2122, // TM
		0x85: 0x2026, // ellipsis
	}
	for b, want := range cases {
		got, err := Decode([]byte{b})
		if err != nil {
			t.Fatal(err)
		}
		if []rune(got.Text)[0] != want {
			t.Errorf("0x%02X -> U+%04X, want U+%04X", b, []rune(got.Text)[0], want)
		}
	}
}

func TestTruncatedUTF16IsConversionError(t *testing.T) {
	raw := utf16Bytes("abc", false, true)
	raw = raw[:len(raw)-1] // odd byte count after BOM detection
	if _, err := Decode(raw); err == nil {
		t.Error("truncated UTF-16 must be a conversion error")
	}
}

func TestBOMOnlyInput(t *testing.T) {
	got, err := Decode([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "" {
		t.Errorf("BOM-only input should decode to empty, got %q", got.Text)
	}
}

func TestEmptyInput(t *testing.T) {
	got, err := Decode(nil)
	if err != nil || got.Text != "" {
		t.Errorf("empty input: %q %v", got.Text, err)
	}
}

// Review C6: a UTF-8 BOM followed by invalid UTF-8 takes the same
// Windows-1252 fallback as BOM-less input, instead of yielding U+FFFD.
func TestUTF8BOMWithInvalidUTF8FallsBackToCP1252(t *testing.T) {
	got, err := Decode([]byte{0xEF, 0xBB, 0xBF, 'c', 'a', 'f', 0xE9, 0x96})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "café–" || got.SourceEncoding != "cp1252" {
		t.Errorf("got %q (%s)", got.Text, got.SourceEncoding)
	}
	if len(got.Warnings) != 1 {
		t.Errorf("want the cp1252 fallback warning, got %v", got.Warnings)
	}
}
