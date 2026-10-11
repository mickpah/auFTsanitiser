package hl7escape

import (
	"strings"
	"testing"

	"auFTsanitiser/internal/conform"
)

func TestEscapeMapping(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"plain text":     "plain text",
		"A|B":            `A\F\B`,
		"A^B":            `A\S\B`,
		"A~B":            `A\R\B`,
		"A&B":            `A\T\B`,
		`A\B`:            `A\E\B`,
		"line1\nline2":   `line1\.br\line2`,
		"trailing\n":     `trailing\.br\`,
		"|^~&\\":         `\F\\S\\R\\T\\E\`,
		"a||b":           `a\F\\F\b`,
		`\\`:             `\E\\E\`,
		"T&N: 5|6^7":     `T\T\N: 5\F\6\S\7`,
		`\F\ preescaped`: `\E\F\E\ preescaped`, // input backslashes are data
	}
	for in, want := range cases {
		if got := Escape(in); got != want {
			t.Errorf("Escape(%q) = %q, want %q", in, got, want)
		}
	}
}

// SC-003: escape then reference-unescape restores the sanitised original.
func TestRoundTrip(t *testing.T) {
	inputs := []string{
		"simple report line",
		"A|B^C~D&E\\F\nline two\nline three\n",
		"|||",
		"\\.br\\ literal in input",
		"mixed | and \\ and \n breaks",
		"",
	}
	for _, in := range inputs {
		esc := Escape(in)
		back, err := referenceUnescape(esc)
		if err != nil {
			t.Errorf("unescape(Escape(%q)) failed: %v (escaped %q)", in, err, esc)
			continue
		}
		if back != in {
			t.Errorf("round-trip: %q -> %q -> %q", in, esc, back)
		}
	}
}

// Escaped output must satisfy the hl7-escape conformance mode.
func TestEscapedOutputConforms(t *testing.T) {
	p := conform.Params{Charset: "ascii", LineBreak: "lf", HL7Escape: true, OutputEncoding: "utf8"}
	inputs := []string{
		"A|B^C~D&E\\F\nline2",
		"plain",
		"\n\n\n",
	}
	for _, in := range inputs {
		esc := Escape(in)
		if err := conform.Check([]byte(esc), p); err != nil {
			t.Errorf("Escape(%q) = %q violates conformance: %v", in, esc, err)
		}
		if strings.ContainsAny(strings.NewReplacer(
			`\F\`, "", `\S\`, "", `\R\`, "", `\T\`, "", `\E\`, "", `\.br\`, "",
		).Replace(esc), "|^~&\\\n\r") {
			t.Errorf("raw delimiter or break survived in %q", esc)
		}
	}
}
