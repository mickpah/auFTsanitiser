# Where auFTsanitiser sits under TGA software regulation

An engineering note, written 11 October 2026 by the maintainers after reading the
TGA's current guidance. It is how we understand the rules and how they apply to what
the tool does. It is not legal advice; if your organisation needs that, get it from
someone qualified. The TGA puts the call on the manufacturer ("it is your
responsibility to check whether your software or digital product is excluded from our
regulation before you release it for supply in Australia"), so this page is us doing
that homework and showing our working. Re-read it whenever section 6 applies.

## 1. The short version

As we read the rules, auFTsanitiser is not a medical device. What it is intended to do
is make a report's characters fit the set the Australian HL7 v2.4 localisation allows.
It does not diagnose, monitor, predict, treat or recommend anything, and we make no
claims that it does. That keeps it outside the medical device definition in section
41BD of the Therapeutic Goods Act 1989.

If anyone wanted to argue the point, the tool also fits two of the categories the TGA
has specifically carved out of regulation: item 14K (middleware) and item 14F
(communications software) in Schedule 1 of the Therapeutic Goods (Excluded Goods)
Determination 2018. Excluded software is not regulated by the TGA, must not go on the
Australian Register of Therapeutic Goods (ARTG), and needs no notification. There is
nothing to lodge before using it.

All of that depends on the intended purpose staying as narrow as it is. Section 6 lists
the changes that would make us redo this assessment.

## 2. What the tool is meant to do

In the words we would use on the tin:

auFTsanitiser replaces characters that fall outside the AU-permitted set (printable
ASCII plus line breaks) with an ASCII rendering of the same character, so a report
written elsewhere can travel in an HL7 message that Australian receivers accept. It
runs inside an interface engine (Mirth Connect), between the reporting system and the
receiver. It:

- does not read, interpret or score what the report says;
- does not add, drop or reorder findings, values or statements; every substitution is a
  fixed, published, character-for-character rendering (`prd-auFTsanitiser.md`
  section 7 lists all of them);
- makes no claim about any disease, condition, ailment or defect;
- controls no device: no analyser, no modality, no worklist;
- does no analysis, computation or logic about a patient. Exponent and fraction
  renderings are typographic (`10¹²` becomes `10E12`, `½` becomes `1/2`); nothing is
  calculated from the report;
- is deterministic: same input and flags, same bytes, on every platform, so a receiver
  can read any substitution back from the published table.

The clinical content stays the author's. The tool's job stops at the character set.

## 3. The medical device definition, and why it does not bite

Section 41BD defines a medical device by what its supplier intends it for: use on
people for diagnosis, prevention, monitoring, prediction, prognosis, treatment or
alleviation of disease, injury or disability, investigating or modifying anatomy or a
physiological process, and so on. The TGA's guidance (Understanding regulation of
software-based medical devices, updated 14 August 2025) says software is a medical
device if it fits that definition "unless otherwise excluded", and that where there is
no medical purpose or claim and the product does not meet the definition, "it is
unlikely to be a medical device and is not regulated by us".

Our reading: the tool's purpose is interoperability. It handles reports that a
clinician will read to make a diagnosis, but it does not do any of that itself and does
not claim to. The TGA's communications-software guidance draws exactly this line:
software is excluded "even if a health professional using it might be diagnosing or
treating a patient", provided "the software itself does not perform that activity".

## 4. The exclusions that fit

The Determination lists 15 excluded software categories. The TGA notes that when
software does several things, every one of them must meet the exclusion. This tool
does one thing.

### 4.1 Item 14K, middleware (the best fit)

Item 14K excludes "software that is middleware and is: (a) intended by its
manufacturer to connect or interface applications to an underlying operating system or
another application, including by communicating or transmitting information; and
(b) not intended by its manufacturer to: (i) control medical devices; or (ii) perform
analysis, computation or logic that relates to the intended purpose of a medical
device; or (iii) be used for the purpose of diagnosis, screening, prevention,
monitoring, prediction, prognosis, alleviation, treatment, or making a recommendation
or decision about the treatment, of a disease, condition, ailment or defect".

The TGA turns that into four questions. Our answers:

| TGA question | Answer | Why |
|---|---|---|
| Does the software control a medical device? | No | Text in on stdin, text out on stdout. No device interface, no network. |
| Does it perform analysis, computation or logic that relates to the intended purpose of a medical device? | No | Fixed character substitution. Nothing is worked out from what the report means. |
| Does it make claims about a disease, condition, defect or ailment? | No | The only claim is HL7 AU character-set conformance. |
| Is it middleware that only enables communications between software layers or components? | Yes | It sits between the reporting system and the HL7 receiver, inside the interface engine, and passes the report along in a form the next hop accepts. |

The TGA's own excluded example is "hospital integration middleware" that
"communicates and integrates between user-facing software and the hospital's
mainframe" and "does nothing else, for example it does not analyse medical test
results". Its not-excluded examples are pathology middleware that drives an IVD
analyser, and middleware that analyses biopsy samples to produce a diagnostic result.
We are the first kind.

### 4.2 Item 14F, communications software (also fits)

Item 14F excludes software "intended by its manufacturer to enable communications,
including the transmission of patient information, for the purposes of supporting the
delivery of health services", as long as it is not intended for any diagnostic or
treatment purpose. The TGA's excluded example is an email or SMS platform that sends
written results. The sanitiser serves the same purpose one hop earlier.

### 4.3 Item 14O, laboratory information systems (the line we must not cross)

Item 14O excludes a laboratory information management system that is not intended to
"manipulate information or data to change, or generate new, diagnostic outputs (other
than automating simple calculations or generating report comments)". The TGA glossary
reads "change, or generate new, diagnostic outputs" as producing "additional
information that was not previously available, which effectively influences decisions
about the diagnosis of a patient".

auFTsanitiser is not a laboratory information system, so 14O is not our exclusion, but
it is the clearest statement anywhere of where our safety boundary sits: a substitution
must never change a diagnostic output. Rendering `µg` as `ug` or `≤` as `<=` does not.
Collapsing `10¹²` into `1012` would have, which is why that behaviour was fixed before
this note was written (see the superscript rows in `prd-auFTsanitiser.md` and the AU
localisation section 4.10 review).

## 5. What still applies, outside the TGA

Being outside TGA regulation does not mean no obligations. These are the ones we see,
and how the project meets them.

- **Clinical governance of the substitution table.** The tool changes the bytes of a
  clinical report, so the practice supplying the reports owns whether each rendering is
  clinically acceptable, under its own governance (RANZCR standards for radiology,
  NPAAC and NATA requirements for pathology). The project supports that with one
  published table, a site override mechanism, test-first changes, and a recorded reason
  for contested entries (for example `ß` to `ss` by default with the beta override,
  PRD question T044). Table changes should be reviewed by a clinician of the supplying
  practice, with the sign-off recorded in the commit or the site's change log.
- **Privacy.** Report text is health information under the Privacy Act 1988 and the
  state health records Acts. The tool's rule that diagnostics carry codepoints and
  counts only, never report text, is enforced by tests (`internal/cli/phileak_test.go`,
  the canary and fuzz tests) and by the repository's PHI scan. The interface engine's
  logging has to keep the same rule; `mirth-integration.md` says what may be logged.
- **HL7 AU conformance.** It is the reason the tool exists. Conformance is enforced at
  the single output point (`internal/conform`) and by the golden, corpus and fuzz
  suites.
- **Incidents.** A substitution that changed clinical meaning would be a clinical
  incident under the organisation's own system, not a TGA adverse event. Because the
  tool is deterministic and the table is published, such an incident is reproducible
  and bounded: the codepoint involved identifies every affected message.
- **Security.** The TGA's cyber security guidance binds medical devices only, but the
  tool follows its substance anyway: no network access, a static binary with no
  dependencies, a bounded input size, hard non-zero exits on every failure, and
  checksummed artefacts.
- **Identification.** Essential Principle 13B (version and build identification) does
  not apply to excluded goods. The tool still reports a single-sourced version with
  `--version`, which is what 13B asks of devices.

## 6. When this note has to be redone

Any of these changes the intended purpose and reopens the section 41BD question. Do
not ship the change until this page has been revisited.

1. Adding any logic that reads the meaning of the report: flagging abnormal values,
   pulling out findings, summarising, classifying, translating, or "correcting"
   clinical terms.
2. Adding a substitution that changes a value rather than its glyph, or doing any
   arithmetic on report content.
3. Any claim, in docs or marketing, that the tool improves diagnostic accuracy, safety
   of interpretation, or clinical outcomes. The claim stays at "conforms the character
   set for AU HL7 v2.4 messaging".
4. Using the tool to drive or control a device.
5. A change to the Determination. The TGA republished all 15 exclusion guidance pages
   on 16 March 2026 and has a review of the digital mental health exclusion running,
   with consultation planned for 2026. The middleware and communications items were
   not flagged, but check the current Schedule 1 before each new site deployment.
6. Supplying it to a new site, or as a product. Under the Act the manufacturer is
   whoever's name the software is supplied under. A distributor who rebrands or bundles
   it takes on this assessment.

## 7. What to keep on file

The TGA asks nothing of excluded goods, but an auditor or a receiving site may ask why
the tool is not on the ARTG. Keep these together:

- the intended-purpose statement (README, "Intended purpose");
- this note, with the guidance dates below;
- the substitution table and its reasoning (`prd-auFTsanitiser.md` section 7) and the
  override format contract;
- the test evidence for the character-set invariant, determinism and PHI hygiene
  (`just check`, the corpus and fuzz suites);
- the change log of table and override changes with clinical sign-off.

## 8. What we read

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
  Version 2.4 (HL7AUSD-STD-OO-ADRM-2021.1), the standard the tool exists to meet.
