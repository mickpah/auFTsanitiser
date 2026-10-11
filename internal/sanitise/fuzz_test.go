package sanitise

import (
	"strings"
	"testing"

	"auFTsanitiser/internal/conform"
)

// FuzzSanitise asserts the five release properties for arbitrary input
// (SC-001, SC-007, FR-012, NFR-002): never panic, charset invariant,
// determinism, idempotence, and no reserved delimiter ever introduced.
func FuzzSanitise(f *testing.F) {
	f.Add("37.5°C, µg/L, β-hCG, 5–10 mm, “normal”, m², ±2")
	f.Add("plain ascii text\nwith lines\n")
	f.Add("snow ☃ and ⛄")
	f.Add("mixed\r\nbreaks\rhere\nandnel ls ps")
	f.Add("café straße Łódź ﬁle")
	f.Add("tabs\tand\x01controls\x9F")
	f.Add(`delims |^~&\ pass through`)
	f.Add("")

	params := conform.Params{Charset: "ascii", LineBreak: "lf", OutputEncoding: "utf8"}
	for _, add := range []string{} {
		f.Add(add)
	}
	f.Fuzz(func(t *testing.T, input string) {
		res, err := Sanitise(input, Options{Charset: "ascii"}) // must never panic
		if err != nil {
			t.Fatalf("non-strict Sanitise returned error: %v", err)
		}

		// SC-001: every output byte inside the permitted set.
		if cerr := conform.Check([]byte(res.Text), params); cerr != nil {
			t.Fatalf("charset invariant violated: %v", cerr)
		}

		// NFR-002: determinism.
		res2, _ := Sanitise(input, Options{Charset: "ascii"})
		if res2.Text != res.Text || res2.Substituted != res.Substituted || res2.Unmappable != res.Unmappable {
			t.Fatal("nondeterministic result")
		}

		// SC-007: idempotence.
		again, err := Sanitise(res.Text, Options{Charset: "ascii"})
		if err != nil {
			t.Fatalf("re-sanitise errored: %v", err)
		}
		if again.Text != res.Text {
			t.Fatalf("not idempotent: %q -> %q", res.Text, again.Text)
		}

		// FR-012: substitution never introduces a reserved delimiter.
		if !strings.ContainsAny(input, `|^~&\`) && strings.ContainsAny(res.Text, `|^~&\`) {
			t.Fatalf("delimiter introduced: %q -> %q", input, res.Text)
		}
	})
}
