package sanitise

import (
	"regexp"
	"testing"
)

var unmappableRe = regexp.MustCompile(`^unmappable character U\+[0-9A-F]{4,6} replaced \(x\d+\)$`)

func TestUnmappableDefaultPolicy(t *testing.T) {
	// U+2603 SNOWMAN three times: ? each, ONE warning with count (FR-013).
	res, err := Sanitise("snow ☃☃ and ☃ end", Options{Charset: "ascii"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "snow ?? and ? end" {
		t.Errorf("got %q", res.Text)
	}
	if res.Unmappable != 3 {
		t.Errorf("unmappable = %d, want 3", res.Unmappable)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("want exactly one aggregated warning, got %+v", res.Warnings)
	}
	w := res.Warnings[0]
	if w.Codepoint != 0x2603 || w.Count != 3 || w.Category != CategoryUnmappable {
		t.Errorf("warning %+v, want U+2603 x3 unmappable", w)
	}
	if !unmappableRe.MatchString(w.String()) {
		t.Errorf("warning string %q violates NFR-003 shape", w.String())
	}
}

func TestUnmappableDistinctCodepoints(t *testing.T) {
	res, err := Sanitise("☃ then ⛄ then ☃", Options{Charset: "ascii"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("want one warning per distinct codepoint, got %+v", res.Warnings)
	}
	if res.Warnings[0].Codepoint != 0x2603 || res.Warnings[0].Count != 2 {
		t.Errorf("first warning %+v", res.Warnings[0])
	}
	if res.Warnings[1].Codepoint != 0x26C4 || res.Warnings[1].Count != 1 {
		t.Errorf("second warning %+v", res.Warnings[1])
	}
}

func TestStrictModeFails(t *testing.T) {
	res, err := Sanitise("ok ☃ fail", Options{Charset: "ascii", Strict: true})
	if err == nil {
		t.Fatalf("strict mode must fail, got result %+v", res)
	}
	ue, ok := err.(*UnmappableError)
	if !ok {
		t.Fatalf("want *UnmappableError, got %T", err)
	}
	if ue.Codepoint != 0x2603 {
		t.Errorf("codepoint U+%04X, want U+2603", ue.Codepoint)
	}
	// Error text is metadata-only.
	if regexp.MustCompile(`ok|fail`).MatchString(err.Error()) {
		t.Errorf("strict error leaks input text: %q", err.Error())
	}
}

func TestStrictModePassesCleanAndMappable(t *testing.T) {
	res, err := Sanitise("37.5°C µg", Options{Charset: "ascii", Strict: true})
	if err != nil {
		t.Fatalf("strict must succeed when everything maps: %v", err)
	}
	if res.Text != "37.5 deg C ug" {
		t.Errorf("got %q", res.Text)
	}
}
