# Changelog

All notable changes to auFTsanitiser. The format follows Keep a Changelog; versions
follow Semantic Versioning (in 0.x, breaking changes bump MINOR). Releases are git tags
only; no binaries are published (`SUPPORT.md`).

## Unreleased

Nothing yet.

## 0.1.0, 2026-10-11

First public version.

- Character substitution for the Australian localisation of HL7 v2.4: every character
  outside printable ASCII plus line breaks is replaced with a fixed ASCII rendering from
  the published table (`docs/prd-auFTsanitiser.md`); Latin-1 output when selected.
- A superscript run that follows a number renders as an exponent, `10E12` after a base
  of 10 and `2exp3` after any other base, per AU localisation 4.10 without a caret;
  after a unit letter the UCUM form is kept (`m2`, `L-1`). All superscript and
  subscript digits and signs are mapped.
- Input detection for UTF-8, UTF-16 and CP1252; `--hl7-escape` field escaping;
  `--strict` failure on unmappable characters; `--table` site overrides with the shipped
  `docs/overrides/auspdi-beta.tsv`; atomic `-o` writes; `--base64` and
  `--base64-input`.
- `--format json` envelope for interface engines with the `modified` correction-event
  flag; `--format json` with `--output-encoding latin1` requires `--base64`.
- Every failure is a hard non-zero exit with a message that never carries input: exit 1
  for conversion failures and recovered panics (`internal error: ...`), exit 2 for usage
  and input errors including input over 64 MiB. Input is read before the `--table` file
  is loaded so a caller piping a large report receives the table diagnostic.
- Test tiers: unit, golden fixtures, local-only real-report corpus, PHI-leak guards
  (stderr shape allowlist, canary in every failure path, `FuzzRun` over the CLI),
  property fuzzing, 1 MB timing budget, cross-platform parity.
- `scripts/anonymise-hl7.py` (deterministic anonymisation of an HL7 corpus) and
  `scripts/scan-for-phi.py` (PHI-shaped identifier, HL7 person-field and titled-name
  scan); `just phi-scan` is part of `just check`.
- Documents: CLI contract, override format, architecture with design decisions, man
  page, Mirth integration guide with the complete error-handling calling pattern, TGA
  software regulation assessment, support and security policies.
- Apache License 2.0 (`LICENSE`, `NOTICE`), copyright 2026 Michael Paholski.
