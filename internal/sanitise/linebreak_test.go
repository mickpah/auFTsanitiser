package sanitise

import "testing"

func TestNormaliseLineBreaks(t *testing.T) {
	// Mixed conventions in one document (FR-015): CRLF, CR, LF, NEL, LS, PS.
	in := "one\r\ntwo\rthree\nfour\u0085five\u2028six\u2029seven"
	got := NormaliseLineBreaks(in)
	want := "one\ntwo\nthree\nfour\nfive\nsix\nseven"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestNormaliseLineBreaksEdges(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"\r\n":         "\n",
		"\r":           "\n",
		"\r\r":         "\n\n",
		"\r\n\r\n":     "\n\n",
		"\n\r":         "\n\n", // LF then lone CR: two breaks
		"a":            "a",
		"\u2028\u2029": "\n\n",
	}
	for in, want := range cases {
		if got := NormaliseLineBreaks(in); got != want {
			t.Errorf("NormaliseLineBreaks(%q) = %q, want %q", in, got, want)
		}
	}
}
