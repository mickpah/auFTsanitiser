package sanitise

import (
	"reflect"
	"regexp"
	"testing"
)

// Principle V: the Warning type must be structurally incapable of carrying
// report text — codepoint, count, and category only.
func TestWarningHasNoTextField(t *testing.T) {
	typ := reflect.TypeOf(Warning{})
	allowed := map[string]bool{"Codepoint": true, "Count": true, "Category": true}
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if !allowed[name] {
			t.Errorf("Warning has unexpected field %q — PHI risk", name)
		}
		if typ.Field(i).Type.Kind() == reflect.String && name != "Category" {
			t.Errorf("Warning field %q is a string — PHI risk", name)
		}
	}
	if typ.NumField() != 3 {
		t.Errorf("Warning should have exactly 3 fields, has %d", typ.NumField())
	}
}

// Modified (FR-031): true for any character-level correction — counters or a
// removal/tab warning; false for the encoding-fallback warning alone.
func TestModified(t *testing.T) {
	cases := []struct {
		name string
		res  Result
		want bool
	}{
		{"empty", Result{}, false},
		{"substituted", Result{Substituted: 1}, true},
		{"unmappable", Result{Unmappable: 1}, true},
		{"removed-warning-only", Result{Warnings: []Warning{{Codepoint: 0x00AD, Count: 1, Category: CategoryRemoved}}}, true},
		{"tab-warning-only", Result{Warnings: []Warning{{Count: 1, Category: CategoryTabReplaced}}}, true},
		{"encoding-warning-only", Result{Warnings: []Warning{{Count: 1, Category: CategoryEncoding}}}, false},
	}
	for _, c := range cases {
		if got := c.res.Modified(); got != c.want {
			t.Errorf("%s: Modified() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestWarningAggregationAndOrdering(t *testing.T) {
	var res Result
	res.addWarning(CategoryUnmappable, 0x2603)
	res.addWarning(CategoryRemoved, 0x200B)
	res.addWarning(CategoryUnmappable, 0x2603)
	res.addWarning(CategoryUnmappable, 0x2604)
	res.addWarning(CategoryUnmappable, 0x2603)

	if len(res.Warnings) != 3 {
		t.Fatalf("want 3 aggregated warnings, got %d", len(res.Warnings))
	}
	// Deterministic order: first occurrence.
	if res.Warnings[0].Codepoint != 0x2603 || res.Warnings[0].Count != 3 {
		t.Errorf("warning[0] = %+v, want U+2603 x3", res.Warnings[0])
	}
	if res.Warnings[1].Codepoint != 0x200B || res.Warnings[1].Category != CategoryRemoved {
		t.Errorf("warning[1] = %+v, want removed U+200B", res.Warnings[1])
	}
	if res.Warnings[2].Codepoint != 0x2604 || res.Warnings[2].Count != 1 {
		t.Errorf("warning[2] = %+v, want U+2604 x1", res.Warnings[2])
	}
}

func TestWarningStringsAreMetadataOnly(t *testing.T) {
	patterns := map[Category]*regexp.Regexp{
		CategoryUnmappable:  regexp.MustCompile(`^unmappable character U\+[0-9A-F]{4,6} replaced \(x\d+\)$`),
		CategoryRemoved:     regexp.MustCompile(`^removed character U\+[0-9A-F]{4,6} \(x\d+\)$`),
		CategoryTabReplaced: regexp.MustCompile(`^tab replaced with space \(x\d+\)$`),
		CategoryEncoding:    regexp.MustCompile(`^input decoded as Windows-1252 \(fallback\)$`),
	}
	ws := []Warning{
		{Codepoint: 0x2603, Count: 3, Category: CategoryUnmappable},
		{Codepoint: 0x200B, Count: 2, Category: CategoryRemoved},
		{Codepoint: 0x0009, Count: 4, Category: CategoryTabReplaced},
		{Codepoint: 0, Count: 1, Category: CategoryEncoding},
	}
	for _, w := range ws {
		s := w.String()
		re, ok := patterns[w.Category]
		if !ok {
			t.Fatalf("no pattern for category %v", w.Category)
		}
		if !re.MatchString(s) {
			t.Errorf("warning string %q does not match required shape %v", s, re)
		}
	}
}
