// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"fmt"
	"html/template"
	"reflect"
	"slices"
	"strings"
)

type KeyLocale struct{}

var _ Locale = (*KeyLocale)(nil)

func (k *KeyLocale) Language() string {
	return "dummy"
}

// HasKey implements Locale.
func (k *KeyLocale) HasKey(trKey string) bool {
	return true
}

// TrHTML implements Locale.
func (k *KeyLocale) TrHTML(trKey string, trArgs ...any) template.HTML {
	return template.HTML(k.TrString(trKey, PrepareArgsForHTML(trArgs...)...))
}

// TrString implements Locale.
func (k *KeyLocale) TrString(trKey string, trArgs ...any) string {
	return FormatDummy(trKey, trArgs...)
}

// TrPluralString implements Locale.
func (k *KeyLocale) TrPluralString(count any, trKey string, trArgs ...any) template.HTML {
	return template.HTML(FormatDummy(trKey, PrepareArgsForHTML(trArgs...)...))
}

// TrPluralStringAllForms implements Locale.
func (k *KeyLocale) TrPluralStringAllForms(trKey string) ([]string, []string) {
	return []string{trKey}, nil
}

func FormatDummy(trKey string, args ...any) string {
	if len(args) == 0 {
		return fmt.Sprintf("(%s)", trKey)
	}

	fmtArgs := make([]any, 0, len(args)+1)
	fmtArgs = append(fmtArgs, trKey)
	for _, arg := range args {
		val := reflect.ValueOf(arg)
		if val.Kind() == reflect.Slice {
			for i := 0; i < val.Len(); i++ {
				fmtArgs = append(fmtArgs, val.Index(i).Interface())
			}
		} else {
			fmtArgs = append(fmtArgs, arg)
		}
	}

	template := fmt.Sprintf("(%%s: %s)", strings.Join(slices.Repeat([]string{"%v"}, len(fmtArgs)-1), ", "))
	return fmt.Sprintf(template, fmtArgs...)
}
