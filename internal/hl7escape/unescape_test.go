package hl7escape

import (
	"fmt"
	"testing"
)

// referenceUnescape is the test-only oracle for SC-003, implementing the
// Medical Objects algorithm verbatim (research.md R5): iterate from the
// start; on a backslash, read up to the next backslash and look the token
// up. Never search-and-replace.
func referenceUnescape(s string) (string, error) {
	tokens := map[string]string{
		"F": "|", "S": "^", "R": "~", "T": "&", "E": `\`, ".br": "\n",
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] != '\\' {
			out = append(out, s[i])
			i++
			continue
		}
		j := i + 1
		for j < len(s) && s[j] != '\\' {
			j++
		}
		if j >= len(s) {
			return "", fmt.Errorf("unterminated escape at offset %d", i)
		}
		rep, ok := tokens[s[i+1:j]]
		if !ok {
			return "", fmt.Errorf("unknown escape token at offset %d", i)
		}
		out = append(out, rep...)
		i = j + 1
	}
	return string(out), nil
}

// Self-test of the oracle itself (T027).
func TestReferenceUnescape(t *testing.T) {
	cases := map[string]string{
		`plain`:         "plain",
		`A\F\B`:         "A|B",
		`A\S\B\R\C\T\D`: "A^B~C&D",
		`back\E\slash`:  `back\slash`,
		`one\.br\two`:   "one\ntwo",
		`\F\\F\`:        "||",
		`\E\\E\`:        `\\`,
		`end\.br\`:      "end\n",
		`\.br\lead`:     "\nlead",
	}
	for in, want := range cases {
		got, err := referenceUnescape(in)
		if err != nil {
			t.Errorf("referenceUnescape(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("referenceUnescape(%q) = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{`lone\`, `un\Q\known`, `empty\\pair`} {
		if _, err := referenceUnescape(bad); err == nil {
			t.Errorf("referenceUnescape(%q) should fail", bad)
		}
	}
}
