package cli

import (
	"errors"
	"fmt"
)

// Options is the validated CLI invocation (data-model.md). Flags are
// accepted in any order (FR-033); exactly one positional input argument.
type Options struct {
	InputPath      string
	OutputPath     string
	Format         string // "text" | "json"
	Charset        string // "ascii" | "latin1"
	OutputEncoding string // "utf8" | "latin1"
	LineBreak      string // "lf" | "cr"
	HL7Escape      bool
	Strict         bool
	Base64Output   bool
	Base64Input    bool
	TablePath      string
	ShowHelp       bool
	ShowVersion    bool
}

func parseOptions(args []string) (Options, error) {
	o := Options{
		Format:         "text",
		Charset:        "ascii",
		OutputEncoding: "utf8",
		LineBreak:      "lf",
	}
	valueFlags := map[string]*string{
		"-o":                &o.OutputPath,
		"--format":          &o.Format,
		"--charset":         &o.Charset,
		"--output-encoding": &o.OutputEncoding,
		"--line-break":      &o.LineBreak,
		"--table":           &o.TablePath,
	}
	boolFlags := map[string]*bool{
		"--hl7-escape":   &o.HL7Escape,
		"--strict":       &o.Strict,
		"--base64":       &o.Base64Output,
		"--base64-input": &o.Base64Input,
		"--help":         &o.ShowHelp,
		"-h":             &o.ShowHelp,
		"--version":      &o.ShowVersion,
	}
	seenInput := false
	for i := 1; i < len(args); i++ {
		a := args[i]
		if p, ok := boolFlags[a]; ok {
			*p = true
			continue
		}
		if p, ok := valueFlags[a]; ok {
			if i+1 >= len(args) {
				return o, fmt.Errorf("flag %s requires a value", a)
			}
			i++
			*p = args[i]
			continue
		}
		if len(a) > 1 && a[0] == '-' && a != "-" {
			return o, fmt.Errorf("unknown flag %s", a)
		}
		if seenInput {
			return o, errors.New("multiple input arguments")
		}
		o.InputPath = a
		seenInput = true
	}
	if o.ShowHelp || o.ShowVersion {
		return o, nil
	}
	if !seenInput {
		return o, errors.New("missing input (file path or - for stdin)")
	}
	if o.Format != "text" && o.Format != "json" {
		return o, errors.New("invalid --format (text|json)")
	}
	if o.Charset != "ascii" && o.Charset != "latin1" {
		return o, errors.New("invalid --charset (ascii|latin1)")
	}
	if o.OutputEncoding != "utf8" && o.OutputEncoding != "latin1" {
		return o, errors.New("invalid --output-encoding (utf8|latin1)")
	}
	// Latin-1 bytes are not UTF-8; encoding/json would rewrite them as U+FFFD.
	if o.Format == "json" && o.OutputEncoding == "latin1" && !o.Base64Output {
		return o, errors.New("--output-encoding latin1 with --format json requires --base64")
	}
	if o.LineBreak != "lf" && o.LineBreak != "cr" {
		return o, errors.New("invalid --line-break (lf|cr)")
	}
	return o, nil
}
