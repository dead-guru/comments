package i18n

import "testing"

func TestNormalizeLocale(t *testing.T) {
	tests := map[string]string{
		"":      LocaleEnglish,
		"en-US": LocaleEnglish,
		"uk":    LocaleUkrainian,
		"uk-UA": LocaleUkrainian,
		"fr":    LocaleEnglish,
	}
	for input, want := range tests {
		if got := Normalize(input, ""); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestEmbedCatalogFallsBackToEnglishKeys(t *testing.T) {
	english := Embed(LocaleEnglish)
	ukrainian := Embed(LocaleUkrainian)
	for key := range english {
		if ukrainian[key] == "" {
			t.Fatalf("expected Ukrainian catalog to include key %q", key)
		}
	}
}
