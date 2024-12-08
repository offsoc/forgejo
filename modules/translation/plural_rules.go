// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Some useful links:
// https://www.unicode.org/cldr/charts/46/supplemental/language_plural_rules.html
// https://www.gnu.org/software/gettext/manual/gettext.html#index-ngettext
// https://translate.codeberg.org/languages/$LANGUAGE_CODE/#information
// Note that in some cases there is ambiguity about the correct form for a given language. In this case, ask the locale's translators.

package translation

import (
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/translation/i18n"
)

/* The constants refer to indices below in `PluralRules` and also in i18n.js, keep them in sync! */
const (
	PluralRuleDefault   = 0
	PluralRuleFrench    = 1
	PluralRuleOne       = 2
	PluralRuleLatvian   = 3
	PluralRuleIrish     = 4
	PluralRuleRomanian  = 5
	PluralRuleLithunian = 6
	PluralRuleRussian   = 7
	PluralRuleCzech     = 8
	PluralRulePolish    = 9
	PluralRuleSlovenian = 10
	PluralRuleArabic    = 11
)

func GetPluralRuleImpl(langName string) int {
	langName = strings.Split(strings.Split(langName, "_")[0], "-")[0]

	/* When adding a new language not in the list, add its plural rule definition here. */
	switch langName {
	case "bg", "bn", "ca", "da", "de", "el", "en", "eo", "es", "et", "fi", "fo", "fy", "gl", "he", "hu", "id", "is", "it", "ml", "nb", "nds", "nl", "nn", "no", "sv", "tr":
		return PluralRuleDefault

	case "fil", "fr", "hi", "pt", "si":
		return PluralRuleFrench

	case "lv":
		return PluralRuleLatvian
	case "ga":
		return PluralRuleIrish
	case "ro":
		return PluralRuleRomanian
	case "lt":
		return PluralRuleLithunian
	case "pl":
		return PluralRulePolish
	case "sl":
		return PluralRuleSlovenian
	case "ar":
		return PluralRuleArabic

	case "be", "bs", "hr", "ru", "sr", "uk":
		return PluralRuleRussian

	case "cs", "sk":
		return PluralRuleCzech

	case "fa", "ja", "ko", "vi", "yi", "zh":
		return PluralRuleOne

	default:
		break
	}

	log.Error("No plural rule defined for language %s", langName)
	return PluralRuleDefault
}

var PluralRules = []i18n.PluralFormRule{
	// [ 0] Common 2-form, e.g. English, German
	func(n int) int {
		if n == 1 {
			return i18n.PluralFormOne
		}
		return i18n.PluralFormOther
	},

	// [ 1] French 2-form
	func(n int) int {
		if n == 0 || n == 1 {
			return i18n.PluralFormOne
		}
		return i18n.PluralFormOther
	},

	// [ 2] One-Form
	func(n int) int {
		return i18n.PluralFormOther
	},

	// [ 3] Latvian 3-form
	func(n int) int {
		if n == 0 {
			return i18n.PluralFormZero
		}
		if n%10 == 1 && n%100 != 11 {
			return i18n.PluralFormOne
		}
		return i18n.PluralFormOther
	},

	// [ 4] Irish 3-form
	func(n int) int {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n == 2 {
			return i18n.PluralFormTwo
		}
		return i18n.PluralFormOther
	},

	// [ 5] Romanian 3-form
	func(n int) int {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n == 0 || (n%100 > 0 && n%100 < 20) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormOther
	},

	// [ 6] Lithunian 3-form
	func(n int) int {
		if n%10 == 1 && n%100 != 11 {
			return i18n.PluralFormOne
		}
		if n%10 >= 2 && (n%100 < 10 || n%100 >= 20) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormOther
	},

	// [ 7] Russian 3-form
	func(n int) int {
		if n%10 == 1 && n%100 != 11 {
			return i18n.PluralFormOne
		}
		if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormMany
	},

	// [ 8] Czech 3-form
	func(n int) int {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n >= 2 && n <= 4 {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormOther
	},

	// [ 9] Polish 3-form
	func(n int) int {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormMany
	},

	// [10] Slovenian 4-form
	func(n int) int {
		if n%100 == 1 {
			return i18n.PluralFormOne
		}
		if n%100 == 2 {
			return i18n.PluralFormTwo
		}
		if n%100 == 3 || n%100 == 4 {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormOther
	},

	// [11] Arabic 6-form
	func(n int) int {
		if n == 0 {
			return i18n.PluralFormZero
		}
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n == 2 {
			return i18n.PluralFormTwo
		}
		if n%100 >= 3 && n%100 <= 10 {
			return i18n.PluralFormFew
		}
		if n%100 >= 11 {
			return i18n.PluralFormMany
		}
		return i18n.PluralFormOther
	},
}
