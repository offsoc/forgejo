// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var MockPluralRule PluralFormRule = func(n int64) PluralFormIndex {
	if n == 0 {
		return PluralFormZero
	}
	if n == 1 {
		return PluralFormOne
	}
	if n >= 2 && n <= 4 {
		return PluralFormFew
	}
	return PluralFormOther
}

var MockPluralRuleEnglish PluralFormRule = func(n int64) PluralFormIndex {
	if n == 1 {
		return PluralFormOne
	}
	return PluralFormOther
}

var (
	UsedPluralFormsEnglish = []PluralFormIndex{PluralFormOne, PluralFormOther}
	UsedPluralFormsMock    = []PluralFormIndex{PluralFormZero, PluralFormOne, PluralFormFew, PluralFormOther}
)

func TestLocaleStore(t *testing.T) {
	testData1 := []byte(`
.dot.name = Dot Name
fmt = %[1]s %[2]s

[section]
sub = Sub String
mixed = test value; <span style="color: red\; background: none;">%s</span>
`)

	testData2 := []byte(`
fmt = %[2]s %[1]s

[section]
sub = Changed Sub String
commits = fallback value for commits
`)

	testDataJSON2 := []byte(`
{
	"section.json": "the JSON is %s",
	"section.commits": {
		"one": "one %d commit",
		"few": "some %d commits",
		"other": "lots of %d commits"
	},
	"section.incomplete": {
		"few": "some %d objects (translated)"
	},
	"nested": {
		"outer": {
			"inner": {
				"json": "Hello World",
				"issue": {
					"one": "one %d issue",
					"few": "some %d issues",
					"other": "lots of %d issues"
				}
			}
		}
	}
}
`)
	testDataJSON1 := []byte(`
{
	"section.incomplete": {
		"one": "[untranslated] some %d object",
		"other": "[untranslated] some %d objects"
	}
}
`)

	ls := NewLocaleStore()
	require.NoError(t, ls.AddLocaleByIni("lang1", "Lang1", MockPluralRuleEnglish, UsedPluralFormsEnglish, testData1, nil))
	require.NoError(t, ls.AddLocaleByIni("lang2", "Lang2", MockPluralRule, UsedPluralFormsMock, testData2, nil))
	require.NoError(t, ls.AddToLocaleFromJSON("lang1", testDataJSON1))
	require.NoError(t, ls.AddToLocaleFromJSON("lang2", testDataJSON2))
	ls.SetDefaultLang("lang1")

	lang1, _ := ls.Locale("lang1")
	lang2, _ := ls.Locale("lang2")

	result := lang1.TrString("fmt", "a", "b")
	assert.Equal(t, "a b", result)

	result = lang2.TrString("fmt", "a", "b")
	assert.Equal(t, "b a", result)

	result = lang1.TrString("section.sub")
	assert.Equal(t, "Sub String", result)

	result = lang2.TrString("section.sub")
	assert.Equal(t, "Changed Sub String", result)

	langNone, _ := ls.Locale("none")
	result = langNone.TrString(".dot.name")
	assert.Equal(t, "Dot Name", result)

	result2 := lang2.TrHTML("section.mixed", "a&b")
	assert.EqualValues(t, `test value; <span style="color: red; background: none;">a&amp;b</span>`, result2)

	result = lang2.TrString("section.json", "valid")
	assert.Equal(t, "the JSON is valid", result)

	result = lang2.TrString("nested.outer.inner.json")
	assert.Equal(t, "Hello World", result)

	result = lang2.TrString("section.commits")
	assert.Equal(t, "lots of %d commits", result)

	result2 = lang2.TrPluralString(1, "section.commits", 1)
	assert.EqualValues(t, "one 1 commit", result2)

	result2 = lang2.TrPluralString(3, "section.commits", 3)
	assert.EqualValues(t, "some 3 commits", result2)

	result2 = lang2.TrPluralString(8, "section.commits", 8)
	assert.EqualValues(t, "lots of 8 commits", result2)

	result2 = lang2.TrPluralString(0, "section.commits")
	assert.EqualValues(t, "section.commits", result2)

	result2 = lang2.TrPluralString(1, "nested.outer.inner.issue", 1)
	assert.EqualValues(t, "one 1 issue", result2)

	result2 = lang2.TrPluralString(3, "nested.outer.inner.issue", 3)
	assert.EqualValues(t, "some 3 issues", result2)

	result2 = lang2.TrPluralString(9, "nested.outer.inner.issue", 9)
	assert.EqualValues(t, "lots of 9 issues", result2)

	result2 = lang2.TrPluralString(3, "section.incomplete", 3)
	assert.EqualValues(t, "some 3 objects (translated)", result2)

	result2 = lang2.TrPluralString(1, "section.incomplete", 1)
	assert.EqualValues(t, "[untranslated] some 1 object", result2)

	result2 = lang2.TrPluralString(7, "section.incomplete", 7)
	assert.EqualValues(t, "[untranslated] some 7 objects", result2)

	langs, descs := ls.ListLangNameDesc()
	assert.ElementsMatch(t, []string{"lang1", "lang2"}, langs)
	assert.ElementsMatch(t, []string{"Lang1", "Lang2"}, descs)

	// Test HasKey for JSON
	found := lang2.HasKey("section.json")
	assert.True(t, found)

	// Test HasKey for INI
	found = lang2.HasKey("section.sub")
	assert.True(t, found)

	found = lang1.HasKey("no-such")
	assert.False(t, found)
	assert.Equal(t, "no-such", lang1.TrString("no-such"))
	require.NoError(t, ls.Close())
}

func TestLocaleStoreMoreSource(t *testing.T) {
	testData1 := []byte(`
a=11
b=12
`)

	testData2 := []byte(`
b=21
c=22
`)

	ls := NewLocaleStore()
	require.NoError(t, ls.AddLocaleByIni("lang1", "Lang1", MockPluralRule, UsedPluralFormsMock, testData1, testData2))
	lang1, _ := ls.Locale("lang1")
	assert.Equal(t, "11", lang1.TrString("a"))
	assert.Equal(t, "21", lang1.TrString("b"))
	assert.Equal(t, "22", lang1.TrString("c"))
}

type stringerPointerReceiver struct {
	s string
}

func (s *stringerPointerReceiver) String() string {
	return s.s
}

type stringerStructReceiver struct {
	s string
}

func (s stringerStructReceiver) String() string {
	return s.s
}

type errorStructReceiver struct {
	s string
}

func (e errorStructReceiver) Error() string {
	return e.s
}

type errorPointerReceiver struct {
	s string
}

func (e *errorPointerReceiver) Error() string {
	return e.s
}

func TestLocaleWithTemplate(t *testing.T) {
	ls := NewLocaleStore()
	require.NoError(t, ls.AddLocaleByIni("lang1", "Lang1", MockPluralRule, UsedPluralFormsMock, []byte(`key=<a>%s</a>`), nil))
	lang1, _ := ls.Locale("lang1")

	tmpl := template.New("test").Funcs(template.FuncMap{"tr": lang1.TrHTML})
	tmpl = template.Must(tmpl.Parse(`{{tr "key" .var}}`))

	cases := []struct {
		in   any
		want string
	}{
		{"<str>", "<a>&lt;str&gt;</a>"},
		{[]byte("<bytes>"), "<a>[60 98 121 116 101 115 62]</a>"},
		{template.HTML("<html>"), "<a><html></a>"},
		{stringerPointerReceiver{"<stringerPointerReceiver>"}, "<a>{&lt;stringerPointerReceiver&gt;}</a>"},
		{&stringerPointerReceiver{"<stringerPointerReceiver ptr>"}, "<a>&lt;stringerPointerReceiver ptr&gt;</a>"},
		{stringerStructReceiver{"<stringerStructReceiver>"}, "<a>&lt;stringerStructReceiver&gt;</a>"},
		{&stringerStructReceiver{"<stringerStructReceiver ptr>"}, "<a>&lt;stringerStructReceiver ptr&gt;</a>"},
		{errorStructReceiver{"<errorStructReceiver>"}, "<a>&lt;errorStructReceiver&gt;</a>"},
		{&errorStructReceiver{"<errorStructReceiver ptr>"}, "<a>&lt;errorStructReceiver ptr&gt;</a>"},
		{errorPointerReceiver{"<errorPointerReceiver>"}, "<a>{&lt;errorPointerReceiver&gt;}</a>"},
		{&errorPointerReceiver{"<errorPointerReceiver ptr>"}, "<a>&lt;errorPointerReceiver ptr&gt;</a>"},
	}

	buf := &strings.Builder{}
	for _, c := range cases {
		buf.Reset()
		require.NoError(t, tmpl.Execute(buf, map[string]any{"var": c.in}))
		assert.Equal(t, c.want, buf.String())
	}
}

func TestLocaleStoreQuirks(t *testing.T) {
	const nl = "\n"
	q := func(q1, s string, q2 ...string) string {
		return q1 + s + strings.Join(q2, "")
	}
	testDataList := []struct {
		in   string
		out  string
		hint string
	}{
		{` xx`, `xx`, "simple, no quote"},
		{`" xx"`, ` xx`, "simple, double-quote"},
		{`' xx'`, ` xx`, "simple, single-quote"},
		{"` xx`", ` xx`, "simple, back-quote"},

		{`x\"y`, `x\"y`, "no unescape, simple"},
		{q(`"`, `x\"y`, `"`), `"x\"y"`, "unescape, double-quote"},
		{q(`'`, `x\"y`, `'`), `x\"y`, "no unescape, single-quote"},
		{q("`", `x\"y`, "`"), `x\"y`, "no unescape, back-quote"},

		{q(`"`, `x\"y`) + nl + "b=", `"x\"y`, "half open, double-quote"},
		{q(`'`, `x\"y`) + nl + "b=", `'x\"y`, "half open, single-quote"},
		{q("`", `x\"y`) + nl + "b=`", `x\"y` + nl + "b=", "half open, back-quote, multi-line"},

		{`x ; y`, `x ; y`, "inline comment (;)"},
		{`x # y`, `x # y`, "inline comment (#)"},
		{`x \; y`, `x ; y`, `inline comment (\;)`},
		{`x \# y`, `x # y`, `inline comment (\#)`},
	}

	for _, testData := range testDataList {
		ls := NewLocaleStore()
		err := ls.AddLocaleByIni("lang1", "Lang1", nil, nil, []byte("a="+testData.in), nil)
		lang1, _ := ls.Locale("lang1")
		require.NoError(t, err, testData.hint)
		assert.Equal(t, testData.out, lang1.TrString("a"), testData.hint)
		require.NoError(t, ls.Close())
	}

	// TODO: Crowdin needs the strings to be quoted correctly and doesn't like incomplete quotes
	//       and Crowdin always outputs quoted strings if there are quotes in the strings.
	//       So, Gitea's `key="quoted" unquoted` content shouldn't be used on Crowdin directly,
	//       it should be converted to `key="\"quoted\" unquoted"` first.
	// TODO: We can not use UnescapeValueDoubleQuotes=true, because there are a lot of back-quotes in en-US.ini,
	//       then Crowdin will output:
	//       > key = "`x \" y`"
	//       Then Gitea will read a string with back-quotes, which is incorrect.
	// TODO: Crowdin might generate multi-line strings, quoted by double-quote, it's not supported by LocaleStore
	//       LocaleStore uses back-quote for multi-line strings, it's not supported by Crowdin.
	// TODO: Crowdin doesn't support back-quote as string quoter, it mainly uses double-quote
	//       so, the following line will be parsed as: value="`first", comment="second`" on Crowdin
	//       > a = `first; second`
}
