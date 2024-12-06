// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

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
	case "en", "de", "nds", "nl", "sv", "da", "no", "nn", "nb", "fo", "es", "it", "el", "bg", "fi", "et", "he", "id", "eo", "hu", "tr":
		return PluralRuleDefault

	case "fr", "pt":
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

	case "ru", "uk", "be", "sr", "hr":
		return PluralRuleRussian

	case "cs", "sk":
		return PluralRuleCzech

	default:
		break
	}

	log.Error("No plural rule defined for language %s", langName)
	return PluralRuleDefault
}
