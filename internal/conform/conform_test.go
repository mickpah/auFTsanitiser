package conform

import (
	"strings"
	"testing"
)

func ascii(lb string) Params {
	return Params{Charset: "ascii", LineBreak: lb, OutputEncoding: "utf8"}
}

func TestASCIIMode(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		p       Params
		ok      bool
	}{
		{"plain text lf", "Report: 37.5 deg C.\nLine two.\n", ascii("lf"), true},
		{"empty", "", ascii("lf"), true},
		{"all printables", " !\"#$%&'()*+,-./0:;<=>?@AZ[\\]^_`az{|}~", ascii("lf"), true},
		{"utf8 multibyte", "37.5°C", ascii("lf"), false},
		{"raw latin1 byte", string([]byte{0xB5}), ascii("lf"), false},
		{"tab", "a\tb", ascii("lf"), false},
		{"nul", "a\x00b", ascii("lf"), false},
		{"del 0x7f", "a\x7fb", ascii("lf"), false},
		{"cr in lf mode", "a\rb", ascii("lf"), false},
		{"crlf in lf mode", "a\r\nb", ascii("lf"), false},
		{"cr mode ok", "a\rb", ascii("cr"), true},
		{"lf in cr mode", "a\nb", ascii("cr"), false},
	}
	for _, c := range cases {
		err := Check([]byte(c.payload), c.p)
		if c.ok && err != nil {
			t.Errorf("%s: unexpected violation: %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: expected violation, got nil", c.name)
		}
	}
}

func TestLatin1ByteMode(t *testing.T) {
	p := Params{Charset: "latin1", LineBreak: "lf", OutputEncoding: "latin1"}
	if err := Check([]byte{'a', 0xB5, 'g', '\n'}, p); err != nil {
		t.Errorf("latin1 printable byte should pass: %v", err)
	}
	if err := Check([]byte{'a', 0x9F}, p); err == nil {
		t.Error("C1 byte 0x9F must fail in latin1 byte mode")
	}
	if err := Check([]byte{'a', 0x1F}, p); err == nil {
		t.Error("C0 byte must fail in latin1 byte mode")
	}
}

func TestLatin1UTF8Mode(t *testing.T) {
	p := Params{Charset: "latin1", LineBreak: "lf", OutputEncoding: "utf8"}
	if err := Check([]byte("µg/L café\n"), p); err != nil {
		t.Errorf("latin1 codepoints as UTF-8 should pass: %v", err)
	}
	if err := Check([]byte{0xB5}, p); err == nil {
		t.Error("invalid UTF-8 must fail in utf8 encoding mode")
	}
	if err := Check([]byte("Ā"), p); err == nil {
		t.Error("codepoint above 0xFF must fail in latin1 charset")
	}
	if err := Check([]byte("–"), p); err == nil {
		t.Error("en dash must fail in latin1 charset")
	}
}

func TestHL7EscapeMode(t *testing.T) {
	p := Params{Charset: "ascii", LineBreak: "lf", HL7Escape: true, OutputEncoding: "utf8"}
	ok := []string{
		"",
		"plain text",
		`A\F\B\S\C\R\D\T\E\E\F`,
		`line one\.br\line two`,
	}
	for _, s := range ok {
		if err := Check([]byte(s), p); err != nil {
			t.Errorf("%q should pass hl7-escape mode: %v", s, err)
		}
	}
	bad := map[string]string{
		"raw pipe":       "A|B",
		"raw caret":      "A^B",
		"raw tilde":      "A~B",
		"raw ampersand":  "A&B",
		"raw lf":         "A\nB",
		"raw cr":         "A\rB",
		"lone backslash": `A\`,
		"unknown escape": `A\Q\B`,
		"empty escape":   `A\\B`,
	}
	for name, s := range bad {
		if err := Check([]byte(s), p); err == nil {
			t.Errorf("%s (%q) must fail hl7-escape mode", name, s)
		}
	}
}

func TestViolationReportsMetadataOnly(t *testing.T) {
	payload := "PATIENT SMITH µg"
	err := Check([]byte(payload), ascii("lf"))
	if err == nil {
		t.Fatal("expected violation")
	}
	msg := err.Error()
	if !strings.Contains(msg, "U+00B5") {
		t.Errorf("violation should name the codepoint, got %q", msg)
	}
	if !strings.Contains(msg, "offset") {
		t.Errorf("violation should name the byte offset, got %q", msg)
	}
	if strings.Contains(msg, "SMITH") || strings.Contains(msg, "PATIENT") {
		t.Errorf("violation must not contain payload text, got %q", msg)
	}
}

// Review C1: latin1 charset in hl7-escape mode permits the Latin-1 range in
// the selected output encoding; ascii charset still rejects it.
func TestHL7EscapeLatin1(t *testing.T) {
	bytesLatin1 := Params{Charset: "latin1", HL7Escape: true, OutputEncoding: "latin1"}
	utf8Latin1 := Params{Charset: "latin1", HL7Escape: true, OutputEncoding: "utf8"}
	asciiEsc := Params{Charset: "ascii", HL7Escape: true, OutputEncoding: "utf8"}
	if err := Check([]byte{'c', 'a', 'f', 0xE9, '\\', 'F', '\\'}, bytesLatin1); err != nil {
		t.Errorf("latin1 byte in hl7-escape latin1/latin1 should pass: %v", err)
	}
	if err := Check([]byte(`café\.br\µg`), utf8Latin1); err != nil {
		t.Errorf("latin1 codepoints as UTF-8 in hl7-escape should pass: %v", err)
	}
	bad := map[string]struct {
		payload []byte
		p       Params
	}{
		"ascii charset rejects latin1":   {[]byte("café"), asciiEsc},
		"C1 byte in latin1 byte mode":    {[]byte{'a', 0x9F}, bytesLatin1},
		"codepoint above 0xFF":           {[]byte("Ā"), utf8Latin1},
		"invalid UTF-8 in utf8 encoding": {[]byte{'a', 0xE9}, utf8Latin1},
		"raw delimiter still rejected":   {[]byte("é|"), utf8Latin1},
	}
	for name, c := range bad {
		if err := Check(c.payload, c.p); err == nil {
			t.Errorf("%s: expected violation", name)
		}
	}
}
