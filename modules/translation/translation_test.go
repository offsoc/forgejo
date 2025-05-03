// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package translation

// TODO: make this package friendly to testing

import (
	"testing"

	"forgejo.org/modules/translation/i18n"

	"github.com/stretchr/testify/assert"
)

func TestTrSize(t *testing.T) {
	l := NewLocale("")
	size := int64(1)
	assert.Equal(t, "1 munits.data.b", l.TrSize(size).String())
	size *= 2048
	assert.Equal(t, "2 munits.data.kib", l.TrSize(size).String())
	size *= 2048
	assert.Equal(t, "4 munits.data.mib", l.TrSize(size).String())
	size *= 2048
	assert.Equal(t, "8 munits.data.gib", l.TrSize(size).String())
	size *= 2048
	assert.Equal(t, "16 munits.data.tib", l.TrSize(size).String())
	size *= 2048
	assert.Equal(t, "32 munits.data.pib", l.TrSize(size).String())
	size *= 128
	assert.Equal(t, "4 munits.data.eib", l.TrSize(size).String())
}

func TestPrettyNumber(t *testing.T) {
	i18n.ResetDefaultLocales()

	allLangMap = make(map[string]*LangType)
	allLangMap["id-ID"] = &LangType{Lang: "id-ID", Name: "Bahasa Indonesia"}

	l := NewLocale("id-ID")
	assert.Equal(t, "1.000.000", l.PrettyNumber(1000000))
	assert.Equal(t, "1.000.000,1", l.PrettyNumber(1000000.1))
	assert.Equal(t, "1.000.000", l.PrettyNumber("1000000"))
	assert.Equal(t, "1.000.000", l.PrettyNumber("1000000.0"))
	assert.Equal(t, "1.000.000,1", l.PrettyNumber("1000000.1"))

	l = NewLocale("nosuch")
	assert.Equal(t, "1,000,000", l.PrettyNumber(1000000))
	assert.Equal(t, "1,000,000.1", l.PrettyNumber(1000000.1))
}

func TestGetPluralRule(t *testing.T) {
	assert.Equal(t, PluralRuleDefault, GetPluralRuleImpl("en"))
	assert.Equal(t, PluralRuleDefault, GetPluralRuleImpl("en-US"))
	assert.Equal(t, PluralRuleDefault, GetPluralRuleImpl("en_UK"))
	assert.Equal(t, PluralRuleDefault, GetPluralRuleImpl("nds"))
	assert.Equal(t, PluralRuleDefault, GetPluralRuleImpl("de-DE"))

	assert.Equal(t, PluralRuleOneForm, GetPluralRuleImpl("zh"))
	assert.Equal(t, PluralRuleOneForm, GetPluralRuleImpl("ja"))

	assert.Equal(t, PluralRuleBengali, GetPluralRuleImpl("bn"))

	assert.Equal(t, PluralRuleIcelandic, GetPluralRuleImpl("is"))

	assert.Equal(t, PluralRuleFilipino, GetPluralRuleImpl("fil"))

	assert.Equal(t, PluralRuleCzech, GetPluralRuleImpl("cs"))

	assert.Equal(t, PluralRuleRussian, GetPluralRuleImpl("ru"))

	assert.Equal(t, PluralRulePolish, GetPluralRuleImpl("pl"))

	assert.Equal(t, PluralRuleLatvian, GetPluralRuleImpl("lv"))

	assert.Equal(t, PluralRuleLithuanian, GetPluralRuleImpl("lt"))

	assert.Equal(t, PluralRuleFrench, GetPluralRuleImpl("fr"))

	assert.Equal(t, PluralRuleCatalan, GetPluralRuleImpl("ca"))

	assert.Equal(t, PluralRuleSlovenian, GetPluralRuleImpl("sl"))

	assert.Equal(t, PluralRuleArabic, GetPluralRuleImpl("ar"))

	assert.Equal(t, PluralRuleCatalan, GetPluralRuleImpl("pt-PT"))
	assert.Equal(t, PluralRuleFrench, GetPluralRuleImpl("pt-BR"))

	assert.Equal(t, PluralRuleDefault, GetPluralRuleImpl("invalid"))
}

func TestApplyPluralRule(t *testing.T) {
	testCases := []struct {
		expect     i18n.PluralFormIndex
		pluralRule int
		values     []int64
	}{
		{i18n.PluralFormOne, PluralRuleDefault, []int64{1}},
		{i18n.PluralFormOther, PluralRuleDefault, []int64{0, 2, 10, 256}},

		{i18n.PluralFormOther, PluralRuleOneForm, []int64{0, 1, 2}},

		{i18n.PluralFormOne, PluralRuleBengali, []int64{0, 1}},
		{i18n.PluralFormOther, PluralRuleBengali, []int64{2, 10, 256}},

		{i18n.PluralFormOne, PluralRuleIcelandic, []int64{1, 21, 31}},
		{i18n.PluralFormOther, PluralRuleIcelandic, []int64{0, 2, 11, 15, 256}},

		{i18n.PluralFormOne, PluralRuleFilipino, []int64{0, 1, 2, 3, 5, 7, 8, 10, 11, 12, 257}},
		{i18n.PluralFormOther, PluralRuleFilipino, []int64{4, 6, 9, 14, 16, 19, 256}},

		{i18n.PluralFormOne, PluralRuleCzech, []int64{1}},
		{i18n.PluralFormFew, PluralRuleCzech, []int64{2, 3, 4}},
		{i18n.PluralFormOther, PluralRuleCzech, []int64{5, 0, 12, 78, 254}},

		{i18n.PluralFormOne, PluralRuleRussian, []int64{1, 21, 31}},
		{i18n.PluralFormFew, PluralRuleRussian, []int64{2, 23, 34}},
		{i18n.PluralFormMany, PluralRuleRussian, []int64{0, 5, 11, 37, 111, 256}},

		{i18n.PluralFormOne, PluralRulePolish, []int64{1}},
		{i18n.PluralFormFew, PluralRulePolish, []int64{2, 23, 34}},
		{i18n.PluralFormMany, PluralRulePolish, []int64{0, 5, 11, 21, 37, 256}},

		{i18n.PluralFormZero, PluralRuleLatvian, []int64{0, 10, 11, 17}},
		{i18n.PluralFormOne, PluralRuleLatvian, []int64{1, 21, 71}},
		{i18n.PluralFormOther, PluralRuleLatvian, []int64{2, 7, 22, 23, 256}},

		{i18n.PluralFormOne, PluralRuleLithuanian, []int64{1, 21, 31}},
		{i18n.PluralFormFew, PluralRuleLithuanian, []int64{2, 5, 9, 23, 34, 256}},
		{i18n.PluralFormMany, PluralRuleLithuanian, []int64{0, 10, 11, 18}},

		{i18n.PluralFormOne, PluralRuleFrench, []int64{0, 1}},
		{i18n.PluralFormMany, PluralRuleFrench, []int64{1000000, 2000000}},
		{i18n.PluralFormOther, PluralRuleFrench, []int64{2, 4, 10, 256}},

		{i18n.PluralFormOne, PluralRuleCatalan, []int64{1}},
		{i18n.PluralFormMany, PluralRuleCatalan, []int64{1000000, 2000000}},
		{i18n.PluralFormOther, PluralRuleCatalan, []int64{0, 2, 4, 10, 256}},

		{i18n.PluralFormOne, PluralRuleSlovenian, []int64{1, 101, 201, 501}},
		{i18n.PluralFormTwo, PluralRuleSlovenian, []int64{2, 102, 202, 502}},
		{i18n.PluralFormFew, PluralRuleSlovenian, []int64{3, 103, 203, 503, 4, 104, 204, 504}},
		{i18n.PluralFormOther, PluralRuleSlovenian, []int64{0, 5, 11, 12, 20, 256}},

		{i18n.PluralFormZero, PluralRuleArabic, []int64{0}},
		{i18n.PluralFormOne, PluralRuleArabic, []int64{1}},
		{i18n.PluralFormTwo, PluralRuleArabic, []int64{2}},
		{i18n.PluralFormFew, PluralRuleArabic, []int64{3, 4, 9, 10, 103, 104}},
		{i18n.PluralFormMany, PluralRuleArabic, []int64{11, 12, 13, 14, 17, 111, 256}},
		{i18n.PluralFormOther, PluralRuleArabic, []int64{100, 101, 102}},
	}

	for _, tc := range testCases {
		for _, n := range tc.values {
			assert.Equal(t, tc.expect, PluralRules[tc.pluralRule](n), "Testcase for plural rule %d, value %d", tc.pluralRule, n)
		}
	}
}
