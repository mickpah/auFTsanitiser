package cli

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

type env struct {
	Text        string   `json:"text"`
	Encoding    string   `json:"encoding"`
	Error       string   `json:"error"`
	Warnings    []string `json:"warnings"`
	Substituted int      `json:"substituted"`
	Unmappable  int      `json:"unmappable"`
	Modified    bool     `json:"modified"`
}

func decodeEnvelope(t *testing.T, out string) env {
	t.Helper()
	var e env
	if err := json.Unmarshal([]byte(out), &e); err != nil {
		t.Fatalf("stdout is not one JSON object: %v (%q)", err, out)
	}
	return e
}

func TestJSONSuccessEnvelope(t *testing.T) {
	code, out, errs := runCLI(t, "37.5°C ☃\n", "-", "--format", "json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if errs != "" {
		t.Errorf("stderr must be empty in json mode, got %q", errs)
	}
	e := decodeEnvelope(t, out)
	if e.Text != "37.5 deg C ?\n" || e.Encoding != "utf8" || e.Error != "" {
		t.Errorf("envelope %+v", e)
	}
	if e.Substituted != 1 || e.Unmappable != 1 {
		t.Errorf("counters %+v", e)
	}
	if len(e.Warnings) != 1 || !strings.Contains(e.Warnings[0], "U+2603") {
		t.Errorf("warnings %v", e.Warnings)
	}
}

func TestJSONFieldOrderIsFixed(t *testing.T) {
	_, out, _ := runCLI(t, "x\n", "-", "--format", "json")
	idx := func(k string) int { return strings.Index(out, `"`+k+`"`) }
	order := []string{"text", "encoding", "error", "warnings", "substituted", "unmappable", "modified"}
	for i := 1; i < len(order); i++ {
		if idx(order[i-1]) < 0 || idx(order[i-1]) > idx(order[i]) {
			t.Fatalf("field order broken in %q", out)
		}
	}
}

func TestJSONEmptyWarningsIsArray(t *testing.T) {
	_, out, _ := runCLI(t, "clean\n", "-", "--format", "json")
	if !strings.Contains(out, `"warnings":[]`) {
		t.Errorf("warnings must be [], got %q", out)
	}
}

func TestJSONBase64(t *testing.T) {
	code, out, _ := runCLI(t, "µg\n", "-", "--format", "json", "--base64")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	e := decodeEnvelope(t, out)
	if e.Encoding != "base64" {
		t.Errorf("encoding %q", e.Encoding)
	}
	raw, err := base64.StdEncoding.DecodeString(e.Text)
	if err != nil || string(raw) != "ug\n" {
		t.Errorf("base64 payload %q -> %q, %v", e.Text, raw, err)
	}
}

func TestTextModeBase64(t *testing.T) {
	code, out, _ := runCLI(t, "µg\n", "-", "--base64")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(out))
	if err != nil || string(raw) != "ug\n" {
		t.Errorf("base64 stdout %q, %v", out, err)
	}
}

// Post-parse failures in json mode: envelope with error set, empty stderr,
// exit code preserved (contracts/cli-contract.md).
func TestJSONStrictFailureEnvelope(t *testing.T) {
	code, out, errs := runCLI(t, "snow ☃\n", "-", "--format", "json", "--strict")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if errs != "" {
		t.Errorf("stderr must be empty in json mode, got %q", errs)
	}
	e := decodeEnvelope(t, out)
	if e.Text != "" || e.Error == "" {
		t.Errorf("failure envelope %+v", e)
	}
	if !strings.Contains(e.Error, "U+2603") {
		t.Errorf("error should name codepoint: %q", e.Error)
	}
	if strings.Contains(e.Error, "snow") {
		t.Errorf("error leaks text: %q", e.Error)
	}
}

// TestJSONModifiedFlag pins the correction-event flag for the calling Mirth
// channel (FR-031): true exactly when a character-level correction was
// applied. Counters alone cannot express it (removal/tab bump no counter) and
// warnings alone over-report it (encoding fallback is not a correction).
func TestJSONModifiedFlag(t *testing.T) {
	cases := []struct {
		name string
		in   string
		args []string
		want bool
	}{
		{"clean", "clean text\n", nil, false},
		{"substitution", "µg\n", nil, true},
		{"unmappable", "snow ☃\n", nil, true},
		{"removal-only-no-counter", "so\u00adft\n", nil, true},
		{"tab-only-no-counter", "a\tb\n", nil, true},
		{"linebreak-normalisation-is-not-a-correction", "a\r\nb\r\n", nil, false},
		{"encoding-fallback-passthrough-is-not-a-correction", "caf\xe9\n", []string{"--charset", "latin1"}, false},
	}
	for _, c := range cases {
		args := append([]string{"-", "--format", "json"}, c.args...)
		code, out, _ := runCLI(t, c.in, args...)
		if code != 0 {
			t.Fatalf("%s: exit %d", c.name, code)
		}
		e := decodeEnvelope(t, out)
		if e.Modified != c.want {
			t.Errorf("%s: modified %v, want %v (%+v)", c.name, e.Modified, c.want, e)
		}
	}

	// The removal-only case must report modified with both counters zero —
	// this is the gap the flag exists to close.
	_, out, _ := runCLI(t, "so\u00adft\n", "-", "--format", "json")
	e := decodeEnvelope(t, out)
	if e.Substituted != 0 || e.Unmappable != 0 || !e.Modified {
		t.Errorf("removal-only: %+v, want counters 0 and modified true", e)
	}
}

func TestJSONInvalidBase64StillTextUsageError(t *testing.T) {
	// Invalid base64 is a usage error (exit 2). It occurs post-parse, so in
	// json mode it produces the envelope? No — FR-004 classes it exit 2 and
	// the contract routes POST-PARSE failures through the selected format.
	// Exit code must be 2 either way; envelope carries the error.
	code, out, errs := runCLI(t, "!!!", "-", "--format", "json", "--base64-input")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if out == "" && errs == "" {
		t.Error("expected a diagnostic somewhere")
	}
}
