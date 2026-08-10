package builder

import "testing"

func TestLanguageBasePathPreservesPrefixedDefaultLanguage(t *testing.T) {
	got := languageBasePath("/rus/about/", "rus", "rus")
	if got != "/rus/" {
		t.Fatalf("expected /rus/, got %q", got)
	}
}

func TestLanguageBasePathKeepsUnprefixedDefaultLanguageAtRoot(t *testing.T) {
	got := languageBasePath("/about/", "rus", "rus")
	if got != "/" {
		t.Fatalf("expected /, got %q", got)
	}
}

func TestLanguageBasePathPrefixesNonDefaultLanguage(t *testing.T) {
	got := languageBasePath("/est/about/", "est", "rus")
	if got != "/est/" {
		t.Fatalf("expected /est/, got %q", got)
	}
}
