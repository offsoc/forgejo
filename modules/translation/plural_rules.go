// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Some useful links:
// https://www.unicode.org/cldr/charts/46/supplemental/language_plural_rules.html
// https://translate.codeberg.org/languages/$LANGUAGE_CODE/#information
// https://github.com/WeblateOrg/language-data/blob/main/languages.csv
// Note that in some cases there is ambiguity about the correct form for a given language. In this case, ask the locale's translators.

package translation

import (
	"strings"

	"forgejo.org/modules/log"
	"forgejo.org/modules/translation/i18n"
)

// The constants refer to indices below in `PluralRules` and also in i18n.js, keep them in sync!
const (
	PluralRuleDefault    = 0
	PluralRuleBengali    = 1
	PluralRuleIcelandic  = 2
	PluralRuleFilipino   = 3
	PluralRuleOneForm    = 4
	PluralRuleCzech      = 5
	PluralRuleRussian    = 6
	PluralRulePolish     = 7
	PluralRuleLatvian    = 8
	PluralRuleLithuanian = 9
	PluralRuleFrench     = 10
	PluralRuleCatalan    = 11
	PluralRuleSlovenian  = 12
	PluralRuleArabic     = 13
)

func GetPluralRuleImpl(langName string) int {
	// First, check for languages with country-specific plural rules.
	switch langName {
	case "pt-BR":
		return PluralRuleFrench

	case "pt-PT":
		return PluralRuleCatalan

	default:
		break
	}

	// Remove the country portion of the locale name.
	langName = strings.Split(strings.Split(langName, "_")[0], "-")[0]

	// When adding a new language not in the list, add its plural rule definition here.
	switch langName {
	case "en", "aa", "ab", "abr", "ada", "ae", "aeb", "af", "afh", "aii", "ain", "akk", "ale", "aln", "alt", "ami", "an", "ang", "anp", "apc", "arc", "arp", "arq", "arw", "arz", "asa", "ast", "av", "avk", "awa", "ayc", "az", "azb", "ba", "bal", "ban", "bar", "bas", "bbc", "bci", "bej", "bem", "ber", "bew", "bez", "bg", "bgc", "bgn", "bhb", "bhi", "bi", "bik", "bin", "bjj", "bjn", "bla", "bnt", "bqi", "bra", "brb", "brh", "brx", "bua", "bug", "bum", "byn", "cad", "cak", "car", "ce", "cgg", "ch", "chb", "chg", "chk", "chm", "chn", "cho", "chp", "chr", "chy", "ckb", "co", "cop", "cpe", "cpf", "cr", "crp", "cu", "cv", "da", "dak", "dar", "dcc", "de", "del", "den", "dgr", "din", "dje", "dnj", "dnk", "dru", "dry", "dua", "dum", "dv", "dyu", "ee", "efi", "egl", "egy", "eka", "el", "elx", "enm", "eo", "et", "eu", "ewo", "ext", "fan", "fat", "fbl", "ffm", "fi", "fj", "fo", "fon", "frk", "frm", "fro", "frr", "frs", "fuq", "fur", "fuv", "fvr", "fy", "gaa", "gay", "gba", "gbm", "gez", "gil", "gl", "glk", "gmh", "gn", "goh", "gom", "gon", "gor", "got", "grb", "gsw", "guc", "gum", "gur", "guz", "gwi", "ha", "hai", "haw", "haz", "hil", "hit", "hmn", "hnd", "hne", "hno", "ho", "hoc", "hoj", "hrx", "ht", "hu", "hup", "hus", "hz", "ia", "iba", "ibb", "ie", "ik", "ilo", "inh", "io", "jam", "jgo", "jmc", "jpr", "jrb", "ka", "kaa", "kac", "kaj", "kam", "kaw", "kbd", "kcg", "kfr", "kfy", "kg", "kha", "khn", "kho", "ki", "kj", "kk", "kkj", "kl", "kln", "kmb", "kmr", "kok", "kpe", "kr", "krc", "kri", "krl", "kru", "ks", "ksb", "ku", "kum", "kut", "kv", "kxm", "ky", "la", "lad", "laj", "lam", "lb", "lez", "lfn", "lg", "li", "lij", "ljp", "lki", "lmn", "lmo", "lol", "loz", "lrc", "lu", "lua", "lui", "lun", "luo", "lus", "luy", "luz", "mad", "mag", "mai", "mak", "man", "mas", "mdf", "mdh", "mdr", "men", "mer", "mfa", "mga", "mgh", "mgo", "mh", "mhr", "mic", "min", "mjw", "ml", "mn", "mnc", "mni", "mnw", "moe", "moh", "mos", "mr", "mrh", "mtr", "mus", "mwk", "mwl", "mwr", "mxc", "myv", "myx", "mzn", "na", "nah", "nap", "nb", "nd", "ndc", "nds", "ne", "new", "ng", "ngl", "nia", "nij", "niu", "nl", "nn", "nnh", "nod", "noe", "nog", "non", "nr", "nuk", "nv", "nwc", "ny", "nym", "nyn", "nyo", "nzi", "oj", "om", "or", "os", "ota", "otk", "ovd", "pag", "pal", "pam", "pap", "pau", "pbb", "pdt", "peo", "phn", "pi", "pms", "pon", "pro", "ps", "pwn", "qu", "quc", "qug", "qya", "raj", "rap", "rar", "rcf", "rej", "rhg", "rif", "rkt", "rm", "rmt", "rn", "rng", "rof", "rom", "rue", "rup", "rw", "rwk", "sad", "sai", "sam", "saq", "sas", "sc", "sck", "sco", "sd", "sdh", "sef", "seh", "sel", "sga", "sgn", "sgs", "shn", "sid", "sjd", "skr", "sm", "sml", "sn", "snk", "so", "sog", "sou", "sq", "srn", "srr", "ss", "ssy", "st", "suk", "sus", "sux", "sv", "sw", "swg", "swv", "sxu", "syc", "syl", "syr", "szy", "ta", "tay", "tcy", "te", "tem", "teo", "ter", "tet", "tig", "tiv", "tk", "tkl", "tli", "tly", "tmh", "tn", "tog", "tr", "trv", "ts", "tsg", "tsi", "tsj", "tts", "tum", "tvl", "tw", "ty", "tyv", "tzj", "tzl", "udm", "ug", "uga", "umb", "und", "unr", "ur", "uz", "vai", "ve", "vls", "vmf", "vmw", "vo", "vot", "vro", "vun", "wae", "wal", "war", "was", "wbq", "wbr", "wep", "wtm", "xal", "xh", "xnr", "xog", "yao", "yap", "yi", "yua", "za", "zap", "zbl", "zen", "zgh", "zun", "zza":
		return PluralRuleDefault

	case "ach", "ady", "ak", "am", "arn", "as", "bh", "bho", "bn", "csw", "doi", "fa", "ff", "frc", "frp", "gu", "gug", "gun", "guw", "hi", "hy", "kab", "kn", "ln", "mfe", "mg", "mi", "mia", "nso", "oc", "pa", "pcm", "pt", "qdt", "qtp", "si", "tg", "ti", "wa", "zu":
		return PluralRuleBengali

	case "is":
		return PluralRuleIcelandic

	case "fil":
		return PluralRuleFilipino

	case "ace", "ay", "bm", "bo", "cdo", "cpx", "crh", "dz", "gan", "hak", "hnj", "hsn", "id", "ig", "ii", "ja", "jbo", "jv", "kde", "kea", "km", "ko", "kos", "lkt", "lo", "lzh", "ms", "my", "nan", "nqo", "osa", "sah", "ses", "sg", "son", "su", "th", "tlh", "to", "tok", "tpi", "tt", "vi", "wo", "wuu", "yo", "yue", "zh":
		return PluralRuleOneForm

	case "cpp", "cs", "sk":
		return PluralRuleCzech

	case "be", "bs", "cnr", "hr", "ru", "sr", "uk", "wen":
		return PluralRuleRussian

	case "csb", "pl", "szl":
		return PluralRulePolish

	case "lv", "prg":
		return PluralRuleLatvian

	case "lt":
		return PluralRuleLithuanian

	case "fr":
		return PluralRuleFrench

	case "ca", "es", "it":
		return PluralRuleCatalan

	case "sl":
		return PluralRuleSlovenian

	case "ar":
		return PluralRuleArabic

	default:
		break
	}

	log.Error("No plural rule defined for language %s", langName)
	return PluralRuleDefault
}

var PluralRules = []i18n.PluralFormRule{
	// [ 0] Common 2-form, e.g. English, German
	func(n int64) i18n.PluralFormIndex {
		if n != 1 {
			return i18n.PluralFormOther
		}
		return i18n.PluralFormOne
	},

	// [ 1] Bengali
	func(n int64) i18n.PluralFormIndex {
		if n > 1 {
			return i18n.PluralFormOther
		}
		return i18n.PluralFormOne
	},

	// [ 2] Icelandic
	func(n int64) i18n.PluralFormIndex {
		if n%10 != 1 || n%100 == 11 {
			return i18n.PluralFormOther
		}
		return i18n.PluralFormOne
	},

	// [ 3] Filipino
	func(n int64) i18n.PluralFormIndex {
		if n != 1 && n != 2 && n != 3 && (n%10 == 4 || n%10 == 6 || n%10 == 9) {
			return i18n.PluralFormOther
		}
		return i18n.PluralFormOne
	},

	// [ 4] OneForm
	func(n int64) i18n.PluralFormIndex {
		return i18n.PluralFormOther
	},

	// [ 5] Czech
	func(n int64) i18n.PluralFormIndex {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n >= 2 && n <= 4 {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormOther
	},

	// [ 6] Russian
	func(n int64) i18n.PluralFormIndex {
		if n%10 == 1 && n%100 != 11 {
			return i18n.PluralFormOne
		}
		if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormMany
	},

	// [ 7] Polish
	func(n int64) i18n.PluralFormIndex {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormMany
	},

	// [ 8] Latvian
	func(n int64) i18n.PluralFormIndex {
		if n%10 == 0 || n%100 >= 11 && n%100 <= 19 {
			return i18n.PluralFormZero
		}
		if n%10 == 1 && n%100 != 11 {
			return i18n.PluralFormOne
		}
		return i18n.PluralFormOther
	},

	// [ 9] Lithuanian
	func(n int64) i18n.PluralFormIndex {
		if n%10 == 1 && (n%100 < 11 || n%100 > 19) {
			return i18n.PluralFormOne
		}
		if n%10 >= 2 && n%10 <= 9 && (n%100 < 11 || n%100 > 19) {
			return i18n.PluralFormFew
		}
		return i18n.PluralFormMany
	},

	// [10] French
	func(n int64) i18n.PluralFormIndex {
		if n == 0 || n == 1 {
			return i18n.PluralFormOne
		}
		if n != 0 && n%1000000 == 0 {
			return i18n.PluralFormMany
		}
		return i18n.PluralFormOther
	},

	// [11] Catalan
	func(n int64) i18n.PluralFormIndex {
		if n == 1 {
			return i18n.PluralFormOne
		}
		if n != 0 && n%1000000 == 0 {
			return i18n.PluralFormMany
		}
		return i18n.PluralFormOther
	},

	// [12] Slovenian
	func(n int64) i18n.PluralFormIndex {
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

	// [13] Arabic
	func(n int64) i18n.PluralFormIndex {
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

var UsedPluralForms = [][]i18n.PluralFormIndex{
	// [ 0] Common 2-form, e.g. English, German
	{i18n.PluralFormOne, i18n.PluralFormOther},
	// [ 1] Bengali
	{i18n.PluralFormOne, i18n.PluralFormOther},
	// [ 2] Icelandic
	{i18n.PluralFormOne, i18n.PluralFormOther},
	// [ 3] Filipino
	{i18n.PluralFormOne, i18n.PluralFormOther},
	// [ 4] OneForm
	{i18n.PluralFormOther},
	// [ 5] Czech
	{i18n.PluralFormOne, i18n.PluralFormFew, i18n.PluralFormOther},
	// [ 6] Russian
	{i18n.PluralFormOne, i18n.PluralFormFew, i18n.PluralFormMany},
	// [ 7] Polish
	{i18n.PluralFormOne, i18n.PluralFormFew, i18n.PluralFormOther},
	// [ 8] Latvian
	{i18n.PluralFormZero, i18n.PluralFormOne, i18n.PluralFormOther},
	// [ 9] Lithuanian
	{i18n.PluralFormOne, i18n.PluralFormFew, i18n.PluralFormMany},
	// [10] French
	{i18n.PluralFormOne, i18n.PluralFormMany, i18n.PluralFormOther},
	// [11] Catalan
	{i18n.PluralFormOne, i18n.PluralFormMany, i18n.PluralFormOther},
	// [12] Slovenian
	{i18n.PluralFormOne, i18n.PluralFormTwo, i18n.PluralFormFew, i18n.PluralFormOther},
	// [13] Arabic
	{i18n.PluralFormZero, i18n.PluralFormOne, i18n.PluralFormTwo, i18n.PluralFormFew, i18n.PluralFormMany, i18n.PluralFormOther},
}
