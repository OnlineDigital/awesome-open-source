package i18n

import "testing"

func TestBundleKeysComplete(t *testing.T) {
	b, err := Load("en", []string{"en", "ro", "es", "fr", "de"})
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range b.Locales {
		for key := range b.strings["en"] {
			if b.strings[loc][key] == "" {
				t.Errorf("locale %s is missing key %s", loc, key)
			}
		}
		for key := range b.strings[loc] {
			if _, ok := b.strings["en"][key]; !ok {
				t.Errorf("locale %s has unknown key %s", loc, key)
			}
		}
	}
}

func TestNegotiateAndT(t *testing.T) {
	b, _ := Load("en", []string{"en", "ro", "de"})
	if got := b.Negotiate("ro-RO,ro;q=0.9,en;q=0.8"); got != "ro" {
		t.Fatal(got)
	}
	if got := b.Negotiate("fr-FR,de;q=0.7"); got != "de" {
		t.Fatal(got)
	}
	if got := b.Negotiate("ja"); got != "en" {
		t.Fatal(got)
	}
	if got := b.T("ro", "category.title", "name", "Baze de date"); got != "Baze de date open source" {
		t.Fatal(got)
	}
}
