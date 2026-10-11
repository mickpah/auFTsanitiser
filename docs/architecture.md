# Architecture

Single static Go binary, standard library only. One pipeline, one output
gate. Mirrors the sibling `rfttoVisage` converter.

## Pipeline

    main.go
      -> cli.Run(args, stdin, stdout, stderr) int
           parseOptions            flags -> Options, usage errors exit 2 (text, even under --format json)
           recover guard           any panic -> "internal error: ..." exit 1 in the selected format
           read input              file or stdin, capped at 64 MiB; --base64-input unwrap
           table.LoadOverrides     --table file, validated at load (after input, so a caller
                                   piping a large report sees the table error, not a broken pipe)
           decode.Decode           bytes -> Unicode text + encoding warnings
           sanitise.Sanitise       the conversion pass -> Result
           hl7escape.Escape        only with --hl7-escape
           emit                    conform.Check gate -> stdout or atomic -o

Everything after parseOptions routes through `emit` in
`internal/cli/output.go`, the single point where bytes leave the process.
`conform.Check` runs there on every payload; a violation aborts with no
partial output. There is no second exit path. Every failure is a hard,
non-zero exit with a metadata-only message; a crash is exit 1, never a
stack trace with the usage code.

## Packages

| Package | Responsibility | Key fact |
|---|---|---|
| `internal/cli` | contract: flags, streams, exit codes, JSON envelope, atomic writes | usage errors are text-mode even under `--format json` |
| `internal/decode` | encoding detection: BOM, UTF-8 validity, UTF-16 NUL heuristic, CP1252 fallback | forgiving on input, strict on output |
| `internal/sanitise` | one left-to-right pass: table lookup, superscript-exponent rule, fold fallback, removals, unmappable policy; `Result`/`Warning` types | `Warning` is structurally incapable of carrying report text; aggregation is indexed, so hostile input stays linear |
| `internal/table` | compiled-in substitution table + `--table` override loading/validation | maps are keyed, never iterated on the output path |
| `internal/hl7escape` | delimiter/line-break escaping for field embedding | positional single pass; double-escaping structurally impossible |
| `internal/conform` | the executable permitted-set invariant | the same `Check` gates the CLI and every test tier |

Dependency graph is acyclic: `cli` depends on the rest; `decode` and `table`
depend on `sanitise`'s types or nothing; `conform` and `hl7escape` depend on
nothing internal.

## Invariants (constitution, non-negotiable)

- Every output byte in the permitted set; enforced at `emit`, not assumed.
- Substitutions never introduce `|` `^` `~` `&` `\` (U+2248 is `approx.`,
  never `~`). `table.Validate` enforces this for compiled and override
  entries alike.
- Deterministic and idempotent: same input + flags = same bytes on every
  platform; re-running on output is a no-op. Fold/substitution tables are
  static Go source, not runtime Unicode data, so output cannot drift with
  Unicode versions.
- No PHI in diagnostics: codepoints and counts only. No network calls.
- JSON envelope fields are ordered and additive-only; `modified` is derived
  by `Result.Modified()` (corrections only, not line-break normalisation,
  not encoding transcoding).

## Design decisions

Decisions that are not obvious from the code, with the reason they were made.

| Decision | Reason |
|---|---|
| Substitution, not escaping | The AU localisation prohibits `\X..\`, `\C..\` and `\M..\` escapes, so only an ASCII rendering is conformant. Field escaping of delimiters is a separate concern, left to Mirth unless `--hl7-escape` is asked for. |
| A superscript run after a number is an exponent, written `10E12` or `2exp3` | AU localisation 4.10 (SPIA units) requires `10^12`, but FR-012 forbids the caret in any substitution. `E` after a base of 10 is the established x10E9/L rendering; `exp` after any other base avoids a wrong value. After a unit letter the UCUM form (`m2`, `L-1`) is kept. |
| U+00DF maps to `ss`, with `beta` as a shipped override | Unicode-correct and protects German surnames; the one feed that uses ß as a beta glyph takes `docs/overrides/auspdi-beta.tsv`. |
| U+2248 is `approx.`, U+2191 is `increased` | Never `~`, never a glyph a receiver could misread; clinical sense survives the character set. |
| Diagnostics carry codepoints and counts only | The tool runs on patient reports; stderr, the JSON `error` field and a recovered panic message must be safe to log. Enforced by shape, canary and fuzz tests, not by habit. |
| Input capped at 64 MiB; hard exit on everything | A runaway stream must fail fast with exit 2 rather than take the host's memory; real reports are kilobytes. |
| One process per call, no daemon | Measured: 3 ms per spawn against 0.016 ms of work, at a few thousand messages a day. A resident process would add lifecycle, socket and PHI-residency risk for no visible gain. |
| Deterministic, static tables | The same bytes on every platform and no drift with Unicode versions, so a receiver can reverse-read any substitution from the published table. |
| Fail-closed in the channel | Any failure throws and the message ERRORs; no code path forwards unsanitised text. Retry and paging are channel configuration (destination queue, error alert). |
| Source-only releases | The deploying organisation builds from a tag it has read and validates the table under its own clinical governance; see `SUPPORT.md`. |
| Not a medical device | Intended purpose is character-set conformance; excluded as middleware and communications software. See `tga-compliance.md` for the boundary that must not be crossed. |
| Site material lives in a gitignored `private/` tree | Channel exports, environments, runbooks and real-message corpora never enter the public history; `scripts/scan-for-phi.py` fails the gate if they ever do. |

## Where to change what

| Task | Touch |
|---|---|
| Add/change a substitution | `internal/table/data.go` + a golden fixture in `testdata/`; record the clinical reasoning in the commit |
| Change how superscripts render | `exponentMarker` in `internal/sanitise/sanitise.go` + `TestSuperscriptExponents` |
| Add an accented-letter fold | `internal/sanitise/fold.go` (generated from NFKD; keep static) |
| New CLI flag | spec + `contracts/cli-contract.md` first, then `options.go`, `usageText`, tests |
| Envelope field | spec FR-031 first; append to `envelope` in `output.go`, never reorder |
| Site-specific mapping | no code — ship a `--table` override (see `docs/overrides/`) |
| Release | bump `internal/cli/version.go` (`scripts/build-deb.sh` reads it), add a `CHANGELOG.md` entry, tag; no binaries are published |
| Anonymise or scan fixtures | `scripts/anonymise-hl7.py`, `scripts/scan-for-phi.py` (both have `--self-test`) |

Test-first is the workflow: the spec rule becomes a failing test before the
code changes (the PRD and contracts are in `docs/`; the design plan and task ledger are kept outside this repository).

## Test tiers

    just check                 gofmt + go vet + just phi-scan + go test ./...   (the gate)
    just phi-scan [paths]      scanner and anonymiser self-tests, then the PHI scan of testdata/ and docs/
    golden                     testdata/*.input.txt vs *.golden.txt, byte-exact
    corpus                     real reports in Sample_reports/ (local-only, skips when absent)
    PHI leak                   stderr shape allowlist, canary planted in every failure path, FuzzRun over the CLI
    fuzz                       sanitise properties + hl7escape round-trip + FuzzRun
    bench                      NFR: 1 MB < 100 ms (about 80 ms on the dev machine; skipped under -race)
    just race                  the suite under the race detector
    just parity                golden suite vs dist binaries per OS (byte parity)
    just rhino                 Mirth code-template unit tests (needs Java and the private/ tree)

All tiers assert conformance through the same `conform.Check` the binary
ships — the tested invariant and the shipped invariant are one function.
