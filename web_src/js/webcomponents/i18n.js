const {pageData} = window.config;

/** A list of plural rules for all languages. `pageData.PLURAL_RULE_LANG` and `pageData.PLURAL_RULE_FALLBACK` are indices into this list (defined in localestore.go). */
const PLURAL_RULES = [
  function (n) { return n === 1 ? 0 : 1 },  // [ 0] Common 2-form, e.g. English, German
  function (n) { return n <= 1 ? 0 : 1 },   // [ 1] French 2-form
  function (_) { return 0 },                // [ 2] One-Form
  function (n) { return n % 10 === 1 && n % 100 !== 11 ? 0 : n !== 0 ? 1 : 2 },              // [ 3] Latvian 3-form
  function (n) { return n === 1 ? 0 : n === 2 ? 1 : 2 },                                     // [ 4] Irish 3-form
  function (n) { return n === 1 ? 0 : (n === 0 || (n % 100 > 0 && n % 100 < 20)) ? 1 : 2 },  // [ 5] Romanian 3-form
  function (n) { return n % 10 === 1 && n % 100 !== 11 ? 0 : n % 10 >= 2 && (n % 100 < 10 || n % 100 >= 20) ? 1 : 2 },                 // [ 6] Lithunian 3-form
  function (n) { return n % 10 === 1 && n % 100 !== 11 ? 0 : n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 10 || n % 100 >= 20) ? 1 : 2 },  // [ 7] Russian 3-form
  function (n) { return (n === 1) ? 0 : (n >= 2 && n <= 4) ? 1 : 2 },                                                                  // [ 8] Czech 3-form
  function (n) { return n === 1 ? 0 : n % 10 >= 2 && n % 10 <= 4 && (n % 100 < 10 || n % 100 >= 20) ? 1 : 2 },                         // [ 9] Polish 3-form
  function (n) { return n % 100 === 1 ? 0 : n % 100 === 2 ? 1 : n % 100 === 3 || n % 100 === 4 ? 2 : 3 },                              // [10] Slovenian 4-form
  function (n) { return n === 0 ? 0 : n === 1 ? 1 : n === 2 ? 2 : n % 100 >= 3 && n % 100 <= 10 ? 3 : n % 100 >= 11 ? 4 : 5 },         // [11] Arabic 6-form
];

/**
 * Look up the correct localized plural form for amount `n` for the string with the translation key `key`.
 * If the current language does not contain a translation for this key, returns the text in the default language,
 * or `null` if `suppress_fallback` is set to `true`.
 */
export function GetPluralizedString(key, n, suppress_fallback) {
  const result = pageData.PLURALSTRINGS_LANG[key]?.[PLURAL_RULES[pageData.PLURAL_RULE_LANG](n)];
  if (result || suppress_fallback) return result;
  return pageData.PLURALSTRINGS_FALLBACK[key][PLURAL_RULES[pageData.PLURAL_RULE_FALLBACK](n)];
}
