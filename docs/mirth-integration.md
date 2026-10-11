# Mirth Connect integration

How to invoke `auFTsanitiser` from a Mirth Connect channel during HL7 v2.4
transformation, and where the sanitiser's responsibility begins and ends.

## Scope: substitution, not field escaping

`auFTsanitiser` sanitises **report text** — it replaces every character outside the
AU-permitted set (printable ASCII 0x20-0x7E plus line breaks) with a clinically
equivalent ASCII substitution. The Australian localisation of HL7 v2.4 prohibits
the `\Xdddd\` / `\Cxxyy\` / `\Mxxyyzz\` escape sequences, so substituting the
characters before the message is built is the only conformant path.

Two distinct concerns, kept separate on purpose:

1. **Character sanitisation (this tool, default mode).** Extended characters →
   ASCII. Output still contains raw text that may include HL7 delimiters
   (`|^~&\`) if they were literally present in the report.
2. **Field escaping (delimiter encoding).** Turning a literal `|`, `^`, `~`, `&`,
   `\`, or a line break into its HL7 escape (`\F\`, `\S\`, `\R\`, `\T\`, `\E\`,
   `\.br\`) so the value embeds safely in one field.

You can let **Mirth** own concern 2 (the usual arrangement — the channel builds
OBX-5 and its own serialization escapes delimiters), **or** delegate it to this
tool with `--hl7-escape`. Do exactly one of them, never both, or values get
double-escaped. Per the AU localisation and the Medical Objects HL7 v2 parsing
reference, the permitted content set is ASCII 13 (CR) plus 32-126; the escape is
a single left-to-right positional pass so double-escaping is structurally
impossible within a single run — but running two escapers in series is not.

### Recommended split

| Concern | Owner | Why |
|---|---|---|
| UTF-8 / extended char → ASCII | `auFTsanitiser` (default) | The one conformant path under the AU escape prohibition |
| Delimiter / line-break escaping into OBX-5 | Mirth channel serializer | Mirth already escapes field content when it assembles the segment; let it |

Use `--hl7-escape` only when you are inserting the payload as a pre-escaped
literal and Mirth will **not** re-escape it (e.g. writing directly into a raw
segment string). If in doubt, leave `--hl7-escape` off and let Mirth escape.

## Invocation from a channel

The tool reads stdin with `-` and writes the payload to stdout; diagnostics go to
stderr only and never contain report text (codepoints and counts only). For
programmatic use, prefer `--format json` so the channel gets structured metadata
on stdout and an always-empty stderr.

### JSON mode (recommended for transformers)

```bash
auFTsanitiser - --format json
```

stdout is exactly one object:

```json
{
  "text": "37.5 deg C beta-hCG",
  "encoding": "utf8",
  "error": "",
  "warnings": ["unmappable character U+2603 replaced (x1)"],
  "substituted": 12,
  "unmappable": 1,
  "modified": true
}
```

A JavaScript/E4X transformer step invoking the binary via a shell/exec helper
then assigns `result.text` into OBX-5 and can log `substituted` / `unmappable`
counts and `warnings` to the channel log without ever touching PHI. On a usage
error (unknown flag, unparseable invocation) the tool falls back to a text
diagnostic on stderr and exit 2 — the JSON-envelope guarantee only holds once
option parsing succeeds.

### Capturing the correction event

`modified` is the correction-event flag: it is `true` exactly when the tool
applied a character-level correction to the document content — a table/fold
substitution, an unmappable `?` replacement, a character removal (zero-width,
soft hyphen, control), or a tab replacement. It is `false` when the document
passed through content-identical, even if a warning was emitted (the CP1252
fallback warning is transport, not correction) or line breaks were normalised
(representation, not correction). Do not infer the event from the counters:
removals and tab replacements modify the document with both counters at zero.

Typical transformer usage:

```javascript
var result = JSON.parse(json);            // stdout of auFTsanitiser --format json
channelMap.put('reportCorrected', result.modified);
if (result.modified) {
    logger.info('report corrected: ' + result.substituted + ' substituted, '
        + result.unmappable + ' unmappable; ' + result.warnings.join('; '));
}
```

The warning strings carry codepoints and counts only — never report text — so
they are safe to write to the channel log and to dashboards.

### Text mode

```bash
auFTsanitiser - > sanitised.txt          # payload on stdout, warnings on stderr
```

Exit codes: `0` success (including runs with substitutions/warnings); `1`
conversion failure (undecodable input, or an unmappable character under
`--strict`); `2` usage/input error. On exit 1 or 2 no payload is emitted.

## Encoding

The decoder accepts UTF-8, UTF-16 (LE/BOM), and CP1252/Latin-1 input and
produces byte-identical output across all three for the same logical text
(a CP1252 run additionally emits a fallback warning on stderr). Mirth channels
that receive mixed source encodings can pipe the raw bytes straight in.

## Site override: the AUSPDI beta feed

The compiled default maps `U+00DF` (ß) to `ss` — Unicode-correct, and it
protects German surnames (Straßer stays "Strasser", not "Strabetaer"). The
AUSPDI ORU relay uses ß as a beta glyph in lab contexts
(research note R10, T044). For that feed only,
pass the shipped override:

```bash
auFTsanitiser - --table docs/overrides/auspdi-beta.tsv --format json
```

The override file maps `U+00DF` → `beta`. Overrides never relax the
permitted-character invariant — replacement bytes must be printable ASCII and
must not introduce `|^~&\` (validated at load; a bad table is a usage error,
exit 2). See `contracts/substitution-table-format.md` for the full format.

## Determinism and PHI

- **Deterministic**: identical input and flags produce byte-identical output on
  every platform (darwin/linux/windows), asserted by the `just parity` golden
  suite across the `just dist` matrix.
- **No PHI in diagnostics**: warnings and errors carry codepoints and counts
  only, never report text. Safe to write to channel logs.
- **No network calls.** The binary is fully self-contained (`CGO_ENABLED=0`
  static build).
