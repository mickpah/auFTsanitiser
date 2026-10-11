package hl7escape

import (
	"testing"

	"auFTsanitiser/internal/conform"
	"auFTsanitiser/internal/sanitise"
)

// FuzzEscapeRoundTrip: for arbitrary input, sanitise then escape must
// produce conformant field-embeddable text that unescapes back to the
// sanitised original (SC-003).
func FuzzEscapeRoundTrip(f *testing.F) {
	f.Add("A|B^C~D&E\\F\nline two")
	f.Add("plain")
	f.Add("\\.br\\ literal")
	f.Add("|||\n\n")
	f.Add("")

	params := conform.Params{Charset: "ascii", LineBreak: "lf", HL7Escape: true, OutputEncoding: "utf8"}
	f.Fuzz(func(t *testing.T, input string) {
		res, err := sanitise.Sanitise(input, sanitise.Options{Charset: "ascii"})
		if err != nil {
			t.Fatalf("sanitise: %v", err)
		}
		esc := Escape(res.Text)
		if cerr := conform.Check([]byte(esc), params); cerr != nil {
			t.Fatalf("escaped output nonconformant: %v (%q)", cerr, esc)
		}
		back, uerr := referenceUnescape(esc)
		if uerr != nil {
			t.Fatalf("reference unescape failed: %v (%q)", uerr, esc)
		}
		if back != res.Text {
			t.Fatalf("round-trip broken: %q -> %q -> %q", res.Text, esc, back)
		}
	})
}
