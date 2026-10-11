# Architecture

Single static Go binary, standard library only. One pipeline, one output
gate. Mirrors the sibling `rfttoVisage` converter.

## Pipeline

    main.go
      -> cli.Run(args, stdin, stdout, stderr) int
           parseOptions            flags -> Options, usage errors exit 2
           table.LoadOverrides     --table file, validated at load
           read input              file or stdin; --base64-input unwrap
           decode.Decode           bytes -> Unicode text + encoding warnings
           sanitise.Sanitise       the conversion pass -> Result
           hl7escape.Escape        only with --hl7-escape
           emit                    conform.Check gate -> stdout or atomic -o

Everything after parseOptions routes through `emit` in
`internal/cli/output.go` — the single point where bytes leave the process.
`conform.Check` runs there on every payload; a violation aborts with no
partial output. There is no second exit path.

## Packages

| Package | Responsibility | Key fact |
|---|---|---|
| `internal/cli` | contract: flags, streams, exit codes, JSON envelope, atomic writes | usage errors are text-mode even under `--format json` |
| `internal/decode` | encoding detection: BOM, UTF-8 validity, UTF-16 NUL heuristic, CP1252 fallback | forgiving on input, strict on output |
| `internal/sanitise` | one left-to-right pass: table lookup, fold fallback, removals, unmappable policy; `Result`/`Warning` types | `Warning` is structurally incapable of carrying report text |
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
  by `Result.Modified()` (corrections only — not line-break normalisation,
  not encoding transcoding).

## Where to change what

| Task | Touch |
|---|---|
| Add/change a substitution | `internal/table/data.go` + a golden fixture in `testdata/` |
| Add an accented-letter fold | `internal/sanitise/fold.go` (generated from NFKD; keep static) |
| New CLI flag | spec + `contracts/cli-contract.md` first, then `options.go`, `usageText`, tests |
| Envelope field | spec FR-031 first; append to `envelope` in `output.go`, never reorder |
| Site-specific mapping | no code — ship a `--table` override (see `docs/overrides/`) |
| Release | bump `internal/cli/version.go` (`scripts/build-deb.sh` reads it) |

Test-first is the workflow: the spec rule becomes a failing test before the
code changes (the PRD and contracts are in `docs/`; the design plan and task ledger are kept outside this repository).

## Test tiers

    just check                 gofmt + go vet + go test ./...   (the CI gate)
    golden                     testdata/*.input.txt vs *.golden.txt, byte-exact
    corpus                     real PHI reports in Sample_reports/ (local-only, skips when absent)
    fuzz                       sanitise properties + hl7escape round-trip
    bench                      NFR: 1 MB < 100 ms (currently ~8 ms)
    just parity                golden suite vs dist binaries per OS (byte parity)

All tiers assert conformance through the same `conform.Check` the binary
ships — the tested invariant and the shipped invariant are one function.
