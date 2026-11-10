// Package i18n holds UI strings for every supported locale and negotiates the
// request locale. Content (projects, categories) is translated in the database.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed locales/*.json
var files embed.FS

type Bundle struct {
	Default string
	Locales []string
	strings map[string]map[string]string
}

// NativeNames are shown in the language switcher.
var NativeNames = map[string]string{"en": "English", "ro": "Română", "es": "Español", "fr": "Français", "de": "Deutsch", "it": "Italiano", "pt": "Português", "nl": "Nederlands", "pl": "Polski", "ja": "日本語", "zh": "中文"}

// EnglishNames are used in translation prompts.
var EnglishNames = map[string]string{"en": "English", "ro": "Romanian", "es": "Spanish", "fr": "French", "de": "German", "it": "Italian", "pt": "Portuguese", "nl": "Dutch", "pl": "Polish", "ja": "Japanese", "zh": "Chinese"}

// OGLocales maps to Open Graph locale codes.
var OGLocales = map[string]string{"en": "en_US", "ro": "ro_RO", "es": "es_ES", "fr": "fr_FR", "de": "de_DE", "it": "it_IT", "pt": "pt_PT", "nl": "nl_NL", "pl": "pl_PL", "ja": "ja_JP", "zh": "zh_CN"}

func Load(defaultLocale string, enabled []string) (*Bundle, error) {
	b := &Bundle{Default: defaultLocale, strings: map[string]map[string]string{}}
	for _, loc := range enabled {
		data, err := files.ReadFile("locales/" + loc + ".json")
		if err != nil {
			return nil, fmt.Errorf("i18n: no UI strings for locale %q (add app/i18n/locales/%s.json)", loc, loc)
		}
		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("i18n: %s.json: %w", loc, err)
		}
		b.strings[loc] = m
		b.Locales = append(b.Locales, loc)
	}
	if _, ok := b.strings[defaultLocale]; !ok {
		return nil, fmt.Errorf("i18n: default locale %q is not enabled", defaultLocale)
	}
	sort.SliceStable(b.Locales, func(i, j int) bool { return b.Locales[i] == defaultLocale && b.Locales[j] != defaultLocale })
	return b, nil
}

func (b *Bundle) Has(loc string) bool { _, ok := b.strings[loc]; return ok }

// Strings returns the dictionary for a locale, falling back to the default per key.
func (b *Bundle) Strings(loc string) map[string]string {
	out := make(map[string]string, len(b.strings[b.Default]))
	for k, v := range b.strings[b.Default] {
		out[k] = v
	}
	for k, v := range b.strings[loc] {
		out[k] = v
	}
	return out
}

// T translates a key and substitutes {name} placeholders from pairs (name, value, ...).
func (b *Bundle) T(loc, key string, pairs ...any) string {
	s, ok := b.strings[loc][key]
	if !ok {
		s, ok = b.strings[b.Default][key]
	}
	if !ok {
		return key
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		s = strings.ReplaceAll(s, "{"+fmt.Sprint(pairs[i])+"}", fmt.Sprint(pairs[i+1]))
	}
	return s
}

// Negotiate picks the best enabled locale from an Accept-Language header.
func (b *Bundle) Negotiate(header string) string {
	type cand struct {
		tag string
		q   float64
	}
	var cs []cand
	for _, part := range strings.Split(header, ",") {
		tag, q, _ := strings.Cut(strings.TrimSpace(part), ";q=")
		quality := 1.0
		if q != "" {
			if f, err := strconv.ParseFloat(q, 64); err == nil {
				quality = f
			}
		}
		cs = append(cs, cand{strings.ToLower(strings.TrimSpace(tag)), quality})
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].q > cs[j].q })
	for _, c := range cs {
		base, _, _ := strings.Cut(c.tag, "-")
		if b.Has(c.tag) {
			return c.tag
		}
		if b.Has(base) {
			return base
		}
	}
	return b.Default
}
