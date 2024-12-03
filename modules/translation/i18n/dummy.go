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

// HasKey implements Locale.
func (k *KeyLocale) HasKey(trKey string) bool {
	return true
}

// TrHTML implements Locale.
func (k *KeyLocale) TrHTML(trKey string, trArgs ...any) template.HTML {
	args := slices.Clone(trArgs)
	for i, v := range args {
		switch v := v.(type) {
		case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, template.HTML:
			// for most basic types (including template.HTML which is safe), just do nothing and use it
		case string:
			args[i] = template.HTMLEscapeString(v)
		case fmt.Stringer:
			args[i] = template.HTMLEscapeString(v.String())
		default:
			args[i] = template.HTMLEscapeString(fmt.Sprint(v))
		}
	}
	return template.HTML(k.TrString(trKey, args...))
}

// TrString implements Locale.
func (k *KeyLocale) TrString(trKey string, trArgs ...any) string {
	return FormatDummy(trKey, trArgs...)
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
