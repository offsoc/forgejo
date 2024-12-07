// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Some useful links:
// https://www.unicode.org/cldr/cldr-aux/charts/22/supplemental/language_plural_rules.html
// https://www.gnu.org/software/gettext/manual/gettext.html#index-ngettext
// https://translate.codeberg.org/languages/$LANGUAGE_CODE/#information
// Note that in some cases there is ambiguity about the correct form for a given language. In this case, ask the locale's translators.

package translation

import (
	"strings"

	"code.gitea.io/gitea/modules/log"
)

/* The constants refer to indices in i18n.js, keep them in sync! */
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
