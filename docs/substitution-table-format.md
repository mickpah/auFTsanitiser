# Contract: substitution table override file (`--table`)

**Status**: v1 target. The compiled-in core table (PRD section 7) is normative; this file format carries site-specific additions and overrides (FR-011).

## Format

Plain UTF-8 text, line-oriented:

```text
# comment lines start with '#'; blank lines ignored
U+2191	(up)
U+2193	(down)
U+00B0	deg
U+00AD	
```

- One mapping per line: `U+XXXX` (4–6 uppercase hex digits), a single TAB, then the replacement string running to end of line. LF and CRLF line endings are both accepted (a trailing CR is not part of the replacement).
- An empty replacement (nothing after the TAB) means **remove** the character.
- Later lines override earlier lines; the whole file overrides the compiled table (last wins).
- Trailing spaces in the replacement are significant (e.g. `EUR `).

## Validation (all violations are usage errors, exit 2)

1. Replacement bytes must be printable ASCII 0x20–0x7E.
2. Replacement must not contain `|`, `^`, `~`, `&`, or `\` (FR-012 holds under overrides).
3. Codepoint field must parse as `U+` followed by 4–6 hex digits; the codepoint must be a valid Unicode scalar value outside 0x20–0x7E (entries for already-permitted ASCII are rejected — they would break idempotence).
4. Separator must be exactly one TAB.
5. File must be valid UTF-8 and readable.

Diagnostics for rejected files reference line numbers and codepoints only — never file content beyond the offending codepoint (NFR-003 applies to table files conservatively, since sites may paste report fragments by mistake).
