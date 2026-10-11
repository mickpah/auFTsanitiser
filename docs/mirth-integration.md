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

The channel spawns one process per OBX-5 value, writes the text to stdin, reads
the envelope from stdout, and treats anything other than exit 0 with a clean
envelope as a failure that errors the message. The pattern below is the whole
contract: process control, every exit code, and what may be logged.

### Calling pattern with error handling (Rhino, Mirth Connect 4.x)

Put the binary path, the site table and the watchdog timeout in the channel's
configuration map (Settings, Configuration Map), never inline in the script:

| Key | Value |
|---|---|
| `auftsanitiser.bin` | absolute path of the binary (a symlink that the deploy procedure repoints) |
| `auftsanitiser.table` | absolute path of the site override `.tsv` |
| `auftsanitiser.timeoutMs` | per-invocation watchdog, default 10000 |

Code-template library (pure functions, unit-testable in a Rhino shell):

```javascript
// What the channel can conclude from the exit code (cli-contract.md).
// 137 and 143 are SIGKILL/SIGTERM: the watchdog fired.
function exitCodeLabel(code) {
    switch (Number(code)) {
        case 0: return 'success';
        case 1: return 'conversion or internal error';
        case 2: return 'usage or configuration error';
        case 137: case 143: return 'killed, watchdog timeout';
        default: return 'unexpected exit';
    }
}

// At most 512 characters of stderr on one line. The tool's stderr is
// metadata-only by contract (usage text, codepoints, a recovered-panic
// message), never report text, so it is safe to carry in an error.
function stderrTail(stderr) {
    if (!stderr) {
        return '';
    }
    return String(stderr).replace(/\s*\n\s*/g, ' | ').replace(/(\s*\|\s*)+$/, '')
        .replace(/^\s+|\s+$/g, '').substring(0, 512);
}

// The Error for a non-zero exit: exit code and label, then the envelope's
// error field when the tool wrote one, else the stderr tail.
function toolFailure(exitCode, stdout, stderr) {
    var detail = '';
    try {
        detail = String(JSON.parse(String(stdout)).error || '');
    } catch (e) {
        // no envelope: usage error, watchdog kill or crash
    }
    if (!detail) {
        detail = stderrTail(stderr);
    }
    return new Error('auFTsanitiser failed: exit ' + exitCode + ' (' + exitCodeLabel(exitCode) + ')' +
        (detail ? ': ' + detail : ''));
}

// Validate one stdout capture. Anything other than a well-formed success
// envelope throws, which errors the message. Never put env.text in a message.
function parseEnvelope(s) {
    if (!s) {
        throw new Error('auFTsanitiser produced no output');
    }
    var env;
    try {
        env = JSON.parse(String(s));
    } catch (e) {
        throw new Error('auFTsanitiser output is not a valid JSON envelope');
    }
    if (env === null || typeof env !== 'object') {
        throw new Error('auFTsanitiser envelope is not an object');
    }
    if (env.error) {
        throw new Error('auFTsanitiser conversion failed: ' + env.error);
    }
    if (typeof env.text !== 'string') {
        throw new Error('auFTsanitiser envelope is truncated (no text field)');
    }
    return {
        text: env.text,
        modified: env.modified === true,
        substituted: Number(env.substituted) || 0,
        unmappable: Number(env.unmappable) || 0,
        warnings: env.warnings || []
    };
}
```

Transformer step (process control; the tool reads all of stdin before it
writes anything, so write-then-read cannot deadlock):

```javascript
function runAuFTsanitiser(bin, table, timeoutMs, text) {
    var cmd = new Packages.java.util.ArrayList();
    cmd.add(bin);
    cmd.add('-');
    cmd.add('--format');
    cmd.add('json');
    cmd.add('--table');
    cmd.add(table);
    var proc = new Packages.java.lang.ProcessBuilder(cmd).start();
    var timer = new Packages.java.util.Timer(true);
    timer.schedule(new Packages.java.util.TimerTask({
        run: function () { proc.destroyForcibly(); }
    }), timeoutMs);
    try {
        try {
            var stdin = proc.getOutputStream();
            stdin.write(new Packages.java.lang.String(text).getBytes('UTF-8'));
            stdin.close();
        } catch (e) {
            // the tool exited before draining stdin: the exit code and
            // stderr below carry the cause, not this broken-pipe exception
        }
        var stdout = String(Packages.org.apache.commons.io.IOUtils
            .toString(proc.getInputStream(), 'UTF-8'));
        var stderr = String(Packages.org.apache.commons.io.IOUtils
            .toString(proc.getErrorStream(), 'UTF-8'));
        proc.waitFor();
        if (proc.exitValue() !== 0) {
            throw toolFailure(proc.exitValue(), stdout, stderr);
        }
        return stdout;
    } finally {
        timer.cancel();
        if (proc.isAlive()) {
            proc.destroyForcibly();
        }
    }
}

var bin = String($cfg('auftsanitiser.bin') || '');
var table = String($cfg('auftsanitiser.table') || '');
if (!bin || !table) {
    throw new Error('auftsanitiser.bin / auftsanitiser.table not set in the configuration map');
}
var timeoutMs = parseInt($cfg('auftsanitiser.timeoutMs'), 10) || 10000;

// One call per OBX-5 value; env.text is written back in place.
var env = parseEnvelope(runAuFTsanitiser(bin, table, timeoutMs, obx5Value));
channelMap.put('reportCorrected', env.modified);
channelMap.put('substituted', env.substituted);
channelMap.put('unmappable', env.unmappable);
if (env.modified) {
    logger.info('auFTsanitiser ' + msg['MSH']['MSH.10']['MSH.10.1'].toString() +
        ': report corrected: substituted=' + env.substituted +
        ' unmappable=' + env.unmappable + ' warnings=' + env.warnings.join('; '));
}
```

### What each outcome means and what the step does

| Outcome | Cause | stdout | stderr | Step |
|---|---|---|---|---|
| exit 0, `error` empty | success, possibly with warnings | envelope | empty | write `text` back, record the correction event |
| exit 0, `error` set | cannot happen; treated as failure | envelope | empty | throw (`parseEnvelope`) |
| exit 1 | undecodable input, `--strict` unmappable, conformance gate, recovered crash (`internal error: ...`) | envelope with `error` | empty | throw with the envelope error |
| exit 2 after parsing | unreadable input, input over 64 MiB, invalid base64, bad `--table`, unwritable `-o` | envelope with `error` | empty | throw with the envelope error |
| exit 2 while parsing | unknown flag, bad flag value, illegal combination | empty | usage text | throw with the stderr tail: a channel configuration bug |
| exit 137 or 143 | watchdog killed a hung process | empty or partial | empty | throw "killed, watchdog timeout" |
| no output, exit 0 | binary replaced by something else | empty | anything | throw (`parseEnvelope`) |
| `IOException` on start | binary missing or not executable | n/a | n/a | the exception errors the message; the message names the path, not the report |

Every thrown message carries the exit code, the envelope `error` or up to 512
characters of stderr. None of those can contain report text: the tool's
diagnostics are codepoints, counts, flag names, table line numbers and runtime
messages only (see `cli-contract.md`).

### Fail-closed and alerting

- A throw in the transformer sets the message to ERROR and nothing is forwarded.
  No code path returns the original text.
- Run the step in a **destination** transformer with the destination queue
  enabled (Queue Messages: Always, retry count as the site decides) so a
  transient failure, such as a binary being redeployed, is retried in order
  rather than NAKed back to the sender. In the source connector a throw NAKs
  the sender and the message must be reprocessed by hand.
- Configure a channel-scoped alert (Settings, Alerts, trigger "channel error"
  on this channel) so the operator is paged on the first failure. Without it a
  failure is visible only in the dashboard.
- Log only metadata: the MSH-10 control id, the counters and the warning
  strings. Never log `env.text`, the input, or an exception built from
  `JSON.parse` of the payload. The `reportCorrected`, `substituted` and
  `unmappable` channel-map values are safe for dashboards.

### Capturing the correction event

`modified` is the correction-event flag: it is `true` exactly when the tool
applied a character-level correction to the document content, a table or fold
substitution, an unmappable `?` replacement, a character removal (zero-width,
soft hyphen, control), or a tab replacement. It is `false` when the document
passed through content-identical, even if a warning was emitted (the CP1252
fallback warning is transport, not correction) or line breaks were normalised
(representation, not correction). Do not infer the event from the counters:
removals and tab replacements modify the document with both counters at zero.

Note that without `--strict` an unmappable character becomes `?`, exit 0 and
`modified` true: the message is forwarded and the event shows only in the log
line and `unmappable` count. Pass `--strict` if the site would rather hold the
message until the override table covers the character.

### Text mode (shell callers)

```bash
auFTsanitiser report.txt --strict -o sanitised.txt
case $? in
  0) echo "sanitised" ;;
  1) echo "conversion failed: see stderr (codepoints only), nothing written" >&2; exit 1 ;;
  2) echo "usage or input error: see stderr, nothing written" >&2; exit 2 ;;
esac
```

Exit codes: `0` success (including runs with substitutions or warnings); `1`
conversion failure (undecodable input, an unmappable character under
`--strict`, a conformance-gate or internal error); `2` usage or input error. On
exit 1 or 2 no payload is emitted and no partial `-o` file is left.

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
exit 2). See `substitution-table-format.md` for the full format.

## Determinism and PHI

- **Deterministic**: identical input and flags produce byte-identical output on
  every platform (darwin/linux/windows), asserted by the `just parity` golden
  suite across the `just dist` matrix.
- **No PHI in diagnostics**: warnings and errors carry codepoints and counts
  only, never report text. Safe to write to channel logs.
- **No network calls.** The binary is fully self-contained (`CGO_ENABLED=0`
  static build).
