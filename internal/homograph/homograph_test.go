package homograph

import (
	"testing"
)

// --- Non-ASCII detection ---

func TestCheck_PureASCII(t *testing.T) {
	warnings := Check("github.com")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for pure ASCII, got %v", warnings)
	}
}

func TestCheck_CyrillicI(t *testing.T) {
	// U+0456 CYRILLIC SMALL LETTER BYELORUSSIAN-UKRAINIAN I looks like Latin "i"
	warnings := Check("gіthub.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for Cyrillic і in hostname")
	}
	found := false
	for _, w := range warnings {
		if w.Rune == 'і' {
			found = true
			if w.Position != 1 {
				t.Errorf("expected position 1, got %d", w.Position)
			}
			if w.LooksLike == "" {
				t.Error("expected LooksLike to identify the Latin equivalent")
			}
		}
	}
	if !found {
		t.Errorf("expected warning for U+0456, got: %v", warnings)
	}
}

func TestCheck_CyrillicA(t *testing.T) {
	// U+0430 CYRILLIC SMALL LETTER A looks like Latin "a"
	warnings := Check("аmazon.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for Cyrillic а in hostname")
	}
	if warnings[0].Rune != 'а' {
		t.Errorf("expected U+0430, got U+%04X", warnings[0].Rune)
	}
}

func TestCheck_MultipleSpoofs(t *testing.T) {
	// "gіthub" with Cyrillic і (U+0456) and "cоm" with Cyrillic о (U+043E)
	warnings := Check("gіthub.cоm")
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestCheck_PunycodePrefix(t *testing.T) {
	// xn-- is the punycode prefix for internationalized domain names
	warnings := Check("xn--github-c1a.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for punycode domain")
	}
	hasPunycode := false
	for _, w := range warnings {
		if w.IsPunycode {
			hasPunycode = true
		}
	}
	if !hasPunycode {
		t.Error("expected a punycode-type warning")
	}
}

func TestCheck_PunycodeInSubdomain(t *testing.T) {
	warnings := Check("xn--n3h.example.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for punycode subdomain")
	}
}

func TestCheck_NumericAndHyphen(t *testing.T) {
	// Legitimate domains with hyphens and numbers
	warnings := Check("my-cdn-123.example.com")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for ASCII domain with hyphens, got %v", warnings)
	}
}

func TestCheck_IPAddress(t *testing.T) {
	warnings := Check("192.168.1.1")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for IP address, got %v", warnings)
	}
}

func TestCheck_Localhost(t *testing.T) {
	warnings := Check("localhost")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for localhost, got %v", warnings)
	}
}

func TestCheck_EmptyHostname(t *testing.T) {
	warnings := Check("")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for empty hostname, got %v", warnings)
	}
}

// --- Full-width characters ---

func TestCheck_FullWidthLatin(t *testing.T) {
	// U+FF47 FULLWIDTH LATIN SMALL LETTER G
	warnings := Check("ｇithub.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for fullwidth character")
	}
}

// --- Greek confusables ---

func TestCheck_GreekOmicron(t *testing.T) {
	// U+03BF GREEK SMALL LETTER OMICRON looks like Latin "o"
	warnings := Check("gοogle.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for Greek omicron")
	}
}

// --- Non-ASCII character not in confusables table ---

func TestCheck_UnknownNonASCII(t *testing.T) {
	// U+0101 LATIN SMALL LETTER A WITH MACRON — non-ASCII but not in our
	// confusables table. Should still warn, with LooksLike="?"
	warnings := Check("exāmple.com")
	if len(warnings) == 0 {
		t.Fatal("expected warning for non-ASCII character not in confusables")
	}
	found := false
	for _, w := range warnings {
		if w.Rune == 'ā' {
			found = true
			if w.LooksLike != "?" {
				t.Errorf("expected LooksLike='?' for unknown confusable, got %q", w.LooksLike)
			}
			if w.Name != "non-ASCII character" {
				t.Errorf("expected generic name, got %q", w.Name)
			}
		}
	}
	if !found {
		t.Errorf("expected warning for U+0101, got: %v", warnings)
	}
}

// --- CheckURL convenience function ---

func TestCheckURL_ExtractsHostname(t *testing.T) {
	warnings := CheckURL("https://gіthub.com/install.sh")
	if len(warnings) == 0 {
		t.Fatal("expected warning from URL with spoofed hostname")
	}
}

func TestCheckURL_CleanURL(t *testing.T) {
	warnings := CheckURL("https://github.com/install.sh")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for clean URL, got %v", warnings)
	}
}

func TestCheckURL_InvalidURL(t *testing.T) {
	warnings := CheckURL("not a url at all")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for invalid URL, got %v", warnings)
	}
}

func TestCheckURL_NoHost(t *testing.T) {
	// A path-only "URL" parses successfully but has empty Host
	warnings := CheckURL("/just/a/path")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for path-only URL, got %v", warnings)
	}
}

func TestCheckURL_OpaqueURI(t *testing.T) {
	warnings := CheckURL("mailto:user@example.com")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for mailto URI, got %v", warnings)
	}
}

func TestCheckURL_WithPort(t *testing.T) {
	warnings := CheckURL("https://gіthub.com:443/path")
	if len(warnings) == 0 {
		t.Fatal("expected warning even with port in URL")
	}
}
