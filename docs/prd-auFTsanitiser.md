# Plain-Text Report Sanitiser for Australian HL7 v2.4 Messaging

### Whitespace and invisible

| Input | Codepoint | Replacement |
|---|---|---|
| No-break space, narrow NBSP | U+00A0, U+202F | space |
| Zero-width space/joiner/non-joiner, BOM | U+200B–U+200D, U+FEFF | (remove) |
| Soft hyphen | U+00AD | (remove) |
| Line/paragraph separator, NEL | U+2028, U+2029, U+0085 | line break |
| Tab | U+0009 | space |

### Punctuation

| Input | Codepoint | Replacement |
|---|---|---|
| Left/right single quote, prime | U+2018, U+2019, U+2032 | `'` |
| Left/right double quote, double prime | U+201C, U+201D, U+2033 | `"` |
| Single/double low-9 quote | U+201A, U+201E | `'` / `"` |
| En dash, em dash, horizontal bar, minus sign | U+2013, U+2014, U+2015, U+2212 | `-` |
| Ellipsis | U+2026 | `...` |
| Bullet, black circle, middle dot | U+2022, U+25CF, U+00B7 | `*` |
| Guillemets | U+00AB, U+00BB | `<<` / `>>` |
| Dagger, double dagger | U+2020, U+2021 | `+` / `++` |

### Signs, units, mathematics

| Input | Codepoint | Replacement |
|---|---|---|
| Degree sign (and °C/°F combined forms) | U+00B0 (U+2103, U+2109) | `deg` (`deg C`, `deg F`) |
| Micro sign, Greek small mu | U+00B5, U+03BC | `u` (so `µg` -> `ug`, `µmol` -> `umol`) |
| Plus-minus | U+00B1 | `+/-` |
| Multiplication sign | U+00D7 | `x` |
| Division sign, fraction slash | U+00F7, U+2044 | `/` |
| Less/greater-or-equal | U+2264, U+2265 | `<=` / `>=` |
| Not equal | U+2260 | `not =` |
| Almost equal | U+2248 | `approx.` |
| Per mille | U+2030 | `per 1000` |
| Superscript 1, 2, 3, n | U+00B9, U+00B2, U+00B3, U+207F | `1`, `2`, `3`, `n` (so `m²` -> `m2`, `cm³` -> `cm3`) |
| Subscript digits | U+2080–U+2089 | plain digit |
| Vulgar fractions | U+00BD, U+00BC, U+00BE, U+2153, U+2154, U+215B–U+215E | `1/2`, `1/4`, `3/4`, `1/3`, `2/3`, `1/8`, `3/8`, `5/8`, `7/8` |
| Numero sign | U+2116 | `No.` |
| Arrows left/right/up/down | U+2190, U+2192, U+2191, U+2193 | `<-`, `->`, `increased`, `decreased` |



### Greek letters (clinical usage)

| Input | Replacement |
|---|---|
| α / Α | `alpha` |
| β / Β | `beta` |
| γ / Γ | `gamma` |
| δ / Δ | `delta` |
| κ, λ, σ, τ, φ, ω (and capitals) | `kappa`, `lambda`, `sigma`, `tau`, `phi`, `omega` |
| Any other Greek letter | letter name (full table in implementation) |

### Commercial and legal

| Input | Codepoint | Replacement |
|---|---|---|
| Trade mark | U+2122 | `(TM)` |
| Registered | U+00AE | `(R)` |
| Copyright | U+00A9 | `(C)` |
| Euro, pound, yen, cent | U+20AC, U+00A3, U+00A5, U+00A2 | `EUR `, `GBP `, `JPY `, `c` |

### Latin letters (ascii mode)

strip combining marks (é→e, ü→u, ñ→n, ç→c), plus
explicit entries: `Æ/æ`→`AE/ae`, `Œ/œ`→`OE/oe`, `ß`→`ss`, `Ð/ð`→`D/d`, `Þ/þ`→`Th/th`,
`Ø/ø`→`O/o`, `Ł/ł`→`L/l`, `ı`→`i`.



## References

HL7 Australia, *Australian Diagnostics and Referral Messaging — Localisation of HL7
Version 2.4* (archive): overview page, Chapter 3 Datatypes (character sets, escape
sequences, AU prohibition of `\Xdddd\`/`\Cxxyy\`/`\Mxxyyzz\`), Appendix 1 Parsing
HL7v2 (permitted byte ranges, delimiter escaping, segment terminator).
