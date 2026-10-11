package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"auFTsanitiser/internal/conform"
	"auFTsanitiser/internal/sanitise"
)

// envelope is the fixed-field JSON contract (FR-031). Field order is the
// serialisation order; warnings is always an array, never null.
type envelope struct {
	Text        string   `json:"text"`
	Encoding    string   `json:"encoding"`
	Error       string   `json:"error"`
	Warnings    []string `json:"warnings"`
	Substituted int      `json:"substituted"`
	Unmappable  int      `json:"unmappable"`
	Modified    bool     `json:"modified"`
}

// emit is the single output-emission point (plan Structure Decision): every
// payload passes conform.Check before any byte leaves the process.
func emit(o Options, res sanitise.Result, stdout, stderr io.Writer) int {
	payload := encodePayload(o, res.Text)

	if err := conform.Check(payload, conform.Params{
		Charset:        o.Charset,
		LineBreak:      o.LineBreak,
		HL7Escape:      o.HL7Escape,
		OutputEncoding: o.OutputEncoding,
	}); err != nil {
		// Defect guard (Principle I): abort, no partial output.
		return fail(o, fmt.Sprintf("internal conformance failure: %v", err), res, stdout, stderr)
	}

	text := payload
	encoding := "utf8"
	if o.Base64Output {
		encoding = "base64"
		text = []byte(base64.StdEncoding.EncodeToString(payload))
	}

	var outBytes []byte
	if o.Format == "json" {
		env := envelope{
			Text:        string(text),
			Encoding:    encoding,
			Warnings:    warningStrings(res),
			Substituted: res.Substituted,
			Unmappable:  res.Unmappable,
			Modified:    res.Modified(),
		}
		j, err := json.Marshal(env)
		if err != nil {
			fmt.Fprintln(stderr, "auFTsanitiser: internal error encoding JSON")
			return 1
		}
		outBytes = append(j, '\n')
	} else {
		outBytes = text
		for _, w := range res.Warnings {
			fmt.Fprintln(stderr, "auFTsanitiser: warning: "+w.String())
		}
	}

	if o.OutputPath == "" {
		if _, err := stdout.Write(outBytes); err != nil {
			fmt.Fprintln(stderr, "auFTsanitiser: write error")
			return 2
		}
		return 0
	}
	if err := atomicWrite(o.OutputPath, outBytes); err != nil {
		return failWithCode(o, "cannot write output file", res, stdout, stderr, 2)
	}
	return 0
}

// fail reports a post-parse conversion failure (exit 1) per the selected
// format: text mode → stderr diagnostic; json mode → envelope with error
// set, stderr empty (contracts/cli-contract.md).
func fail(o Options, msg string, res sanitise.Result, stdout, stderr io.Writer) int {
	return failWithCode(o, msg, res, stdout, stderr, 1)
}

// failWithCode is fail with an explicit exit code — post-parse usage errors
// (unreadable input, invalid base64, unwritable output) are exit 2 but still
// honour the selected output format.
func failWithCode(o Options, msg string, res sanitise.Result, stdout, stderr io.Writer, code int) int {
	if o.Format == "json" {
		env := envelope{
			Encoding:    "utf8",
			Error:       msg,
			Warnings:    warningStrings(res),
			Substituted: res.Substituted,
			Unmappable:  res.Unmappable,
			Modified:    res.Modified(),
		}
		j, _ := json.Marshal(env)
		stdout.Write(append(j, '\n'))
	} else {
		fmt.Fprintln(stderr, "auFTsanitiser: error: "+msg)
	}
	return code
}

func warningStrings(res sanitise.Result) []string {
	ws := make([]string, 0, len(res.Warnings))
	for _, w := range res.Warnings {
		ws = append(ws, w.String())
	}
	return ws
}

// encodePayload applies the output line-break representation and byte
// encoding (FR-010, FR-016). Internal text always carries LF; in hl7-escape
// mode the escaper has already consumed every line break.
func encodePayload(o Options, text string) []byte {
	if o.LineBreak == "cr" && !o.HL7Escape {
		text = strings.ReplaceAll(text, "\n", "\r")
	}
	if o.OutputEncoding == "latin1" {
		b := make([]byte, 0, len(text))
		for _, r := range text {
			if r <= 0xFF {
				b = append(b, byte(r))
			} else {
				b = append(b, '?') // unreachable post-sanitise; defensive
			}
		}
		return b
	}
	return []byte(text)
}

// atomicWrite writes to a temp file in the destination directory and
// renames into place — no partial file on any failure (FR-030). The result
// keeps the mode of the file it replaces, else 0644 (CreateTemp's 0600
// would hide it from readers running as another user).
func atomicWrite(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".auFTsanitiser-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
