// Package decode turns raw input bytes into Unicode text (FR-002/FR-003):
// BOM-led UTF-8/UTF-16 detection (a UTF-8 BOM over invalid UTF-8 falls back to
// Windows-1252), whole-input UTF-8 validation,
// a NUL-pattern heuristic for BOM-less UTF-16, and a Windows-1252 fallback —
// forgiving on input, strict on output (research.md R3).
package decode

import (
	"errors"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"auFTsanitiser/internal/sanitise"
)

// Result carries decoded text plus decode-stage metadata (data-model.md).
type Result struct {
	Text           string
	SourceEncoding string // utf8 | utf8-bom | utf16le | utf16be | cp1252
	Warnings       []sanitise.Warning
}

// ErrTruncatedUTF16 is a conversion failure (exit 1), not a usage error.
var ErrTruncatedUTF16 = errors.New("UTF-16 input has an odd byte count")

// cp1252High maps the 0x80-0x9F range where Windows-1252 differs from
// ISO 8859-1 / Unicode. Undefined bytes (0x81, 0x8D, 0x8F, 0x90, 0x9D) map
// to their C1 codepoints so the sanitiser removes them with a warning.
var cp1252High = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

// Decode detects the input encoding and returns Unicode text. Detection
// order (research.md R3): BOM, whole-input UTF-8 validity, UTF-16 NUL
// heuristic, Windows-1252 fallback with a metadata-only warning.
func Decode(raw []byte) (Result, error) {
	switch {
	case len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF:
		if !utf8.Valid(raw[3:]) {
			// BOM lied: same Windows-1252 fallback as BOM-less input.
			return decodeCP1252(raw[3:]), nil
		}
		return Result{Text: string(raw[3:]), SourceEncoding: "utf8-bom"}, nil
	case len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE:
		return decodeUTF16(raw[2:], false)
	case len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF:
		return decodeUTF16(raw[2:], true)
	}
	if utf8.Valid(raw) {
		if be, le := nulHeuristic(raw); be || le {
			return decodeUTF16(raw, be)
		}
		return Result{Text: string(raw), SourceEncoding: "utf8"}, nil
	}
	if be, le := nulHeuristic(raw); be || le {
		return decodeUTF16(raw, be)
	}
	return decodeCP1252(raw), nil
}

// nulHeuristic detects BOM-less UTF-16 by the NUL distribution in the first
// 512 bytes: ASCII-dominated clinical text yields a NUL in every other
// position (even for BE, odd for LE).
func nulHeuristic(raw []byte) (be, le bool) {
	if len(raw) < 4 {
		return false, false
	}
	window := raw
	if len(window) > 512 {
		window = window[:512]
	}
	var evenNul, oddNul int
	for i, b := range window {
		if b == 0 {
			if i%2 == 0 {
				evenNul++
			} else {
				oddNul++
			}
		}
	}
	half := len(window) / 2
	threshold := half * 3 / 10 // 30% of positions on one side are NUL
	if threshold < 1 {
		return false, false
	}
	return evenNul >= threshold && oddNul < threshold/2,
		oddNul >= threshold && evenNul < threshold/2
}

func decodeUTF16(payload []byte, bigEndian bool) (Result, error) {
	if len(payload)%2 != 0 {
		return Result{}, ErrTruncatedUTF16
	}
	name := "utf16le"
	if bigEndian {
		name = "utf16be"
	}
	units := make([]uint16, 0, len(payload)/2)
	for i := 0; i < len(payload); i += 2 {
		if bigEndian {
			units = append(units, uint16(payload[i])<<8|uint16(payload[i+1]))
		} else {
			units = append(units, uint16(payload[i])|uint16(payload[i+1])<<8)
		}
	}
	return Result{Text: string(utf16.Decode(units)), SourceEncoding: name}, nil
}

func decodeCP1252(raw []byte) Result {
	var b strings.Builder
	b.Grow(len(raw))
	for _, by := range raw {
		switch {
		case by < 0x80:
			b.WriteByte(by)
		case by < 0xA0:
			b.WriteRune(cp1252High[by-0x80])
		default:
			b.WriteRune(rune(by))
		}
	}
	return Result{
		Text:           b.String(),
		SourceEncoding: "cp1252",
		Warnings: []sanitise.Warning{
			{Category: sanitise.CategoryEncoding, Count: 1},
		},
	}
}
