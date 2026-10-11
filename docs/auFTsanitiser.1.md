# AUFTSANITISER(1)

## NAME

auFTsanitiser - sanitise a plain-text clinical report for Australian HL7 v2.4
messaging

## SYNOPSIS

**auFTsanitiser** *input.txt* | **-** [*options*]

## DESCRIPTION

Replaces every character outside the AU-permitted set (printable ASCII
0x20-0x7E plus line breaks) with a clinically equivalent ASCII substitution:
`37.5°C` becomes `37.5 deg C`, `µg/L` becomes `ug/L`, `β-hCG` becomes
`beta-hCG`, smart quotes become straight quotes. The Australian localisation
of HL7 v2.4 prohibits the `\Xdddd\` hexadecimal escape sequences, so
substitution before message construction is the only conformant path.

Input encoding is detected automatically (UTF-8 with or without BOM, UTF-16
LE/BE with or without BOM, Windows-1252 fallback); identical logical text
produces byte-identical output regardless of source encoding. Substitutions
never introduce the HL7 reserved delimiters `|`, `^`, `~`, `&`, `\`. Output
is deterministic and idempotent: re-running the tool on its own output is a
no-op.

Reads the named file, or standard input when *input.txt* is **-**. The
sanitised payload goes to standard output; all diagnostics go to standard
error and carry codepoints and counts only, never report text.

## OPTIONS

Flags may appear in any order.

**-o** *file*
: Write output to *file* atomically (temp file + rename; no partial file on
failure) instead of standard output. The file keeps the mode of any file it
replaces; a new file is created 0644.

**--format** *text*|*json*
: Output mode. Default *text*. *json* emits a single JSON envelope on stdout
with nothing on stderr (see OUTPUT), for programmatic callers such as a
Mirth Connect channel.

**--charset** *ascii*|*latin1*
: Permitted output set. Default *ascii*. *latin1* additionally passes
ISO 8859-1 printables 0xA0-0xFF through unsubstituted, except codepoints
named in a **--table** override, which are substituted.

**--output-encoding** *utf8*|*latin1*
: Byte encoding of the output. Default *utf8*. With **--format** *json*,
*latin1* requires **--base64**, because Latin-1 bytes cannot be carried in
a JSON string.

**--line-break** *lf*|*cr*
: Output line-break representation. Default *lf*.

**--hl7-escape**
: Escape the five reserved delimiters (`|` `^` `~` `&` `\` to `\F\` `\S\`
`\R\` `\T\` `\E\`) and emit line breaks as `\.br\`, making the payload safe
for direct insertion into an FT/TX field. Do not combine with a channel that
escapes field content itself: one escaper, never two. With
`--charset latin1`, Latin-1 characters pass through unescaped.

**--strict**
: Fail with exit status 1 on the first unmappable character instead of
replacing it with `?`.

**--table** *file*
: Site-specific substitution overrides. One `U+XXXX<TAB>replacement` per
line; `#` comments; empty replacement means remove; later lines win.
Replacements must be printable ASCII and free of reserved delimiters; an
invalid table is a usage error. See
`docs/substitution-table-format.md`.

**--base64**
: Base64-encode the output payload (both output modes).

**--base64-input**
: Base64-decode the input before encoding detection.

**--help**, **-h**
: Print usage and exit 0.

**--version**
: Print `auFTsanitiser <version>` and exit 0.

## EXIT STATUS

**0**
: Success, including runs that made substitutions or emitted warnings.

**1**
: Conversion failure: undecodable input, an unmappable character under
**--strict**, an internal conformance-gate failure, or a recovered internal
error (reported as `internal error: ...` without a stack trace).

**2**
: Usage or input error: unknown flag, missing input, unreadable input file,
input larger than 64 MiB, unwritable output, invalid base64, invalid
**--table** file, or **--output-encoding** *latin1* with **--format** *json*
and no **--base64**.

On exit 1 or 2 no payload is produced and no partial **-o** file is left.

## OUTPUT

In json mode stdout carries exactly one object:

    {"text":"...","encoding":"utf8","error":"","warnings":[...],
     "substituted":N,"unmappable":N,"modified":bool}

`encoding` is `utf8` or `base64`. `modified` is true exactly when a
character-level correction was applied (substitution, unmappable
replacement, character removal, or tab replacement); line-break
normalisation and encoding transcoding do not set it. On failure `text` is
empty and `error` is set. A flag-parsing error is reported as text usage on
stderr with exit 2 regardless of **--format json**.

## DIAGNOSTICS

Warnings name codepoints and counts only, e.g.
`unmappable character U+2603 replaced (x3)`. Report text never appears in
any diagnostic. The tool makes no network calls.

## EXAMPLES

Sanitise a report to stdout:

    auFTsanitiser report.txt

From a Mirth channel, JSON envelope with a site override:

    auFTsanitiser - --format json --table docs/overrides/auspdi-beta.tsv

Field-embeddable payload, strict mode:

    printf 'A|B 37.5°C\n' | auFTsanitiser - --hl7-escape --strict

## SEE ALSO

`docs/mirth-integration.md` (channel-side wiring),
`docs/architecture.md` (internals),
`docs/cli-contract.md` (normative
contract; this page is descriptive).
