// Result and Warning types shared across the conversion pipeline
// (data-model.md). Warning is structurally incapable of carrying report
// text: codepoint, count, and category only (constitution Principle V).
package sanitise

import "fmt"

// Category classifies a warning.
type Category int

const (
	CategoryUnmappable Category = iota
	CategoryRemoved
	CategoryTabReplaced
	CategoryEncoding
)

// Warning is one aggregated diagnostic. It has no text field by design.
type Warning struct {
	Codepoint rune
	Count     int
	Category  Category
}

// String renders the fixed metadata-only diagnostic line for the warning.
func (w Warning) String() string {
	switch w.Category {
	case CategoryUnmappable:
		return fmt.Sprintf("unmappable character U+%04X replaced (x%d)", w.Codepoint, w.Count)
	case CategoryRemoved:
		return fmt.Sprintf("removed character U+%04X (x%d)", w.Codepoint, w.Count)
	case CategoryTabReplaced:
		return fmt.Sprintf("tab replaced with space (x%d)", w.Count)
	}
	// CategoryEncoding — the category set is internal and exhaustive.
	return "input decoded as Windows-1252 (fallback)"
}

// Result carries the sanitised text and its metadata.
type Result struct {
	Text        string
	Substituted int
	Unmappable  int
	Warnings    []Warning
}

// Modified reports whether any character-level correction was applied to the
// document content (FR-031): a substitution, unmappable replacement, character
// removal, or tab replacement. Line-break normalisation and encoding
// transcoding are representation, not corrections, and do not count.
func (res Result) Modified() bool {
	if res.Substituted > 0 || res.Unmappable > 0 {
		return true
	}
	for _, w := range res.Warnings {
		if w.Category == CategoryRemoved || w.Category == CategoryTabReplaced {
			return true
		}
	}
	return false
}

// addWarning aggregates by (category, codepoint), preserving first-occurrence
// order for determinism (NFR-002).
func (res *Result) addWarning(cat Category, cp rune) {
	for i := range res.Warnings {
		if res.Warnings[i].Category == cat && res.Warnings[i].Codepoint == cp {
			res.Warnings[i].Count++
			return
		}
	}
	res.Warnings = append(res.Warnings, Warning{Codepoint: cp, Count: 1, Category: cat})
}
