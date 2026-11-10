package application

import (
	"context"
	"fmt"

	"awesomeopensource.dev/app/store"
)

// TranslateProject machine-translates the default-locale text into the given locales
// (all non-default locales when empty). Human-written translations are kept unless force is set.
// It returns the locales that were translated.
func (a *Application) TranslateProject(ctx context.Context, id int64, locales []string, force bool) ([]string, error) {
	trs, err := a.Store.Translations(ctx, id)
	if err != nil {
		return nil, err
	}
	src, ok := trs[a.Cfg.DefaultLocale]
	if !ok || (src.Tagline == "" && src.Description == "") {
		return nil, fmt.Errorf("add the %s text first: there is nothing to translate", a.Cfg.DefaultLocale)
	}
	if len(locales) == 0 {
		locales = a.I18n.Locales
	}
	var done []string
	for _, loc := range locales {
		if loc == a.Cfg.DefaultLocale || !a.I18n.Has(loc) {
			continue
		}
		if cur, exists := trs[loc]; exists && cur.Source == "human" && (cur.Tagline != "" || cur.Description != "") && !force {
			continue
		}
		out, err := a.Translator.Fields(ctx, a.Cfg.DefaultLocale, loc, map[string]string{"tagline": src.Tagline, "description": src.Description})
		if err != nil {
			return done, fmt.Errorf("%s: %w", loc, err)
		}
		if err := a.Store.SaveTranslation(ctx, id, store.Translation{Locale: loc, Tagline: out["tagline"], Description: out["description"], Source: "machine"}); err != nil {
			return done, err
		}
		done = append(done, loc)
	}
	return done, nil
}

// TranslateCategory fills in missing category names/descriptions with machine translations.
func (a *Application) TranslateCategory(ctx context.Context, id int64, force bool) ([]string, error) {
	trs, err := a.Store.CategoryTranslations(ctx, id)
	if err != nil {
		return nil, err
	}
	src, ok := trs[a.Cfg.DefaultLocale]
	if !ok || src.Name == "" {
		return nil, fmt.Errorf("add the %s name first", a.Cfg.DefaultLocale)
	}
	cat, err := a.Store.CategoryByID(ctx, id, a.Cfg.DefaultLocale)
	if err != nil {
		return nil, err
	}
	var done []string
	for _, loc := range a.I18n.Locales {
		if loc == a.Cfg.DefaultLocale {
			continue
		}
		if cur, exists := trs[loc]; exists && cur.Name != "" && !force {
			continue
		}
		out, err := a.Translator.Fields(ctx, a.Cfg.DefaultLocale, loc, map[string]string{"name": src.Name, "description": src.Description})
		if err != nil {
			return done, fmt.Errorf("%s: %w", loc, err)
		}
		trs[loc] = store.CategoryTranslation{Locale: loc, Name: out["name"], Description: out["description"]}
		done = append(done, loc)
	}
	list := make([]store.CategoryTranslation, 0, len(trs))
	for _, t := range trs {
		list = append(list, t)
	}
	_, err = a.Store.SaveCategory(ctx, *cat, list)
	return done, err
}
