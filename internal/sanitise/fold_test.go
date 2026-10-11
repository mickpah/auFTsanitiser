package sanitise

import "testing"

func TestFoldAccentedLatin(t *testing.T) {
	want := map[rune]string{
		'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
		'á': "a", 'à': "a", 'â': "a", 'ä': "a", 'å': "a", 'ã': "a",
		'í': "i", 'ì': "i", 'î': "i", 'ï': "i",
		'ó': "o", 'ò': "o", 'ô': "o", 'ö': "o", 'õ': "o",
		'ú': "u", 'ù': "u", 'û': "u", 'ü': "u",
		'ñ': "n", 'ç': "c", 'ý': "y", 'ÿ': "y",
		'É': "E", 'Ü': "U", 'Ñ': "N", 'Ç': "C",
		// Latin Extended-A
		'Ā': "A", 'ā': "a", 'ė': "e", 'ī': "i", 'ő': "o", 'š': "s",
		'ž': "z", 'ř': "r", 'ć': "c", 'ę': "e", 'ą': "a", 'ů': "u",
	}
	for r, exp := range want {
		got, ok := Fold(r)
		if !ok {
			t.Errorf("Fold(U+%04X) missing", r)
			continue
		}
		if got != exp {
			t.Errorf("Fold(U+%04X) = %q, want %q", r, got, exp)
		}
	}
}

func TestFoldLigatures(t *testing.T) {
	want := map[rune]string{
		0xFB00: "ff", 0xFB01: "fi", 0xFB02: "fl",
		0xFB03: "ffi", 0xFB04: "ffl",
	}
	for r, exp := range want {
		got, ok := Fold(r)
		if !ok || got != exp {
			t.Errorf("Fold(U+%04X) = %q,%v want %q", r, got, ok, exp)
		}
	}
}

func TestCombiningMarksAreDroppable(t *testing.T) {
	for _, r := range []rune{0x0300, 0x0301, 0x0308, 0x0327, 0x036F} {
		if !IsCombining(r) {
			t.Errorf("U+%04X should be recognised as a combining mark", r)
		}
	}
	for _, r := range []rune{'e', 0x00E9, 0x200B} {
		if IsCombining(r) {
			t.Errorf("U+%04X must not be a combining mark", r)
		}
	}
}

func TestFoldOutputASCIIOnly(t *testing.T) {
	for _, r := range FoldKeys() {
		rep, _ := Fold(r)
		for i := 0; i < len(rep); i++ {
			if rep[i] < 0x20 || rep[i] > 0x7E {
				t.Errorf("Fold(U+%04X) = %q contains non-ASCII byte", r, rep)
			}
		}
		if rep == "" {
			t.Errorf("Fold(U+%04X) is empty — folds must map to a base letter", r)
		}
	}
	if len(FoldKeys()) < 100 {
		t.Errorf("fold table suspiciously small: %d", len(FoldKeys()))
	}
}
