// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package production

import (
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/translation/i18n/common"
)

// This file implements the production LocaleStore

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

// NewLocaleStore creates a production locale store
func NewLocaleStore() *LocaleStore {
	return &LocaleStore{localeMap: make(map[string]*locale), textIdxMap: make(map[string]int)}
}

// AddLocaleByIni adds locale by ini into the store
// if source is a string, then the file is loaded
// if source is a []byte, then the content is used
func (ls *LocaleStore) AddLocaleByIni(langName, langDesc string, source interface{}) error {
	if _, ok := ls.localeMap[langName]; ok {
		return common.ErrLocaleAlreadyExist
	}

	ls.langNames = append(ls.langNames, langName)
	ls.langDescs = append(ls.langDescs, langDesc)

	lc := &locale{store: ls, langName: langName, textMap: make(map[int]string)}
	ls.localeMap[lc.langName] = lc

	return common.AddFromIni(common.AddableFn(func(key, value string) {
		textIdx, ok := ls.textIdxMap[key]
		if !ok {
			textIdx = len(ls.textIdxMap)
			ls.textIdxMap[key] = textIdx
		}
		lc.textMap[textIdx] = value
	}), source)
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

	msg, err := common.Format(trMsg, trArgs...)
	if err != nil {
		log.Error("Error whilst formatting %q in %s: %v", trKey, l.langName, err)
	}
	return msg
}

func (ls *LocaleStore) Close() error {
	return nil
}
