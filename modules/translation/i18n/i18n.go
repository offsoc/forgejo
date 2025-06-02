// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"html/template"
	"io"
)

type (
	PluralFormIndex uint8
	PluralFormRule  func(int64) PluralFormIndex
)

const (
	PluralFormZero PluralFormIndex = iota
	PluralFormOne
	PluralFormTwo
	PluralFormFew
	PluralFormMany
	PluralFormOther
)

var DefaultLocales = NewLocaleStore()

type Locale interface {
	Language() string
	// TrString translates a given key and arguments for a language
	TrString(trKey string, trArgs ...any) string
	// TrPluralString translates a given pluralized key and arguments for a language.
	// This function returns an error if new-style support for the given key is not available.
	TrPluralString(count any, trKey string, trArgs ...any) template.HTML
	// TrHTML translates a given key and arguments for a language, string arguments are escaped to HTML
	TrHTML(trKey string, trArgs ...any) template.HTML
	// HasKey reports if a locale has a translation for a given key
	HasKey(trKey string) bool
	// TrPluralStringAllForms returns all plural form variants for a given string, and also
	// the fallbacks for the default language if the translation is incomplete.
	TrPluralStringAllForms(trKey string) ([]string, []string)
}

// LocaleStore provides the functions common to all locale stores
type LocaleStore interface {
	io.Closer

	// SetDefaultLang sets the default language to fall back to
	SetDefaultLang(lang string)
	// GetDefaultLang returns the name of the default language to fall back to
	GetDefaultLang() string
	// ListLangNameDesc provides paired slices of language names to descriptors
	ListLangNameDesc() (names, desc []string)
	// Locale return the locale for the provided language or the default language if not found
	Locale(langName string) (Locale, bool)
	// HasLang returns whether a given language is present in the store
	HasLang(langName string) bool
	// AddLocaleByIni adds a new old-style language to the store
	AddLocaleByIni(langName, langDesc string, pluralRule PluralFormRule, usedPluralForms []PluralFormIndex, source, moreSource []byte) error
	// AddLocaleByJSON adds new-style content to an existing language to the store
	AddToLocaleFromJSON(langName string, source []byte) error
}

// ResetDefaultLocales resets the current default locales
// NOTE: this is not synchronized
func ResetDefaultLocales() {
	if DefaultLocales != nil {
		_ = DefaultLocales.Close()
	}
	DefaultLocales = NewLocaleStore()
}

// GetLocale returns the locale from the default locales
func GetLocale(lang string) (Locale, bool) {
	return DefaultLocales.Locale(lang)
}
