package homograph

import (
	"net/url"
	"strings"
	"unicode"
)

// Warning describes a suspicious character found in a hostname.
type Warning struct {
	Rune       rune
	Position   int
	Codepoint  string
	Name       string
	LooksLike  string
	IsPunycode bool
}

// Check inspects a hostname for homograph attack indicators.
// Returns one Warning per suspicious character, or a punycode warning for
// xn-- labels. Returns nil for clean hostnames.
func Check(hostname string) []Warning {
	if hostname == "" {
		return nil
	}

	var warnings []Warning

	// Check for punycode labels (xn-- prefix)
	for _, label := range strings.Split(hostname, ".") {
		if strings.HasPrefix(strings.ToLower(label), "xn--") {
			warnings = append(warnings, Warning{
				IsPunycode: true,
				Name:       "Punycode internationalized domain",
			})
		}
	}

	// Check each character for non-ASCII / confusable
	for i, r := range hostname {
		if r > unicode.MaxASCII {
			w := Warning{
				Rune:     r,
				Position: i,
			}
			if lookalike, ok := confusables[r]; ok {
				w.LooksLike = string(lookalike)
				w.Name = confusableNames[r]
			} else {
				w.LooksLike = "?"
				w.Name = "non-ASCII character"
			}
			warnings = append(warnings, w)
		}
	}

	return warnings
}

// CheckURL extracts the hostname from a URL and checks it for homograph
// indicators. Returns nil for clean URLs or unparseable inputs.
func CheckURL(rawURL string) []Warning {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil
	}
	hostname := u.Hostname()
	return Check(hostname)
}

// confusables maps Unicode characters that visually resemble ASCII letters.
// This is a curated subset of Unicode's confusables.txt covering the most
// common homograph attack vectors (Cyrillic, Greek, fullwidth Latin).
var confusables = map[rune]rune{
	// Cyrillic → Latin
	'а': 'a', // CYRILLIC SMALL LETTER A
	'е': 'e', // CYRILLIC SMALL LETTER IE
	'і': 'i', // CYRILLIC SMALL LETTER BYELORUSSIAN-UKRAINIAN I
	'о': 'o', // CYRILLIC SMALL LETTER O
	'р': 'p', // CYRILLIC SMALL LETTER ER
	'с': 'c', // CYRILLIC SMALL LETTER ES
	'у': 'y', // CYRILLIC SMALL LETTER U
	'х': 'x', // CYRILLIC SMALL LETTER HA
	'ѕ': 's', // CYRILLIC SMALL LETTER DZE
	'һ': 'h', // CYRILLIC SMALL LETTER SHHA
	'ԁ': 'd', // CYRILLIC SMALL LETTER KOMI DE

	// Cyrillic uppercase → Latin
	'А': 'A', // CYRILLIC CAPITAL LETTER A
	'В': 'B', // CYRILLIC CAPITAL LETTER VE
	'Е': 'E', // CYRILLIC CAPITAL LETTER IE
	'К': 'K', // CYRILLIC CAPITAL LETTER KA
	'М': 'M', // CYRILLIC CAPITAL LETTER EM
	'Н': 'H', // CYRILLIC CAPITAL LETTER EN
	'О': 'O', // CYRILLIC CAPITAL LETTER O
	'Р': 'P', // CYRILLIC CAPITAL LETTER ER
	'С': 'C', // CYRILLIC CAPITAL LETTER ES
	'Т': 'T', // CYRILLIC CAPITAL LETTER TE
	'Х': 'X', // CYRILLIC CAPITAL LETTER HA

	// Greek → Latin
	'ο': 'o', // GREEK SMALL LETTER OMICRON
	'α': 'a', // GREEK SMALL LETTER ALPHA (close)
	'ν': 'v', // GREEK SMALL LETTER NU

	// Fullwidth Latin → Latin
	'ａ': 'a', // FULLWIDTH LATIN SMALL LETTER A
	'ｂ': 'b', // FULLWIDTH LATIN SMALL LETTER B
	'ｃ': 'c', // FULLWIDTH LATIN SMALL LETTER C
	'ｄ': 'd', // FULLWIDTH LATIN SMALL LETTER D
	'ｅ': 'e', // FULLWIDTH LATIN SMALL LETTER E
	'ｆ': 'f', // FULLWIDTH LATIN SMALL LETTER F
	'ｇ': 'g', // FULLWIDTH LATIN SMALL LETTER G
	'ｈ': 'h', // FULLWIDTH LATIN SMALL LETTER H
	'ｉ': 'i', // FULLWIDTH LATIN SMALL LETTER I
	'ｊ': 'j', // FULLWIDTH LATIN SMALL LETTER J
	'ｋ': 'k', // FULLWIDTH LATIN SMALL LETTER K
	'ｌ': 'l', // FULLWIDTH LATIN SMALL LETTER L
	'ｍ': 'm', // FULLWIDTH LATIN SMALL LETTER M
	'ｎ': 'n', // FULLWIDTH LATIN SMALL LETTER N
	'ｏ': 'o', // FULLWIDTH LATIN SMALL LETTER O
	'ｐ': 'p', // FULLWIDTH LATIN SMALL LETTER P
	'ｑ': 'q', // FULLWIDTH LATIN SMALL LETTER Q
	'ｒ': 'r', // FULLWIDTH LATIN SMALL LETTER R
	'ｓ': 's', // FULLWIDTH LATIN SMALL LETTER S
	'ｔ': 't', // FULLWIDTH LATIN SMALL LETTER T
	'ｕ': 'u', // FULLWIDTH LATIN SMALL LETTER U
	'ｖ': 'v', // FULLWIDTH LATIN SMALL LETTER V
	'ｗ': 'w', // FULLWIDTH LATIN SMALL LETTER W
	'ｘ': 'x', // FULLWIDTH LATIN SMALL LETTER X
	'ｙ': 'y', // FULLWIDTH LATIN SMALL LETTER Y
	'ｚ': 'z', // FULLWIDTH LATIN SMALL LETTER Z
}

var confusableNames = map[rune]string{
	'а': "CYRILLIC SMALL LETTER A",
	'е': "CYRILLIC SMALL LETTER IE",
	'і': "CYRILLIC SMALL LETTER BYELORUSSIAN-UKRAINIAN I",
	'о': "CYRILLIC SMALL LETTER O",
	'р': "CYRILLIC SMALL LETTER ER",
	'с': "CYRILLIC SMALL LETTER ES",
	'у': "CYRILLIC SMALL LETTER U",
	'х': "CYRILLIC SMALL LETTER HA",
	'ѕ': "CYRILLIC SMALL LETTER DZE",
	'һ': "CYRILLIC SMALL LETTER SHHA",
	'ԁ': "CYRILLIC SMALL LETTER KOMI DE",
	'А': "CYRILLIC CAPITAL LETTER A",
	'В': "CYRILLIC CAPITAL LETTER VE",
	'Е': "CYRILLIC CAPITAL LETTER IE",
	'К': "CYRILLIC CAPITAL LETTER KA",
	'М': "CYRILLIC CAPITAL LETTER EM",
	'Н': "CYRILLIC CAPITAL LETTER EN",
	'О': "CYRILLIC CAPITAL LETTER O",
	'Р': "CYRILLIC CAPITAL LETTER ER",
	'С': "CYRILLIC CAPITAL LETTER ES",
	'Т': "CYRILLIC CAPITAL LETTER TE",
	'Х': "CYRILLIC CAPITAL LETTER HA",
	'ο': "GREEK SMALL LETTER OMICRON",
	'α': "GREEK SMALL LETTER ALPHA",
	'ν': "GREEK SMALL LETTER NU",
	'ａ': "FULLWIDTH LATIN SMALL LETTER A",
	'ｂ': "FULLWIDTH LATIN SMALL LETTER B",
	'ｃ': "FULLWIDTH LATIN SMALL LETTER C",
	'ｄ': "FULLWIDTH LATIN SMALL LETTER D",
	'ｅ': "FULLWIDTH LATIN SMALL LETTER E",
	'ｆ': "FULLWIDTH LATIN SMALL LETTER F",
	'ｇ': "FULLWIDTH LATIN SMALL LETTER G",
	'ｈ': "FULLWIDTH LATIN SMALL LETTER H",
	'ｉ': "FULLWIDTH LATIN SMALL LETTER I",
	'ｊ': "FULLWIDTH LATIN SMALL LETTER J",
	'ｋ': "FULLWIDTH LATIN SMALL LETTER K",
	'ｌ': "FULLWIDTH LATIN SMALL LETTER L",
	'ｍ': "FULLWIDTH LATIN SMALL LETTER M",
	'ｎ': "FULLWIDTH LATIN SMALL LETTER N",
	'ｏ': "FULLWIDTH LATIN SMALL LETTER O",
	'ｐ': "FULLWIDTH LATIN SMALL LETTER P",
	'ｑ': "FULLWIDTH LATIN SMALL LETTER Q",
	'ｒ': "FULLWIDTH LATIN SMALL LETTER R",
	'ｓ': "FULLWIDTH LATIN SMALL LETTER S",
	'ｔ': "FULLWIDTH LATIN SMALL LETTER T",
	'ｕ': "FULLWIDTH LATIN SMALL LETTER U",
	'ｖ': "FULLWIDTH LATIN SMALL LETTER V",
	'ｗ': "FULLWIDTH LATIN SMALL LETTER W",
	'ｘ': "FULLWIDTH LATIN SMALL LETTER X",
	'ｙ': "FULLWIDTH LATIN SMALL LETTER Y",
	'ｚ': "FULLWIDTH LATIN SMALL LETTER Z",
}
