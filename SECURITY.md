# Security policy

## Reporting a vulnerability

Report privately through the repository's Security tab (GitHub private vulnerability
reporting). Do not open a public issue. A report is acknowledged within 14 days;
`SUPPORT.md` says what fix effort you can expect.

Include: the version (`auFTsanitiser --version`), platform, the flags used, and a
reproduction built from synthetic text. **Never include report text, message content or
any patient information.** The tool's own diagnostics are safe to paste: they carry
codepoints and counts only. If you need a whole message to reproduce, run it through
`scripts/anonymise-hl7.py` first.

## Scope

In scope: the Go sources under `internal/` and `main.go`, the scripts under `scripts/`,
and the documents that specify behaviour. Of particular interest:

- any output byte outside the permitted set (printable ASCII plus line breaks, or
  Latin-1 when selected), or a substitution that introduces `|`, `^`, `~`, `&` or `\`;
- any diagnostic, envelope field, exit path or recovered crash that carries input text;
- any input that makes the tool exit 0 with corrupted or missing text, or that defeats
  the 64 MiB input cap or the hard-exit rule;
- non-determinism: the same input and flags producing different bytes.

Out of scope: the Mirth Connect channel and its site configuration (private, not in this
repository), the operating system, and Go toolchain vulnerabilities (report those
upstream; this project rebuilds on the next release).

## Design properties you can rely on

The binary makes no network calls, has no dependencies beyond the Go standard library,
reads one input and writes one output, and never writes a partial file. Every failure is
a non-zero exit. These are enforced by tests, including a canary test that plants a
marker in every failure path and a fuzz target over the whole CLI.

## Disclosure

Fixes land on `main` and ship in the next tagged release with a `CHANGELOG.md` entry
that credits the reporter if they wish. Coordinated disclosure is preferred; 90 days
from acknowledgement is the default window, shorter if a fix ships earlier.
