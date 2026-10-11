package sanitise

import "strings"

// NormaliseLineBreaks converts every line-break convention — CRLF, lone CR,
// LF, NEL (U+0085), LS (U+2028), PS (U+2029) — to a single LF (FR-015).
// Mixed conventions within one document are handled per occurrence. The
// output line-break representation (LF, CR, or \.br\) is applied at emission.
func NormaliseLineBreaks(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\r':
			if i+1 < len(runes) && runes[i+1] == '\n' {
				i++ // CRLF is one break
			}
			b.WriteByte('\n')
		case '\n', '\u0085', '\u2028', '\u2029':
			b.WriteByte('\n')
		default:
			b.WriteRune(runes[i])
		}
	}
	return b.String()
}
