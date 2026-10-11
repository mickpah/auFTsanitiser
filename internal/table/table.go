// Package table holds the compiled-in codepoint→replacement substitution
// table (PRD section 7, FR-011) and its validation rules (FR-012). Lookups
// are keyed; the map is never iterated on the output path (NFR-002) — All()
// exists for tests and validation only.
package table

import (
	"fmt"
	"sort"
	"strings"
)

// reserved delimiters that no replacement may contain (FR-012).
const reserved = `|^~&\`

// Lookup returns the replacement for r and whether one exists. An empty
// replacement with ok=true means "remove" (zero-width characters, soft
// hyphen).
func Lookup(r rune) (string, bool) {
	rep, ok := substitutions[r]
	return rep, ok
}

// SpaceWrapped reports whether r's replacement takes surrounding spaces with
// local collapse (PRD Q4: degree-sign family only).
func SpaceWrapped(r rune) bool {
	return spaceWrapped[r]
}

// All returns every mapped codepoint in ascending order (test/validation use
// only).
func All() []rune {
	keys := make([]rune, 0, len(substitutions))
	for r := range substitutions {
		keys = append(keys, r)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// Validate enforces the table invariants on m: printable-ASCII delimiter-free
// replacements keyed by codepoints outside the permitted ASCII range. Used by
// the core table's self-test and by --table override loading.
func Validate(m map[rune]string) error {
	for r, rep := range m {
		if r >= 0x20 && r <= 0x7E {
			return fmt.Errorf("table entry U+%04X is inside the permitted ASCII range", r)
		}
		if strings.ContainsAny(rep, reserved) {
			return fmt.Errorf("table entry U+%04X replacement contains a reserved delimiter", r)
		}
		for i := 0; i < len(rep); i++ {
			if rep[i] < 0x20 || rep[i] > 0x7E {
				return fmt.Errorf("table entry U+%04X replacement contains non-printable-ASCII byte 0x%02X", r, rep[i])
			}
		}
	}
	return nil
}
