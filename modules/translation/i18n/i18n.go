// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package i18n

import (
	"errors"
	"fmt"
	"reflect"

	"code.gitea.io/gitea/modules/log"

	"gopkg.in/ini.v1"
)

var (
	ErrLocaleAlreadyExist = errors.New("lang already exists")

	DefaultLocales = NewLocaleStore()
)

type locale struct {
	store    *LocaleStore
	langName string
	textMap  map[int]string // the map key (idx) is generated by store's textIdxMap
}

type LocaleStore struct {
	// After initializing has finished, these fields are read-only.
	langNames []string
	langDescs []string

	localeMap  map[string]*locale
	textIdxMap map[string]int

	defaultLang string
}

func NewLocaleStore() *LocaleStore {
	return &LocaleStore{localeMap: make(map[string]*locale), textIdxMap: make(map[string]int)}
}

// AddLocaleByIni adds locale by ini into the store
// if source is a string, then the file is loaded
// if source is a []byte, then the content is used
func (ls *LocaleStore) AddLocaleByIni(langName, langDesc string, source interface{}) error {
	if _, ok := ls.localeMap[langName]; ok {
		return ErrLocaleAlreadyExist
	}
	iniFile, err := ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment:         true,
		UnescapeValueCommentSymbols: true,
	}, source)
	if err != nil {
		return fmt.Errorf("unable to load ini: %w", err)
	}
	iniFile.BlockMode = false

	// Common code between production and development.
	ls.langNames = append(ls.langNames, langName)
	ls.langDescs = append(ls.langDescs, langDesc)

	lc := &locale{store: ls, langName: langName, textMap: make(map[int]string)}
	ls.localeMap[lc.langName] = lc
	for _, section := range iniFile.Sections() {
		for _, key := range section.Keys() {
			var trKey string
			if section.Name() == "" || section.Name() == "DEFAULT" {
				trKey = key.Name()
			} else {
				trKey = section.Name() + "." + key.Name()
			}
			textIdx, ok := ls.textIdxMap[trKey]
			if !ok {
				textIdx = len(ls.textIdxMap)
				ls.textIdxMap[trKey] = textIdx
			}
			lc.textMap[textIdx] = key.Value()
		}
	}
	iniFile = nil
	return nil
}

func (ls *LocaleStore) HasLang(langName string) bool {
	_, ok := ls.localeMap[langName]
	return ok
}

func (ls *LocaleStore) ListLangNameDesc() (names, desc []string) {
	return ls.langNames, ls.langDescs
}

// SetDefaultLang sets default language as a fallback
func (ls *LocaleStore) SetDefaultLang(lang string) {
	ls.defaultLang = lang
}

// Tr translates content to target language. fall back to default language.
func (ls *LocaleStore) Tr(lang, trKey string, trArgs ...interface{}) string {
	l, ok := ls.localeMap[lang]
	if !ok {
		l, ok = ls.localeMap[ls.defaultLang]
	}
	if ok {
		return l.Tr(trKey, trArgs...)
	}
	return trKey
}

// Tr translates content to locale language. fall back to default language.
func (l *locale) Tr(trKey string, trArgs ...interface{}) string {
	trMsg := trKey
	textIdx, ok := l.store.textIdxMap[trKey]
	if ok {
		if msg, ok := l.textMap[textIdx]; ok {
			trMsg = msg // use current translation
		} else if def, ok := l.store.localeMap[l.store.defaultLang]; ok {
			// try to use default locale's translation
			if msg, ok := def.textMap[textIdx]; ok {
				trMsg = msg
			}
		}
	}

	if len(trArgs) > 0 {
		fmtArgs := make([]interface{}, 0, len(trArgs))
		for _, arg := range trArgs {
			val := reflect.ValueOf(arg)
			if val.Kind() == reflect.Slice {
				// before, it can accept Tr(lang, key, a, [b, c], d, [e, f]) as Sprintf(msg, a, b, c, d, e, f), it's an unstable behavior
				// now, we restrict the strange behavior and only support:
				// 1. Tr(lang, key, [slice-items]) as Sprintf(msg, items...)
				// 2. Tr(lang, key, args...) as Sprintf(msg, args...)
				if len(trArgs) == 1 {
					for i := 0; i < val.Len(); i++ {
						fmtArgs = append(fmtArgs, val.Index(i).Interface())
					}
				} else {
					log.Error("the args for i18n shouldn't contain uncertain slices, key=%q, args=%v", trKey, trArgs)
					break
				}
			} else {
				fmtArgs = append(fmtArgs, arg)
			}
		}
		return fmt.Sprintf(trMsg, fmtArgs...)
	}
	return trMsg
}

func ResetDefaultLocales() {
	DefaultLocales = NewLocaleStore()
}

// Tr use default locales to translate content to target language.
func Tr(lang, trKey string, trArgs ...interface{}) string {
	return DefaultLocales.Tr(lang, trKey, trArgs...)
}
