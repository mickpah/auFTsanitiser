# TGA software regulation: compliance assessment

Status: assessment by the maintainers, dated 2026-10-11, against the TGA guidance
current on that date. It is not legal advice. The Therapeutic Goods Administration
(TGA) places the decision with the manufacturer: "it is your responsibility to check
whether your software or digital product is excluded from our regulation before you
release it for supply in Australia". Revisit this page when the conditions in section
6 change.

## 1. Conclusion

auFTsanitiser is not a medical device under section 41BD of the Therapeutic Goods
Act 1989, because its intended purpose is message conformance, not diagnosis,
monitoring, treatment or any other purpose in that definition. If that view were ever
contested, the tool meets the conditions of excluded-goods item 14K (middleware) and
item 14F (communications software) in Schedule 1 of the Therapeutic Goods (Excluded
Goods) Determination 2018. Excluded goods are not regulated by the TGA, must not be
entered in the Australian Register of Therapeutic Goods (ARTG), and need no
notification. No TGA action is required before supply.

The conclusion rests on the intended purpose staying as narrow as it is today. Section
6 lists the changes that would reopen it.

## 2. What the tool is, as its manufacturer intends it

auFTsanitiser replaces characters outside the set permitted by the Australian
localisation of HL7 v2.4 (printable ASCII plus line breaks) with ASCII renderings of
the same character, so that a report authored elsewhere can be carried in an HL7
message that Australian receivers accept. It is invoked by an interface engine (Mirth
Connect) between the reporting system and the receiving system. It:

- does not read, interpret or score the clinical content of the report;
- does not add, remove or reorder clinical statements, values or findings; each
  substitution is a fixed, published, character-for-character rendering
  (`prd-auFTsanitiser.md` section 7 lists every one);
- makes no claim about any disease, condition, ailment or defect;
- controls no medical device and drives no analyser or modality;
- performs no analysis, computation or logic about a patient; the only arithmetic
  is none (exponent and fraction renderings are typographic, for example
  `10¹²` to `10E12`, `½` to `1/2`);
- is deterministic: the same input and flags give byte-identical output on every
  platform, so a receiver can reverse-read every substitution from the published table.

The clinical content remains the report author's; the tool's job ends at the
character set.

## 3. The statutory test

Section 41BD defines a medical device by the purpose its supplier intends: use for
human beings for diagnosis, prevention, monitoring, prediction, prognosis, treatment
or alleviation of disease, injury or disability, investigation or modification of
anatomy or a physiological process, control of conception, and similar purposes.
Software is a medical device when it fits that definition "unless otherwise excluded"
(TGA, Understanding regulation of software-based medical devices, updated 14 August
2025). The TGA's guidance adds that where there is no medical purpose or claim and
the product does not meet the definition, "it is unlikely to be a medical device and
is not regulated by us".

Applied here: the tool's intended purpose is interoperability. It is used on reports
that will be read for diagnosis, but it does not itself diagnose, monitor or
recommend, and it makes no claim to. The TGA's communications-software guidance makes
the same distinction: software is excluded "even if a health professional using it
might be diagnosing or treating a patient", as long as "the software itself does not
perform that activity".

## 4. The exclusions that apply

The Determination lists 15 excluded software categories. The TGA notes that where
software has several functions, every function must meet the exclusion criteria. This
tool has one function.

### 4.1 Item 14K, middleware (primary fit)

Item 14K excludes "software that is middleware and is: (a) intended by its
manufacturer to connect or interface applications to an underlying operating system or
another application, including by communicating or transmitting information; and
(b) not intended by its manufacturer to: (i) control medical devices; or (ii) perform
analysis, computation or logic that relates to the intended purpose of a medical
device; or (iii) be used for the purpose of diagnosis, screening, prevention,
monitoring, prediction, prognosis, alleviation, treatment, or making a recommendation
or decision about the treatment, of a disease, condition, ailment or defect".

The TGA's four questions for 14K, answered for this tool:

| TGA question | Answer | Basis |
|---|---|---|
| Does the software control a medical device? | No | It reads text on stdin and writes text on stdout; it has no device interface and makes no network calls. |
| Does it perform analysis, computation or logic that relates to the intended purpose of a medical device? | No | Fixed character substitution only. Nothing is computed from the report's meaning. |
| Does it make claims about a disease, condition, defect or ailment? | No | Its only claim is HL7 AU character-set conformance. |
| Is it middleware that only enables communications between software layers or components? | Yes | It sits between the reporting system and the HL7 receiver, inside the interface engine, and passes the report along in a form the next hop accepts. |

The TGA's own excluded example is "hospital integration middleware" that
"communicates and integrates between user-facing software and the hospital's
mainframe" and "does nothing else, for example it does not analyse medical test
results". Its not-excluded examples are pathology middleware that drives an IVD
analyser, and middleware that analyses biopsy samples to produce a diagnostic output.
auFTsanitiser is of the first kind.

### 4.2 Item 14F, communications software (secondary fit)

Item 14F excludes software "intended by its manufacturer to enable communications,
including the transmission of patient information, for the purposes of supporting the
delivery of health services" and not intended for any diagnostic or treatment purpose.
The TGA's excluded example is an email or SMS platform that sends written results.
The sanitiser serves the same transmission purpose one step earlier in the pipeline.

### 4.3 Item 14O, laboratory information systems (the boundary to watch)

Item 14O excludes a laboratory information management system that is not intended to
"manipulate information or data to change, or generate new, diagnostic outputs (other
than automating simple calculations or generating report comments)". The TGA glossary
reads "change, or generate new, diagnostic outputs" as producing "additional
information that was not previously available, which effectively influences decisions
about the diagnosis of a patient".

auFTsanitiser is not a laboratory information system, so 14O does not apply
directly, but its wording is the clearest statement of where this tool's safety
boundary lies: a substitution must never change a diagnostic output. Rendering `µg`
as `ug` or `≤` as `<=` does not; collapsing `10¹²` into `1012` would have, which is
why that defect was fixed before this assessment was written (see `prd-auFTsanitiser.md`,
superscript rows, and the AU localisation section 4.10 review).

## 5. Obligations that remain, outside the TGA

Being outside TGA regulation does not make the tool free of obligations. The ones
that apply, and how the project meets them:

- **Clinical governance of the substitution table.** The tool changes the bytes of a
  clinical report. The practice supplying the reports owns the clinical acceptability
  of every rendering under its own governance (for radiology, RANZCR standards; for
  pathology, NPAAC and NATA requirements). The project supports that with a single
  published table, a site override mechanism, test-first changes, and a recorded
  rationale for contested entries (for example the `ß` to `ss` default with the beta
  override, PRD question T044). Keep table changes reviewed by a clinician of the
  supplying practice and record the sign-off in the commit or the site's change log.
- **Privacy.** Report text is health information under the Privacy Act 1988
  (Australian Privacy Principles) and state health records legislation. The tool's
  design rule that diagnostics carry codepoints and counts only, never report text,
  is enforced by test (`internal/cli/phileak_test.go`, canary and fuzz tests) and by
  the repository's PHI scan. The interface engine's logging must keep the same rule;
  `mirth-integration.md` states what may be logged.
- **HL7 AU conformance.** The tool exists to satisfy the Australian Diagnostics and
  Referral Messaging localisation of HL7 v2.4. Conformance is enforced at the single
  output point (`internal/conform`) and by the golden, corpus and fuzz suites.
- **Incident handling.** A substitution that altered clinical meaning would be a
  clinical incident under the organisation's incident system, not a TGA adverse
  event. The deterministic design and published table make such an incident
  reproducible and bounded: the affected codepoint identifies every affected message.
- **Security.** The TGA's cyber security guidance binds medical devices only, but the
  tool follows its substance: no network access, a static binary with no
  dependencies, a bounded input size, hard non-zero exits on every failure, and
  checksummed release artefacts.
- **Identification.** Essential Principle 13B (version and build identification)
  does not apply to excluded goods. The tool nevertheless reports a single-sourced
  version with `--version`, and the deployment artefacts are checksummed, which is
  what 13B asks of devices.

## 6. When this assessment must be redone

Any of the following changes the intended purpose and reopens the section 41BD
question. Do not ship the change until the assessment is updated.

1. Adding any logic that reads the meaning of the report: flagging abnormal values,
   extracting findings, summarising, classifying, translating, or correcting clinical
   terms.
2. Adding a substitution that changes a value rather than its glyph, or any arithmetic
   on report content.
3. Any claim, in documentation or marketing, that the tool improves diagnostic
   accuracy, safety of interpretation, or clinical outcomes. Claims must stay at
   "conforms the character set for AU HL7 v2.4 messaging".
4. Using the tool to drive or control a device (an analyser, a modality, a worklist).
5. A change to the Determination. The TGA republished all 15 exclusion guidance pages
   on 16 March 2026 and has a review of the digital mental health exclusion with
   consultation planned for 2026; the middleware and communications items were not
   flagged, but check the current Schedule 1 before each new site deployment.
6. Supply to a new site or as a product. The manufacturer under the Act is the person
   under whose name the software is supplied; a distributor who rebrands or bundles
   it takes on this assessment.

## 7. Evidence to keep on file

The TGA asks nothing of excluded goods, but an auditor or a receiving site may ask why
the tool is outside the ARTG. Keep together:

- the intended-purpose statement (README, "Intended purpose");
- this assessment, with the guidance dates below;
- the substitution table with its rationale (`prd-auFTsanitiser.md` section 7) and
  the override format contract;
- the test evidence for the character-set invariant, determinism and PHI hygiene
  (`just check`, the corpus and fuzz suites);
- the change log of table and override changes with clinical sign-off.

## 8. Sources consulted

- Therapeutic Goods Act 1989, section 41BD (definition of medical device).
- Therapeutic Goods (Excluded Goods) Determination 2018, Schedule 1 items 14A to 14O,
  as inserted by the Therapeutic Goods (Excluded Goods) Amendment (Software-based
  Products) Determination 2021, F2021L00047, registered 15 January 2021 (items 14F,
  14K and 14O quoted above from the authorised text).
- TGA, Software-based medical device exclusions (index of the 15 categories), last
  updated 16 March 2026.
- TGA, Understanding the middleware for health systems exclusion, 16 March 2026.
- TGA, Understanding the communications software exclusion, 16 March 2026.
- TGA, Understanding the laboratory information management software exclusion,
  16 March 2026.
- TGA, Understanding the clinical workflow management software exclusion,
  16 March 2026 (for the "every function must meet the criteria" rule).
- TGA, Understanding regulation of software-based medical devices, published 21 June
  2022, last updated 14 August 2025.
- TGA, Is my software regulated? (decision flowchart).
- HL7 Australia, Australian Diagnostics and Referral Messaging, Localisation of HL7
  Version 2.4 (HL7AUSD-STD-OO-ADRM-2021.1), for the conformance purpose the tool serves.
