# Use of AI in this project

AI assistants were used in developing this project: for reviewing code, tests and
documents, for reviewing changes, and for research. 

What that does and does not mean:

- Every change was read, tested and committed by the maintainer, who is responsible
  for it. The test suite, the PHI-leak guards and the conformance gate apply to all
  code regardless of how it was first written.
- No patient data was ever given to an AI tool. The real-report corpus stays on the
  maintainer's machine; anything an assistant saw was synthetic or anonymised.
- Configuration and working files for AI assistants and coding agents are not part of
  the repository and are ignored by git.

Questions about this go in an issue; see `SUPPORT.md`.
