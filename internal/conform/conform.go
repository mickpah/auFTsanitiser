// Package conform is the executable permitted-set check (constitution
// Principle I). The same Check gates the CLI's single output-emission point
// and every test tier (golden, corpus, fuzz), so the tested invariant and the
// shipped invariant are the same code.
//
// The invariant is mode-aware (spec FR-010): byte-level in ascii mode and in
// latin1 mode with latin1 output encoding; codepoint-level (valid UTF-8 of
// permitted codepoints) in latin1 mode with utf8 output encoding.
package conform

import (
	"fmt"
	"unicode/utf8"
)

// Params selects the permitted set for one emission mode.
type Params struct {
	Charset        string // "ascii" | "latin1"
	LineBreak      string // "lf" | "cr"
	HL7Escape      bool
	OutputEncoding string // "utf8" | "latin1"
}

// violation reports offset and codepoint only — never payload text
// (constitution Principle V).
type violation struct {
	offset int
	r      rune
	reason string
}

func (v *violation) Error() string {
	return fmt.Sprintf("conformance violation at byte offset %d: U+%04X %s", v.offset, v.r, v.reason)
}

// hl7EscapeTokens are the sequences permitted between backslashes in
// field-embedding mode (FR-020).
var hl7EscapeTokens = map[string]bool{
	"F": true, "S": true, "R": true, "T": true, "E": true, ".br": true,
}

// Check returns nil when every byte of payload is inside the permitted set
// for p, else a metadata-only violation error.
func Check(payload []byte, p Params) error {
	if p.HL7Escape {
		return checkHL7Escape(payload, p)
	}
	lb := byte('\n')
	if p.LineBreak == "cr" {
		lb = '\r'
	}
	if p.Charset == "latin1" && p.OutputEncoding == "utf8" {
		return checkLatin1UTF8(payload, lb)
	}
	latin1 := p.Charset == "latin1"
	for i := 0; i < len(payload); i++ {
		b := payload[i]
		if b == lb {
			continue
		}
		if b >= 0x20 && b <= 0x7E {
			continue
		}
		if latin1 && b >= 0xA0 {
			continue
		}
		// Report the decoded codepoint when the offending byte starts a valid
		// UTF-8 sequence — diagnostics name codepoints, not encoding bytes.
		r := rune(b)
		if b >= 0x80 {
			if dr, size := utf8.DecodeRune(payload[i:]); dr != utf8.RuneError || size > 1 {
				r = dr
			}
		}
		return &violation{offset: i, r: r, reason: "outside permitted byte set"}
	}
	return nil
}

func checkLatin1UTF8(payload []byte, lb byte) error {
	for i := 0; i < len(payload); {
		r, size := utf8.DecodeRune(payload[i:])
		if r == utf8.RuneError && size == 1 {
			return &violation{offset: i, r: rune(payload[i]), reason: "invalid UTF-8 byte"}
		}
		switch {
		case r == rune(lb):
		case r >= 0x20 && r <= 0x7E:
		case r >= 0xA0 && r <= 0xFF:
		default:
			return &violation{offset: i, r: r, reason: "outside permitted codepoint set"}
		}
		i += size
	}
	return nil
}

// checkHL7Escape validates field-embedding output: printable ASCII (plus the
// Latin-1 range in latin1 charset, per output encoding), no raw line breaks,
// reserved delimiters only inside valid escape sequences.
func checkHL7Escape(payload []byte, p Params) error {
	latin1 := p.Charset == "latin1"
	for i := 0; i < len(payload); {
		b := payload[i]
		switch {
		case b == '\\':
			j := i + 1
			for j < len(payload) && payload[j] != '\\' {
				j++
			}
			if j >= len(payload) {
				return &violation{offset: i, r: rune(b), reason: "unterminated escape sequence"}
			}
			if !hl7EscapeTokens[string(payload[i+1:j])] {
				return &violation{offset: i, r: rune(b), reason: "unknown escape sequence"}
			}
			i = j + 1
		case b == '|' || b == '^' || b == '~' || b == '&':
			return &violation{offset: i, r: rune(b), reason: "raw reserved delimiter in hl7-escape mode"}
		case b >= 0x20 && b <= 0x7E:
			i++
		case latin1 && p.OutputEncoding == "latin1" && b >= 0xA0:
			i++
		case latin1 && p.OutputEncoding == "utf8" && b >= 0x80:
			r, size := utf8.DecodeRune(payload[i:])
			if r < 0xA0 || r > 0xFF { // RuneError (invalid UTF-8) is above 0xFF
				return &violation{offset: i, r: r, reason: "outside permitted codepoint set in hl7-escape mode"}
			}
			i += size
		default:
			return &violation{offset: i, r: rune(b), reason: "outside permitted byte set in hl7-escape mode"}
		}
	}
	return nil
}
