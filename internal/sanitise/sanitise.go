// Package sanitise implements the core conversion pass (FR-010..FR-016):
// one deterministic left-to-right walk that substitutes every character
// outside the AU-permitted set with its clinically equivalent ASCII
// rendering. Line breaks are normalised to LF internally; the output
// representation (LF/CR/\.br\) is applied by the caller at emission.
package sanitise

import (
	"fmt"
	"strings"

	"auFTsanitiser/internal/table"
)

// Options selects the sanitisation mode (subset of the CLI Options relevant
// to this package; see data-model.md).
type Options struct {
	Charset   string          // "ascii" (default) | "latin1"
	Strict    bool            // FR-013: fail on unmappable instead of "?"
	Overrides map[rune]string // validated --table entries, may be nil
}

// UnmappableError is the strict-mode conversion failure. Codepoint only —
// never surrounding text (Principle V).
type UnmappableError struct {
	Codepoint rune
}

func (e *UnmappableError) Error() string {
	return fmt.Sprintf("unmappable character U+%04X (strict mode)", e.Codepoint)
}

// Sanitise runs the single-pass conversion. On strict-mode failure it
// returns a non-nil error and the (discarded by callers) partial result.
func Sanitise(text string, opts Options) (Result, error) {
	var res Result
	src := []rune(NormaliseLineBreaks(text))
	var b strings.Builder
	b.Grow(len(text) + len(text)/8)
	latin1 := opts.Charset == "latin1"

	lastEmitted := func() byte {
		s := b.String()
		if len(s) == 0 {
			return 0
		}
		return s[len(s)-1]
	}

	for i := 0; i < len(src); i++ {
		r := src[i]

		// Permitted passthrough.
		if r == '\n' || (r >= 0x20 && r <= 0x7E) {
			b.WriteRune(r)
			continue
		}
		// Latin-1 passthrough, unless a site override names the codepoint
		// (explicit site intent, e.g. U+00DF as beta).
		if _, over := opts.Overrides[r]; latin1 && r >= 0xA0 && r <= 0xFF && !over {
			b.WriteRune(r)
			continue
		}

		// Tab: plain text cannot honour \.sk n\ semantics — single space
		// plus a count warning (FR-015).
		if r == '\t' {
			b.WriteByte(' ')
			res.addWarning(CategoryTabReplaced, r)
			continue
		}

		// Explicit substitution table, site overrides first (FR-011).
		rep, ok := "", false
		if opts.Overrides != nil {
			rep, ok = opts.Overrides[r]
		}
		fromOverride := ok
		if !ok {
			rep, ok = table.Lookup(r)
		}
		if ok {
			if rep == "" {
				// Removal entries (zero-width, soft hyphen): FR-015.
				res.addWarning(CategoryRemoved, r)
				continue
			}
			if !fromOverride && table.SpaceWrapped(r) {
				// PRD Q4: insert surrounding spaces, collapsing only at
				// this substitution site (spec A2 resolution).
				if last := lastEmitted(); last != 0 && last != ' ' && last != '\n' {
					b.WriteByte(' ')
				}
				b.WriteString(rep)
				if i+1 < len(src) {
					next := src[i+1]
					if next != ' ' && next != '\n' && !IsCombining(next) {
						b.WriteByte(' ')
					}
				}
				res.Substituted++
				continue
			}
			b.WriteString(rep)
			res.Substituted++
			continue
		}

		// Decomposition fold fallback for accented Latin (FR-014).
		if rep, ok := Fold(r); ok {
			b.WriteString(rep)
			res.Substituted++
			continue
		}
		// Combining marks after an emitted base letter fold to nothing.
		if IsCombining(r) {
			res.Substituted++
			continue
		}

		// Remaining C0/C1 controls: illegal below 0x20 per AU Chapter 3;
		// removed with a metadata-only warning (FR-015).
		if r < 0x20 || (r >= 0x7F && r <= 0x9F) {
			res.addWarning(CategoryRemoved, r)
			continue
		}

		// Unmappable policy (FR-013).
		if opts.Strict {
			return res, &UnmappableError{Codepoint: r}
		}
		b.WriteByte('?')
		res.Unmappable++
		res.addWarning(CategoryUnmappable, r)
	}

	res.Text = b.String()
	return res, nil
}
