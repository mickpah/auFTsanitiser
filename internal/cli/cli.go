// Package cli implements the auFTsanitiser command-line contract
// (docs/cli-contract.md, FR-030..FR-033).
package cli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
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

// maxInput bounds stdin and file input (64 MiB): a runaway stream fails with
// exit 2 instead of exhausting memory. Real reports are kilobytes.
const maxInput = 64 << 20

// Run executes the CLI and returns the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
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
	// A crash is a hard exit 1 in the selected format, never a stack trace
	// with the usage-error code. The message carries no input (see
	// internalErrorMessage).
	defer func() {
		if r := recover(); r != nil {
			code = failWithCode(opts, internalErrorMessage(r), sanitise.Result{}, stdout, stderr, 1)
		}
	}()

	// Input acquisition (FR-001). Input is drained before the --table is
	// loaded so a caller piping a large report sees the table error, not a
	// broken pipe.
	var data []byte
	if opts.InputPath == "-" {
		data, err = readLimited(stdin)
		if errors.Is(err, errTooLarge) {
			return failWithCode(opts, "input exceeds 64 MiB", sanitise.Result{}, stdout, stderr, 2)
		}
		if err != nil {
			return failWithCode(opts, "cannot read stdin", sanitise.Result{}, stdout, stderr, 2)
		}
	} else {
		f, ferr := os.Open(opts.InputPath)
		if ferr == nil {
			data, err = readLimited(f)
			f.Close()
		}
		if errors.Is(err, errTooLarge) {
			return failWithCode(opts, "input exceeds 64 MiB", sanitise.Result{}, stdout, stderr, 2)
		}
		if ferr != nil || err != nil {
			return failWithCode(opts, "cannot read input file", sanitise.Result{}, stdout, stderr, 2)
		}
	}

	// Site override table (FR-011): validated at load; failures are usage
	// errors.
	var overrides map[rune]string
	if opts.TablePath != "" {
		overrides, err = table.LoadOverrides(opts.TablePath)
		if err != nil {
			return failWithCode(opts, err.Error(), sanitise.Result{}, stdout, stderr, 2)
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

var errTooLarge = errors.New("input too large")

// readLimited reads at most maxInput bytes; one byte more is errTooLarge.
func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxInput+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxInput {
		return nil, errTooLarge
	}
	return data, nil
}

// internalErrorMessage renders a recovered panic without input text: a Go
// runtime error names only indices and lengths, so its message is kept; any
// other value (a string, an error built from data) is reduced to its type.
func internalErrorMessage(r interface{}) string {
	if err, ok := r.(runtime.Error); ok {
		return "internal error: " + err.Error()
	}
	return fmt.Sprintf("internal error: %T", r)
}
