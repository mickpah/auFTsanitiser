// Package cli implements the auFTsanitiser command-line contract
// (docs/cli-contract.md, FR-030..FR-033).
package cli

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"auFTsanitiser/internal/decode"
	"auFTsanitiser/internal/hl7escape"
	"auFTsanitiser/internal/sanitise"
	"auFTsanitiser/internal/table"
)

const usageText = `usage: auFTsanitiser <input.txt | -> [options]

Sanitise a plain-text clinical report for Australian HL7 v2.4 messaging:
every character outside the AU-permitted set (printable ASCII) is replaced
with a clinically equivalent ASCII substitution.

options:
  -o <file>          write output to <file> (atomic) instead of stdout
  --format <text|json>  output mode (default text on stdout; json for Mirth)
  --charset <ascii|latin1>  permitted output set (default ascii)
  --output-encoding <utf8|latin1>  output byte encoding (default utf8)
  --line-break <lf|cr>  output line-break representation (default lf)
  --hl7-escape       escape HL7 delimiters and emit \.br\ line breaks
  --strict           fail (exit 1) on unmappable characters instead of '?'
  --table <file>     site-specific substitution overrides
  --base64           base64-encode the output payload
  --base64-input     base64-decode the input first
  --help             show this help
  --version          show version

exit codes: 0 success; 1 conversion failure; 2 usage/input error
`

// Run executes the CLI and returns the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args)
	if err != nil {
		// Usage errors are text-mode regardless of --format json
		// (contracts/cli-contract.md, usage-error boundary).
		fmt.Fprintf(stderr, "auFTsanitiser: %v\n%s", err, usageText)
		return 2
	}
	if opts.ShowHelp {
		fmt.Fprint(stdout, usageText)
		return 0
	}
	if opts.ShowVersion {
		fmt.Fprintln(stdout, "auFTsanitiser "+Version)
		return 0
	}

	// Site override table (FR-011): validated at load; failures are usage
	// errors before any input is read.
	var overrides map[rune]string
	if opts.TablePath != "" {
		overrides, err = table.LoadOverrides(opts.TablePath)
		if err != nil {
			return failWithCode(opts, err.Error(), sanitise.Result{}, stdout, stderr, 2)
		}
	}

	// Input acquisition (FR-001).
	var data []byte
	if opts.InputPath == "-" {
		data, err = io.ReadAll(stdin)
		if err != nil {
			return failWithCode(opts, "cannot read stdin", sanitise.Result{}, stdout, stderr, 2)
		}
	} else {
		data, err = os.ReadFile(opts.InputPath)
		if err != nil {
			return failWithCode(opts, "cannot read input file", sanitise.Result{}, stdout, stderr, 2)
		}
	}

	// FR-004: base64 unwrap before decoding; invalid base64 is a usage error.
	if opts.Base64Input {
		decoded, berr := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if berr != nil {
			return failWithCode(opts, "invalid base64 input", sanitise.Result{}, stdout, stderr, 2)
		}
		data = decoded
	}

	// Decode stage (FR-002/FR-003): UTF-8/UTF-16/CP1252 detection.
	dec, derr := decode.Decode(data)
	if derr != nil {
		// Undecodable input is a conversion failure (FR-032).
		return fail(opts, derr.Error(), sanitise.Result{}, stdout, stderr)
	}

	res, serr := sanitise.Sanitise(dec.Text, sanitise.Options{
		Charset:   opts.Charset,
		Strict:    opts.Strict,
		Overrides: overrides,
	})
	if serr != nil {
		// Strict-mode conversion failure: exit 1, no output (FR-013).
		return fail(opts, serr.Error(), res, stdout, stderr)
	}
	// Decode-stage warnings (e.g. CP1252 fallback) precede pass warnings.
	res.Warnings = append(dec.Warnings, res.Warnings...)

	// FR-020: field-embedding escape runs after sanitisation; without the
	// flag, reserved delimiters pass through untouched (FR-021).
	if opts.HL7Escape {
		res.Text = hl7escape.Escape(res.Text)
	}

	return emit(opts, res, stdout, stderr)
}
