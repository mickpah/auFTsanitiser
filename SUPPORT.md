# Support and Release Policy

*Rigidly defined areas of doubt and uncertainty.*

This project is maintained by one person, in their spare time, for as long as it stays
interesting. This document says exactly what you can rely on and exactly what you can't,
so nobody has to guess.

## Areas of certainty

These are commitments. If they change, this file changes first.

- **Licence.** The licence in `LICENSE` applies to every release, forever. You can fork,
  vendor, or build on any version regardless of what happens to this repository.
- **Releases are source only.** A release is a signed-off git tag; no binaries, packages
  or checksums of binaries are published, here or anywhere else. Build from the tag
  yourself (`just build`, `just dist`, `just deb`). A tag is never moved or deleted.
- **No binaries, by design.** This tool rewrites the bytes of clinical reports. The
  organisation that deploys it must build it from a tag it has read, validate it against
  its own feeds, and review the substitution table under its own clinical governance.
  Publishing a binary would invite skipping those steps and would make this repository
  the supplier of an artefact it cannot validate for your site. If you find a binary
  claiming to be this project, it is not from here.
- **Semantic Versioning.** `MAJOR.MINOR.PATCH`. Breaking changes only in a MAJOR bump
  (or in `0.x`, only in a MINOR bump). Breaking changes are listed in `CHANGELOG.md`.
- **Security reports get a response.** Report a vulnerability privately through GitHub's
  private vulnerability reporting (the repository's Security tab), not in a public issue. A
  report will be acknowledged within 14 days. Whether a fix follows, and how fast, is covered below.
  `SECURITY.md` says what to include, what is in scope and how disclosure works.
- **Status is published.** The "Current status" section of this file is kept accurate. If
  this project is abandoned, that will be stated here rather than left to be inferred from
  silence.

## Areas of uncertainty

These are explicitly **not** commitments.

- **Continued development.** There is no promised roadmap and no promise of future releases.
  The project may be paused or stopped at any time, for any reason, including none.
- **Response times.** Issues may be answered in hours, months, or never.
- **Feature requests.** Welcome to file; not owed a reply. Closed as "not planned" is not
  a judgement on the idea.
- **Bug fixes.** Confirmed bugs in the latest release *will probably* be fixed. Bugs in
  anything older will not be.
- **Backports.** None. Fixes land on `main` and ship in the next release only.
- **Platform support.** The test suite (`just check`) is run on macOS and Linux; Linux
  amd64 is the deployment target. The Windows build compiles and is covered by the
  golden parity suite but is not otherwise exercised. Other platforms may work; reports
  from them are welcome, fixes are not guaranteed.
- **Toolchain floor.** There are no runtime dependencies and no Go module dependencies
  (standard library only). The minimum Go version is the one in `go.mod`; it moves when
  convenient or when a security advisory makes it necessary.
- **Clinical suitability.** No claim is made that any substitution is right for your
  reports, your receivers or your patients. The intended purpose is character-set
  conformance for Australian HL7 v2.4 messaging and nothing more; see
  `docs/tga-compliance.md` for what that means and what would change it.

## Supported versions

| Version              | Status      | Bug fixes | Security fixes   |
|----------------------|-------------|-----------|------------------|
| Latest tagged release (built from source) | Supported | Likely | Yes, best effort |
| Everything else      | Unsupported | No        | No               |

"Best effort" means: the maintainer intends to fix it and will try, but is not on call.

## Current status

**Active** — the maintainer is using this project and expects to keep working on it.

Other values this field may take:

- **Maintenance** — no new features; security fixes and critical bugs only.
- **Dormant** — no activity planned. Security reports still read. Forks encouraged.
- **Archived** — repository is read-only. Nothing further will happen here.

If this section has not been touched in over 12 months, assume **Dormant** regardless of
what it says.

## Issues and pull requests

- Search existing issues first.
- Bug reports need: version (`auFTsanitiser --version`), platform, minimal reproduction,
  expected vs. actual behaviour. Reports without a reproduction may be closed without
  investigation.
- **Never put report text or any patient information in an issue.** Give the codepoints
  (the tool's own diagnostics are safe to paste: they carry codepoints and counts only)
  and a synthetic reproduction. `scripts/anonymise-hl7.py` turns a real message into a
  synthetic one if you need a whole-message example. Issues containing patient
  information are deleted, not edited.
- **Contributions are not accepted at this stage of the project.** Pull requests will be
  closed unread, however good they are. This is about the project's stage and my available time, not your work:
  the design is still moving under one pair of hands, and reviewing outside changes costs
  more than it returns right now. If that changes, `CONTRIBUTING.md` and this file will say
  so first.
- Issues are the way in: bug reports, substitution-table disagreements with the clinical
  reasoning stated, HL7 AU standard readings with a section cited, and fork announcements
  (below) are all welcome.

## Conduct

Discussions and issues are requested to be courteous and on topic. Abuse is removed and
the account blocked. No formal code of conduct is adopted at this stage.

## If this project goes quiet

- Fork it. The licence allows this without asking.
- If you maintain a fork that is active, open an issue titled
  "Fork: <url>" and it will be linked from `README.md` when I next look.
- If you'd like to take over this repository rather than fork it, open an issue and say so.
  No promises, but it's the only way it could happen.

## Changes to this policy

This file may change. Changes are recorded in `CHANGELOG.md`. The areas of certainty
listed above will not be weakened for any already-published release.

## Credits

The structure of this policy — a demand for rigidly defined areas of doubt and
uncertainty — is owed to the Amalgamated Union of Philosophers, Sages, Luminaries and
Other Thinking Persons, as represented by Majikthise and Vroomfondel, and to
Douglas Adams, who wrote them into *The Hitchhiker's Guide to the Galaxy*.

Neither the Union nor Mr Adams has reviewed, endorsed, or been consulted about this
document. Deep Thought was not asked either, on the grounds that the answer would take
seven and a half million years and probably be 42.
