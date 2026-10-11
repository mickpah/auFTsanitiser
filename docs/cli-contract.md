# Contract: auFTsanitiser CLI

**Status**: v1, shipped in 0.1.0 and hardened 2026-10-11 (recovered-panic exit, input cap, latin1/json rule). Shared operational shape with `rfttoVisage`. Breaking changes require a spec revision and a MAJOR (or, in 0.x, MINOR) version bump.

## Synopsis

```text
auFTsanitiser <input.txt | -> [options]
```

| Flag | Argument | Default | Effect |
|---|---|---|---|
| (positional) | file path or `-` | required | input source; `-` reads stdin (FR-001) |
| `-o` | file | stdout | atomic output write; no partial file on failure (FR-030) |
| `--format` | `text` \| `json` | `text` | output mode (FR-031) |
| `--charset` | `ascii` \| `latin1` | `ascii` | permitted output set (FR-010) |
| `--output-encoding` | `utf8` \| `latin1` | `utf8` | byte encoding of output (FR-010) |
| `--line-break` | `lf` \| `cr` | `lf` | output line-break representation (FR-016) |
| `--hl7-escape` | — | off | escape reserved delimiters, line breaks → `\.br\` (FR-020) |
| `--strict` | — | off | unmappable character fails conversion, exit 1 (FR-013) |
| `--table` | file | — | site substitution overrides (FR-011; format: substitution-table-format.md) |
| `--base64` | — | off | base64-encode output payload (FR-031) |
| `--base64-input` | — | off | base64-decode input first (FR-004) |
| `--help`, `-h` | — | — | usage to stdout, exit 0 (FR-033) |
| `--version` | — | — | `auFTsanitiser <version>` to stdout, exit 0 (FR-033) |

Flags accepted in any order (FR-033).

## Streams

- **text mode**: stdout carries the sanitised payload only; every warning/diagnostic goes to stderr (FR-030).
- **json mode**: stdout carries exactly one JSON object; stderr is always empty (FR-031).
- **Usage-error boundary**: the JSON-envelope guarantee applies only once option parsing succeeds. A usage error detected during flag parsing (unknown flag, missing value, invalid combination) emits the text usage diagnostic to stderr and exits 2 regardless of any `--format json` on the command line — the tool cannot reliably honour a format request inside an invocation it could not parse. Errors after successful parsing (unreadable input, invalid base64, conversion failure) follow the selected format: in json mode they produce the JSON envelope with `error` set and empty stderr.
- Warnings and errors never contain report text — codepoints and counts only (NFR-003). A recovered crash reports the Go runtime message (indices and lengths) or the panic value's type, never its content.
- Input is read in full before the `--table` file is loaded, so a caller piping a large report always receives the table diagnostic rather than a broken pipe.
- stdin and file input are capped at 64 MiB; a larger input is a usage error (exit 2) before any decoding.

## Exit codes (FR-032)

| Code | Meaning | Examples |
|---|---|---|
| 0 | success, including with substitutions/warnings | clean run; unmappables replaced with `?` in default mode |
| 1 | conversion failure | undecodable input; unmappable under `--strict`; internal conformance-gate failure; internal error (a recovered crash, reported as `internal error: <runtime message or panic type>`, never a stack trace) |
| 2 | usage/input error | unknown flag; missing input; unreadable input file; input over 64 MiB; unwritable output; invalid base64; invalid `--table` file; `--output-encoding latin1` with `--format json` without `--base64` |

On exit 1 or 2 no output payload is produced (no stdout payload in text mode, no `-o` file ever left partial).

## JSON envelope (json mode, both success and failure)

```json
{
  "text": "sanitised report text",
  "encoding": "utf8",
  "error": "",
  "warnings": ["unmappable character U+2603 replaced (x3)"],
  "substituted": 12,
  "unmappable": 3,
  "modified": true
}
```

- `encoding` is `"utf8"` or `"base64"` (when `--base64`).
- `modified` is the correction-event flag for the calling channel (FR-031): true exactly when a character-level correction was applied — substitution, unmappable replacement, character removal, or tab replacement. Line-break normalisation and encoding transcoding do not set it (representation, not correction). Counters can be zero while `modified` is true (removal-only input) and warnings can be present while it is false (cp1252 fallback in latin1 mode).
- On failure: `text` is `""`, `error` is non-empty (metadata only), exit code still 1/2.
- Field order is fixed; output is byte-deterministic for identical input and flags (NFR-002).

## Output guarantees

- Every payload byte is in the permitted set for the selected mode: ASCII 0x20–0x7E + line-break bytes (`ascii`), plus 0xA0–0xFF printables (`latin1`) (FR-010).
- Substituted text never contains `|`, `^`, `~`, `&`, `\` introduced by substitution (FR-012).
- Without `--hl7-escape`, input reserved delimiters pass through; with it, output is delimiter-clean and unescapes to the sanitised original (FR-020/021, SC-003).
- Sanitising already-conformant text is a byte-identical no-op (SC-007).
