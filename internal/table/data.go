package table

// The v1 substitution table (PRD docs/prd-auFTsanitiser.md section 7).
// Grouped as in the PRD; stored flat. Every replacement is printable ASCII
// and contains no HL7 reserved delimiter — enforced by Validate and the
// package tests.

// spaceWrapped entries insert a surrounding space with local collapse
// (PRD Q4): 37.5°C -> 37.5 deg C, while 40 deg C stays untouched.
var spaceWrapped = map[rune]bool{
	0x00B0: true, // degree sign
	0x2103: true, // degree celsius
	0x2109: true, // degree fahrenheit
}

var substitutions = map[rune]string{
	// --- Whitespace and invisible ---
	0x00A0: " ", // no-break space
	0x202F: " ", // narrow no-break space
	0x200B: "",  // zero-width space (remove)
	0x200C: "",  // zero-width non-joiner (remove)
	0x200D: "",  // zero-width joiner (remove)
	0xFEFF: "",  // zero-width no-break space / BOM (remove)
	0x00AD: "",  // soft hyphen (remove)

	// --- Punctuation ---
	0x2018: "'",   // left single quotation mark
	0x2019: "'",   // right single quotation mark
	0x2032: "'",   // prime
	0x201A: "'",   // single low-9 quotation mark
	0x201C: `"`,   // left double quotation mark
	0x201D: `"`,   // right double quotation mark
	0x2033: `"`,   // double prime
	0x201E: `"`,   // double low-9 quotation mark
	0x2013: "-",   // en dash
	0x2014: "-",   // em dash
	0x2015: "-",   // horizontal bar
	0x2212: "-",   // minus sign
	0x2026: "...", // horizontal ellipsis
	0x2022: "*",   // bullet
	0x25CF: "*",   // black circle
	0x00B7: "*",   // middle dot
	0x00AB: "<<",  // left guillemet
	0x00BB: ">>",  // right guillemet
	0x2020: "+",   // dagger
	0x2021: "++",  // double dagger

	// --- Signs, units, mathematics ---
	0x00B0: "deg",      // degree sign (space-wrapped)
	0x2103: "deg C",    // degree celsius (space-wrapped)
	0x2109: "deg F",    // degree fahrenheit (space-wrapped)
	0x00B5: "u",        // micro sign (ug, umol)
	0x00B1: "+/-",      // plus-minus
	0x00D7: "x",        // multiplication sign
	0x00F7: "/",        // division sign
	0x2044: "/",        // fraction slash
	0x2264: "<=",       // less-than or equal
	0x2265: ">=",       // greater-than or equal
	0x2260: "not =",    // not equal
	0x2248: "approx.",  // almost equal — must never map to "~" (FR-012)
	0x2030: "per 1000", // per mille
	0x00B9: "1",        // superscript one
	0x00B2: "2",        // superscript two
	0x00B3: "3",        // superscript three
	0x207F: "n",        // superscript n
	0x2070: "0",        // remaining superscript digits and signs (AU 4.10: no superscripts in units)
	0x2071: "i",
	0x2074: "4",
	0x2075: "5",
	0x2076: "6",
	0x2077: "7",
	0x2078: "8",
	0x2079: "9",
	0x207A: "+",
	0x207B: "-",
	0x207C: "=",
	0x207D: "(",
	0x207E: ")",
	0x208A: "+", // subscript signs
	0x208B: "-",
	0x208C: "=",
	0x208D: "(",
	0x208E: ")",
	0x2080: "0", // subscript digits
	0x2081: "1",
	0x2082: "2",
	0x2083: "3",
	0x2084: "4",
	0x2085: "5",
	0x2086: "6",
	0x2087: "7",
	0x2088: "8",
	0x2089: "9",
	0x00BD: "1/2", // vulgar fractions
	0x00BC: "1/4",
	0x00BE: "3/4",
	0x2153: "1/3",
	0x2154: "2/3",
	0x215B: "1/8",
	0x215C: "3/8",
	0x215D: "5/8",
	0x215E: "7/8",
	0x2116: "No.",       // numero sign
	0x2190: "<-",        // leftwards arrow
	0x2192: "->",        // rightwards arrow
	0x2191: "increased", // upwards arrow (clinical raised)
	0x2193: "decreased", // downwards arrow (clinical lowered)

	// --- Greek letters (clinical usage; full alphabet, PRD section 7) ---
	// Small mu maps to "u" per the micro-sign rule, not to "mu".
	0x0391: "alpha", 0x03B1: "alpha",
	0x0392: "beta", 0x03B2: "beta",
	0x0393: "gamma", 0x03B3: "gamma",
	0x0394: "delta", 0x03B4: "delta",
	0x0395: "epsilon", 0x03B5: "epsilon",
	0x0396: "zeta", 0x03B6: "zeta",
	0x0397: "eta", 0x03B7: "eta",
	0x0398: "theta", 0x03B8: "theta",
	0x0399: "iota", 0x03B9: "iota",
	0x039A: "kappa", 0x03BA: "kappa",
	0x039B: "lambda", 0x03BB: "lambda",
	0x039C: "mu", 0x03BC: "u",
	0x039D: "nu", 0x03BD: "nu",
	0x039E: "xi", 0x03BE: "xi",
	0x039F: "omicron", 0x03BF: "omicron",
	0x03A0: "pi", 0x03C0: "pi",
	0x03A1: "rho", 0x03C1: "rho",
	0x03A3: "sigma", 0x03C3: "sigma", 0x03C2: "sigma",
	0x03A4: "tau", 0x03C4: "tau",
	0x03A5: "upsilon", 0x03C5: "upsilon",
	0x03A6: "phi", 0x03C6: "phi",
	0x03A7: "chi", 0x03C7: "chi",
	0x03A8: "psi", 0x03C8: "psi",
	0x03A9: "omega", 0x03C9: "omega",
	// Common symbol variants
	0x03D1: "theta", // theta symbol
	0x03D5: "phi",   // phi symbol
	0x03F5: "epsilon",

	// --- Commercial and legal ---
	0x2122: "(TM)", // trade mark
	0x00AE: "(R)",  // registered
	0x00A9: "(C)",  // copyright
	0x20AC: "EUR ", // euro
	0x00A3: "GBP ", // pound
	0x00A5: "JPY ", // yen
	0x00A2: "c",    // cent

	// --- Latin letters: explicit entries (PRD section 7; decomposition
	// fallback for accented letters lives in internal/sanitise/fold.go) ---
	0x00C6: "AE", 0x00E6: "ae",
	0x0152: "OE", 0x0153: "oe",
	0x00DF: "ss", // review item T044: ss vs beta for this feed
	0x00D0: "D", 0x00F0: "d",
	0x00DE: "Th", 0x00FE: "th",
	0x00D8: "O", 0x00F8: "o",
	0x0141: "L", 0x0142: "l",
	0x0131: "i",
}
