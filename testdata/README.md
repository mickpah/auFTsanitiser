# Golden fixtures (T002)

Byte-compared fixture pairs for the golden-corpus harness
(internal/sanitise/golden_test.go):

- `<category>.input.txt` — UTF-8 input exercising one PRD section 7 category
- `<category>.golden.txt` — expected default-mode output (charset ascii, LF)

Rules:

- Every fixture is SYNTHETIC. No real patient data may ever be committed
  (constitution Principle V); the real-report corpus lives outside git in
  /Sample_reports/ and is exercised by a skip-when-absent harness.
  `scripts/anonymise-hl7.py` turns such a corpus into synthetic fixtures and
  `scripts/scan-for-phi.py` (run by `just phi-scan`, part of `just check`)
  guards the tracked fixtures against PHI-shaped identifiers and names.
- Inputs are UTF-8 without BOM unless the category tests BOM handling.
- Golden files contain only bytes 0x20-0x7E and LF; the harness also runs
  conform.Check over every golden file, so a nonconformant golden fails CI.
- Categories mirror the PRD table groups: whitespace, punctuation,
  signs-units, greek, commercial, latin, linebreaks, mixed-report.
