# auFTsanitiser

A single-binary CLI that sanitises a plain-text clinical report for Australian
HL7 v2.4 messaging: UTF-8 and extended characters are replaced with clinically
equivalent ASCII substitutions conforming to the
[Australian Diagnostics and Referral Messaging — Localisation of HL7 Version 2.4](https://hl7.org.au/archive/hl7v2wg/Australian-Diagnostics-and-Referral-Messaging---Localisation-of-HL7-Version-2.4_1278265.html)
(HL7 Australia).

The AU localisation requires message content to be ASCII 32-127 plus CR (or ISO
8859-1 when MSH-18 declares it) and prohibits the hexadecimal escape sequences
HL7 International allows, so extended characters must be substituted before the
message is constructed. Examples: `37.5 deg C` from `37.5°C`, `ug/L` from
`µg/L`, `beta-hCG` from `β-hCG`, `+/-2` from `±2`, straight quotes from smart
quotes.

**Status: implemented.** Conversion, encoding detection, the HL7 escaper, the
JSON envelope, atomic writes, and site overrides are complete and covered by
unit, golden, corpus, fuzz, and benchmark tests (`just check` is the CI gate).
The authoritative product requirements are in
[`docs/prd-auFTsanitiser.md`](docs/prd-auFTsanitiser.md); the project mirrors the CLI
contract, constitution, packaging, and test tiers of its sibling utility
`rfttoVisage`. For channel-side wiring see
[`docs/mirth-integration.md`](docs/mirth-integration.md). Site deployment material
(channel export, environments, runbooks, real-message corpora) is kept outside this
repository in the gitignored `private/` folder.

## Intended purpose

auFTsanitiser is interface middleware. Its sole purpose is to render a report's
characters into the set the Australian HL7 v2.4 localisation permits so the message
can be transmitted and accepted. It does not read, interpret, score or alter the
clinical content of a report, makes no claim about any disease or condition, and
controls no device. On that basis it is not a medical device under section 41BD of
the Therapeutic Goods Act 1989 and falls within the middleware and communications
software exclusions; see [`docs/tga-compliance.md`](docs/tga-compliance.md) for the
assessment and the changes that would reopen it.

## Usage (see PRD sections 5 and 7)

```bash
auFTsanitiser <input.txt>              # sanitised text to stdout
auFTsanitiser <input.txt> -o <file>    # write to a file (atomic)
auFTsanitiser -                        # read from stdin
auFTsanitiser <input.txt> --format json    # structured output for Mirth
auFTsanitiser <input.txt> --hl7-escape     # escape HL7 delimiters, line breaks as \.br\
auFTsanitiser <input.txt> --strict         # fail on unmappable characters
auFTsanitiser <input.txt> --table site.tsv # site-specific substitution overrides
auFTsanitiser --help
auFTsanitiser --version
```

A shipped override for feeds where `U+00DF` (ß) means beta:

```bash
auFTsanitiser report.txt --table docs/overrides/auspdi-beta.tsv
```

Override format: [`docs/substitution-table-format.md`](docs/substitution-table-format.md).

Full usage reference: [`docs/auFTsanitiser.1.md`](docs/auFTsanitiser.1.md).
Internals for developers: [`docs/architecture.md`](docs/architecture.md).

Exit codes: 0 success; 1 conversion failure; 2 usage/input error. Diagnostics
go to stderr only and never contain report text (codepoints and counts only).

## Build

```bash
just build        # local binary
just check        # lint + PHI fixture scan + Go tests (CI gate)
just dist         # static cross-compiled binaries (darwin arm64/amd64, linux, windows)
just deb          # Debian 13 (amd64) package
```

Go 1.26, standard library only, `CGO_ENABLED=0` static binaries. macOS is the
development platform; Windows 11 and Debian 13 Linux are deployment targets.
