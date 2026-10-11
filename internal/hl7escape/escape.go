// Package hl7escape prepares sanitised text for direct insertion into an
// FT/TX field (FR-020): the five reserved delimiters and every line break
// are escaped in one left-to-right pass. Appendix 1 documents sequential
// search-and-replace as a failure mode — the escape character is handled
// positionally like every other delimiter, so double-escaping is
// structurally impossible.
package hl7escape

import "strings"

var escapes = map[byte]string{
	'|':  `\F\`,
	'^':  `\S\`,
	'~':  `\R\`,
	'&':  `\T\`,
	'\\': `\E\`,
	'\n': `\.br\`,
}

// Escape converts sanitised text (internal LF line breaks) to
// field-embeddable form. Input backslashes are data and become \E\.
func Escape(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	for i := 0; i < len(s); i++ {
		if rep, ok := escapes[s[i]]; ok {
			b.WriteString(rep)
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
