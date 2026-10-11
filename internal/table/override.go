package table

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// LoadOverrides parses a --table override file
// (contracts/substitution-table-format.md): one `U+XXXX<TAB>replacement`
// per line, '#' comments and blank lines ignored, later lines win, empty
// replacement means remove. Every violation is a usage error; diagnostics
// reference line numbers and codepoints only.
func LoadOverrides(path string) (map[rune]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read table file")
	}
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("table file is not valid UTF-8")
	}
	m := make(map[rune]string)
	for n, line := range strings.Split(string(raw), "\n") {
		lineNo := n + 1
		line = strings.TrimSuffix(line, "\r") // CRLF tables from Windows editors
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cp, rep, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("table line %d: separator must be one TAB", lineNo)
		}
		if strings.ContainsRune(rep, '\t') {
			return nil, fmt.Errorf("table line %d: separator must be exactly one TAB", lineNo)
		}
		if !strings.HasPrefix(cp, "U+") || len(cp) < 6 || len(cp) > 8 {
			return nil, fmt.Errorf("table line %d: codepoint must be U+ and 4-6 hex digits", lineNo)
		}
		v, err := strconv.ParseUint(cp[2:], 16, 32)
		if err != nil || strings.ToUpper(cp[2:]) != cp[2:] {
			return nil, fmt.Errorf("table line %d: codepoint must be U+ and 4-6 uppercase hex digits", lineNo)
		}
		r := rune(v)
		if !utf8.ValidRune(r) {
			return nil, fmt.Errorf("table line %d: U+%04X is not a Unicode scalar value", lineNo, v)
		}
		single := map[rune]string{r: rep}
		if err := Validate(single); err != nil {
			return nil, fmt.Errorf("table line %d: %v", lineNo, err)
		}
		m[r] = rep
	}
	return m, nil
}
