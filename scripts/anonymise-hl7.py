#!/usr/bin/env python3
"""Anonymise HL7 v2 messages so they can be kept as test fixtures.

    python3 scripts/anonymise-hl7.py <in.hl7> [out.hl7]      one file (stdout when out is omitted)
    python3 scripts/anonymise-hl7.py <in-dir> <out-dir>      every *.hl7 under in-dir
    python3 scripts/anonymise-hl7.py --self-test

Every replacement is a deterministic function of the original value (sha256), so one
patient, doctor or accession maps to the same synthetic value in every message and run.

  PID-3, MSH-10, OBR-2/3 ids, XCN/NDL ids, MSH-6   same-shape pseudo value (digit->digit,
                                                   letter->letter), replaced everywhere it occurs
  PID-5 / NK1-2 names                              SURNAME####^GIVEN####
  PV1-7/8/9/17, OBR-16/28 (XCN)                    DOCTOR####^GIVEN####, or PRACTICE#### when the
                                                   family component is a practice name (Imaging,
                                                   Radiology, Medical, ... in it)
  application / site names (ORGS)                  fixed replacements, e.g. ContrastONE -> RISONE,
                                                   wherever they occur
  OBR-32..35 (NDL)                                 RADIOLOGIST####&GIVEN####
  every date / timestamp (fields and free text)    shifted by a per-patient offset of 1-365 days
  Medicare / IHI / DVA / mobile numbers            digits zeroed (patterns from scan-for-phi.py)
  free text                                        the names above replaced in Capitalised and
                                                   Capitalised form; leftover "Dr/Mr/Mrs/Ms Name",
                                                   "Typist: Name", "referring Name" tokenised

Organisation names not listed in ORGS and not practice names in XCN fields are kept.
The script refuses to write output in which any original name or id can still be found.
"""
import datetime
import hashlib
import importlib.util
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_spec = importlib.util.spec_from_file_location("scan_for_phi", os.path.join(ROOT, "scripts", "scan-for-phi.py"))
scan = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(scan)

TITLES = r"(?:Dr|Mr|Mrs|Ms|Miss|Prof|Professor|A/Prof)"
TRIGGER = r"(?:%s\.?|Typist:|[Rr]eferring)\s+" % TITLES
NAME_WORD = r"(?!%s\b)[A-Z][A-Za-z'\-]+(?![A-Za-z0-9])" % TITLES
TOKEN = r"[A-Z]+\d{4}"
MONTHS = "Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec"
# Application and site names: fixed, consistent replacements applied wherever they occur.
ORGS = {"ContrastONE": "RISONE", "Pro Pacs": "PACSONE", "MedicalObj": "MSGONE"}
# OBR-2/3 namespace component (the filler application); matched whole, not as a word in text.
ORG_NAMESPACES = {"Contrast": "RIS"}
# Words that mark an XCN family-name component as a practice name ("Murphy Street Medical"):
# the whole component becomes PRACTICE####; these words are not replaced on their own, so
# report headings such as "Radiology Report" survive.
ORG_WORDS = set("AND AUSTRALIA CENTRE CENTER CLINIC DIAGNOSTIC DR FAMILY GENERAL GROUP HEALTH HOSPITAL IMAGING "
                "LINK LTD MEDICAL MOBILE OF PLUS PRACTICE PTY RADIOLOGY ROAD RURAL SERVICES SOLUTIONS SPECIALIST "
                "STREET SURGERY THE ULTRASOUND XRAY X-RAY".split())


def digest(value):
    return hashlib.sha256(value.encode("utf-8", "surrogateescape")).hexdigest()


def tag(kind, value):
    return f"{kind}{int(digest(value)[:8], 16) % 10000:04d}"


def pseudo(value):
    """Same shape as value (digits, upper, lower kept per position), different content."""
    h = digest(value)
    out = []
    for i, c in enumerate(value):
        n = int(h[(2 * i) % 64:(2 * i) % 64 + 2], 16)
        # First digit never 2-6 and no DVA state letter (N V Q S W T), so a pseudo id can never
        # match scan-for-phi.py's Medicare or DVA shapes by chance.
        out.append(("01789"[n % 5] if i == 0 else str(n % 10)) if c.isdigit()
                   else "ABCDEFGHIJKLMOPRUXYZ"[n % 20] if c.isupper() else chr(97 + n % 26) if c.islower() else c)
    return "".join(out)


def name_forms(upper):
    """Regex for a name as it appears in text: UPPER, or Capitalised with any case after (McKeen)."""
    return r"\b(?:%s|%s(?i:%s))\b" % (re.escape(upper), re.escape(upper[0]), re.escape(upper[1:]))


class Message:
    def __init__(self, text, names=None):
        self.text = text
        self.ids = {}    # original id -> pseudo
        # original name (upper) -> synthetic. A directory run shares one dict so a name that
        # one message's fields register is also replaced in another message's free text.
        self.names = {} if names is None else names
        self.offset = datetime.timedelta(days=1)

    def person(self, field, sep, kind):
        """XCN (sep ^) or NDL (sep &): id^family^given^middle... -> pseudo id and synthetic names."""
        reps = []
        for rep in field.split("~"):
            c = rep.split(sep)
            if c[0]:
                self.ids[c[0]] = pseudo(c[0])
                c[0] = self.ids[c[0]]
            for i, k in ((1, kind), (2, "GIVEN"), (3, "GIVEN")):
                if len(c) > i and c[i]:
                    upper = c[i].upper()
                    words = [w for w in re.split(r"[^A-Z0-9']+", upper) if len(w) >= 2]
                    whole = "PRACTICE" if kind == "DOCTOR" and set(words) & ORG_WORDS else k
                    self.names[upper] = tag(whole, upper)
                    for word in words:  # a surname or town inside a practice name, used alone in text
                        if word not in ORG_WORDS:
                            self.names[word] = tag(k, word)
                    c[i] = self.names[upper]
            reps.append(sep.join(c))
        return "~".join(reps)

    def pid_name(self, field, kind):
        # PID-5 / NK1-2 are XPN: family^given^middle, i.e. XCN without the leading id.
        return self.person("^" + field, "^", kind)[1:]

    def anonymise(self):
        sep = "\r\n" if "\r\n" in self.text else "\r" if "\r" in self.text else "\n"
        segs = []
        for seg in self.text.split(sep):
            f = seg.split("|")
            if f[0] == "MSH" and len(f) > 9:
                self.ids[f[5]] = pseudo(f[5]) if f[5] else ""
                f[9] = pseudo(f[9])
            elif f[0] == "PID" and len(f) > 5:
                self.offset = datetime.timedelta(days=int(digest(f[3])[:4], 16) % 365 + 1)
                self.ids[f[3]] = f[3] = pseudo(f[3]) if f[3] else ""
                f[5] = self.pid_name(f[5], "SURNAME")
            elif f[0] == "NK1" and len(f) > 2:
                f[2] = self.pid_name(f[2], "CONTACT")
            elif f[0] == "PV1":
                for i in (7, 8, 9, 17):
                    if len(f) > i:
                        f[i] = self.person(f[i], "^", "DOCTOR")
            elif f[0] == "OBR":
                for i in (2, 3):
                    if len(f) > i and f[i]:
                        c = f[i].split("^")
                        self.ids[c[0]] = c[0] = pseudo(c[0])
                        f[i] = "^".join([c[0]] + [ORG_NAMESPACES.get(x, x) for x in c[1:]])
                for i in (16, 28):
                    if len(f) > i:
                        f[i] = self.person(f[i], "^", "DOCTOR")
                for i in (32, 33, 34, 35):
                    if len(f) > i:
                        f[i] = self.person(f[i], "&", "RADIOLOGIST")
            segs.append("|".join(f))
        text = sep.join(segs)
        for org, fake in ORGS.items():
            text = re.sub(r"(?<![A-Za-z0-9])" + re.escape(org) + r"(?![A-Za-z0-9])", fake, text)
        self.ids.pop("", None)
        for old in sorted(self.ids, key=len, reverse=True):
            text = re.sub(r"(?<![A-Za-z0-9])" + re.escape(old) + r"(?![A-Za-z0-9])", self.ids[old], text)
        for old in sorted(self.names, key=len, reverse=True):
            text = re.sub(name_forms(old), self.names[old], text)
        text = re.sub(r"\b(%s)((?:%s\s+)*)(%s)(\s+%s)?" % (TRIGGER, TOKEN, NAME_WORD, NAME_WORD), self.title_name, text)
        text = re.sub(r"(?<!\d)(\d{8})(?=(?:\d{6})?(?:[+-]\d{4})?(?!\d))", lambda m: self.shift(m.group(1), "%Y%m%d"), text)
        text = re.sub(r"\b(\d{1,2}[/.-]\d{1,2}[/.-](?:\d{4}|\d{2}))\b", lambda m: self.shift(m.group(1), "%d/%m/%Y", "%d/%m/%y"), text)
        text = re.sub(r"\b(\d{1,2})(?:st|nd|rd|th)?\s+((?:%s)[a-z]*)\s+(\d{4})\b" % MONTHS,
                      lambda m: self.shift(" ".join(m.groups()), "%d %B %Y", "%d %b %Y"), text)
        for rx in (scan.NASH, scan.MEDICARE, scan.MEDICARE_SPACED, scan.MOBILE):
            text = rx.sub(lambda m: re.sub(rb"\d", b"0", m.group(0)), text.encode("utf-8", "surrogateescape")).decode("utf-8", "surrogateescape")
        self.check(text)
        return text

    def title_name(self, m):
        return m.group(1) + m.group(2) + tag("DOCTOR", m.group(3).upper()) + (" " + tag("GIVEN", m.group(4).strip().upper()) if m.group(4) else "")

    def shift(self, value, *formats):
        for fmt in formats:
            try:
                d = datetime.datetime.strptime(value.replace(".", "/").replace("-", "/"), fmt)
            except ValueError:
                continue
            return (d - self.offset).strftime(fmt)
        return value

    def check(self, text):
        for old in self.ids:
            if re.search(r"(?<![A-Za-z0-9])" + re.escape(old) + r"(?![A-Za-z0-9])", text):
                raise SystemExit(f"anonymise: original id still present ({len(old)} chars)")
        for old in self.names:
            if re.search(name_forms(old), text):
                raise SystemExit(f"anonymise: original name still present ({len(old)} chars)")


def anonymise_file(src, dst, names=None):
    with open(src, "rb") as f:
        out = Message(f.read().decode("utf-8", "surrogateescape"), names).anonymise().encode("utf-8", "surrogateescape")
    if dst is None:
        sys.stdout.buffer.write(out)
    else:
        with open(dst, "wb") as f:
            f.write(out)


def self_test():
    msg = ("MSH|^~\\&|ContrastONE|Pro Pacs|MedicalObj|PI254812|20260706234114||ORU^R01|2078659|P|2.4\r"
           "PID|||32795||SALTER^TREVOR^^^^^L||19520424|M|||\r"
           "PV1||O|OP^^^^^^^^Pro Pacs|||||PI254812^Eclipse Imaging^^^^Dr|PI254812^Murphy Street Medical^^^^Dr^^AUSHICPR\r"
           "OBR|1||60462-US^Contrast|US Shoulder||20260706000000+1000||||||||||PI254812^Eclipse Imaging^^^^Dr^^AUSHICPR"
           "||||||||||||||||proVincentL&Lai&Vincent|rad&McKeen&Kim|123-John\r"
           "OBX|1|FT|TXT^^AUSPDI||Thank you for referring Mr Salter (DOB 24/04/1952, ID 32795) seen 6 July 2026.\\.br\\"
           "Dr Jane Bloggs reviewed. Lai agrees. Medicare 2123 45670 1. Call 0412 345 678.\\.br\\"
           "Dr Vincent Ho and Dr Murphy. Typist: John McKEEN. Eclipse Imaging. Radiology Report. Sent via Pro Pacs.||||||F\r")
    out = Message(msg).anonymise()
    seg = {s.split("|")[0]: s.split("|") for s in out.split("\r") if s}
    assert seg["PID"][3] != "32795" and seg["PID"][3].isdigit() and len(seg["PID"][3]) == 5, seg["PID"][3]
    assert re.fullmatch(r"SURNAME\d{4}\^GIVEN\d{4}\^\^\^\^\^L", seg["PID"][5]), seg["PID"][5]
    assert seg["PID"][7] != "19520424" and seg["PID"][7].isdigit(), seg["PID"][7]
    assert seg["MSH"][5] == seg["PV1"][8].split("^")[0] != "PI254812"
    assert re.fullmatch(r"[A-Z]{2}\d{6}\^PRACTICE\d{4}\^\^\^\^Dr\^\^AUSHICPR", seg["OBR"][16]), seg["OBR"][16]
    assert seg["MSH"][2:5] == ["RISONE", "PACSONE", "MSGONE"] and seg["PV1"][3] == "OP^^^^^^^^PACSONE", seg["MSH"]
    assert re.fullmatch(r"\d{5}-[A-Z]{2}\^RIS", seg["OBR"][3]), seg["OBR"][3]
    assert re.fullmatch(r"[a-z]{3}[A-Z][a-z]{6}[A-Z]&RADIOLOGIST\d{4}&GIVEN\d{4}", seg["OBR"][32]), seg["OBR"][32]
    assert re.fullmatch(r"\d{3}-[A-Z][a-z]{3}", seg["OBR"][34]), seg["OBR"][34]
    text = seg["OBX"][5]
    for leaked in ("Salter", "SALTER", "Trevor", "Lai", "Bloggs", "Eclipse", "Murphy", "McKEEN", "Ho ", "Vincent", "32795", "ContrastONE", "Pro Pacs", "MedicalObj",
                   "24/04/1952", "6 July 2026", "2123 45670 1", "0412 345 678"):
        assert leaked not in text, (leaked, text)
    assert re.search(r"referring Mr SURNAME\d{4} \(DOB \d\d/\d\d/\d{4}, ID \d{5}\) seen \d{2} [A-Z][a-z]+ \d{4}", text), text
    assert re.search(r"Dr DOCTOR\d{4} GIVEN\d{4} reviewed\. RADIOLOGIST\d{4} agrees\. Medicare 0000 00000 0\. Call 0000 000 000\.", text), text
    assert re.search(r"Dr GIVEN\d{4} DOCTOR\d{4} and Dr DOCTOR\d{4}\. Typist: DOCTOR\d{4} RADIOLOGIST\d{4}\. "
                     r"PRACTICE\d{4}\. Radiology Report\. Sent via PACSONE\.", text), text
    assert re.fullmatch(r"[a-z]{3}&RADIOLOGIST\d{4}&GIVEN\d{4}", seg["OBR"][33]), seg["OBR"][33]
    assert Message(msg).anonymise() == out, "not deterministic"
    assert not [h for h in scan.scan_blob(out.encode(), "x.hl7")], scan.scan_blob(out.encode(), "x.hl7")
    print("self-test: ok")
    return 0


def main(argv):
    if "--self-test" in argv:
        return self_test()
    if not 1 <= len(argv) <= 2:
        print(__doc__.split("\n\n")[1], file=sys.stderr)
        return 2
    src, dst = argv[0], argv[1] if len(argv) == 2 else None
    if os.path.isdir(src):
        if dst is None:
            print("anonymise: an output directory is required for directory input", file=sys.stderr)
            return 2
        os.makedirs(dst, exist_ok=True)
        files = sorted(n for n in os.listdir(src) if n.lower().endswith(".hl7"))
        registry = {}
        for n in files:  # collection pass: fill the shared name registry before writing anything
            anonymise_file(os.path.join(src, n), os.devnull, registry)
        for n in files:
            anonymise_file(os.path.join(src, n), os.path.join(dst, n), registry)
        print(f"anonymise: {len(files)} file(s) written to {dst}", file=sys.stderr)
    else:
        anonymise_file(src, dst)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
