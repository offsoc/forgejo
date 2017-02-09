package markdown_test

import (
	"fmt"
	"strconv"
	"testing"

	. "code.gitea.io/gitea/modules/markdown"
	"code.gitea.io/gitea/modules/setting"
	"github.com/stretchr/testify/assert"
)

const AppURL = "http://localhost:3000/"

var numericMetas = map[string]string{
	"format": "https://someurl.com/{user}/{repo}/{index}",
	"user":   "someUser",
	"repo":   "someRepo",
	"style":  IssueNameStyleNumeric,
}

var alphanumericMetas = map[string]string{
	"format": "https://someurl.com/{user}/{repo}/{index}",
	"user":   "someUser",
	"repo":   "someRepo",
	"style":  IssueNameStyleAlphanumeric,
}

// numericLink an HTML to a numeric-style issue
func numericIssueLink(baseURL string, index int) string {
	return link(URLJoin(baseURL, strconv.Itoa(index)), fmt.Sprintf("#%d", index))
}

// alphanumLink an HTML link to an alphanumeric-style issue
func alphanumIssueLink(baseURL string, name string) string {
	return link(URLJoin(baseURL, name), name)
}

// urlContentsLink an HTML link whose contents is the target URL
func urlContentsLink(href string) string {
	return link(href, href)
}

// link an HTML link
func link(href, contents string) string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", href, contents)
}

func testRenderIssueIndexPattern(t *testing.T, input, expected string, metas map[string]string) {
	assert.Equal(t, expected,
		string(RenderIssueIndexPattern([]byte(input), AppURL, metas)))
}

func TestRenderIssueIndexPattern(t *testing.T) {
	// numeric: render inputs without valid mentions
	test := func(s string) {
		testRenderIssueIndexPattern(t, s, s, nil)
		testRenderIssueIndexPattern(t, s, s, numericMetas)
	}

	// should not render anything when there are no mentions
	test("")
	test("this is a test")
	test("test 123 123 1234")
	test("#")
	test("# # #")
	test("# 123")
	test("#abcd")
	test("##1234")
	test("test#1234")
	test("#1234test")
	test(" test #1234test")

	// should not render issue mention without leading space
	test("test#54321 issue")

	// should not render issue mention without trailing space
	test("test #54321issue")
}

func TestRenderIssueIndexPattern2(t *testing.T) {
	setting.AppURL = AppURL
	setting.AppSubURL = setting.AppURL

	// numeric: render inputs with valid mentions
	test := func(s, expectedFmt string, indices ...int) {
		links := make([]interface{}, len(indices))
		for i, index := range indices {
			links[i] = numericIssueLink(URLJoin(setting.AppSubURL, "issues"), index)
		}
		expectedNil := fmt.Sprintf(expectedFmt, links...)
		testRenderIssueIndexPattern(t, s, expectedNil, nil)

		for i, index := range indices {
			links[i] = numericIssueLink("https://someurl.com/someUser/someRepo/", index)
		}
		expectedNum := fmt.Sprintf(expectedFmt, links...)
		testRenderIssueIndexPattern(t, s, expectedNum, numericMetas)
	}

	// should render freestanding mentions
	test("#1234 test", "%s test", 1234)
	test("test #8 issue", "test %s issue", 8)
	test("test issue #1234", "test issue %s", 1234)

	// should render mentions in parentheses
	test("(#54321 issue)", "(%s issue)", 54321)
	test("test (#9801 extra) issue", "test (%s extra) issue", 9801)
	test("test (#1)", "test (%s)", 1)

	// should render multiple issue mentions in the same line
	test("#54321 #1243", "%s %s", 54321, 1243)
	test("wow (#54321 #1243)", "wow (%s %s)", 54321, 1243)
	test("(#4)(#5)", "(%s)(%s)", 4, 5)
	test("#1 (#4321) test", "%s (%s) test", 1, 4321)
}

func TestRenderIssueIndexPattern3(t *testing.T) {
	// alphanumeric: render inputs without valid mentions
	test := func(s string) {
		testRenderIssueIndexPattern(t, s, s, alphanumericMetas)
	}
	test("")
	test("this is a test")
	test("test 123 123 1234")
	test("#")
	test("##1234")
	test("# 123")
	test("#abcd")
	test("test #123")
	test("abc-1234")         // issue prefix must be capital
	test("ABc-1234")         // issue prefix must be _all_ capital
	test("ABCDEFGHIJK-1234") // the limit is 10 characters in the prefix
	test("ABC1234")          // dash is required
	test("test ABC- test")   // number is required
	test("test -1234 test")  // prefix is required
	test("testABC-123 test") // leading space is required
	test("test ABC-123test") // trailing space is required
	test("ABC-0123")         // no leading zero
}

func TestRenderIssueIndexPattern4(t *testing.T) {
	// alphanumeric: render inputs with valid mentions
	test := func(s, expectedFmt string, names ...string) {
		links := make([]interface{}, len(names))
		for i, name := range names {
			links[i] = alphanumIssueLink("https://someurl.com/someUser/someRepo/", name)
		}
		expected := fmt.Sprintf(expectedFmt, links...)
		testRenderIssueIndexPattern(t, s, expected, alphanumericMetas)
	}
	test("OTT-1234 test", "%s test", "OTT-1234")
	test("test T-12 issue", "test %s issue", "T-12")
	test("test issue ABCDEFGHIJ-1234567890", "test issue %s", "ABCDEFGHIJ-1234567890")
}

func TestRenderer_AutoLink(t *testing.T) {
	setting.AppURL = AppURL
	setting.AppSubURL = URLJoin(setting.AppURL, "user", "repo")
	SubURLNoProtocol := setting.AppSubURL[5:]

	test := func(input, expected string) {
		buffer := RenderSpecialLink([]byte(input), setting.AppSubURL, map[string]string{})
		assert.Equal(t, expected, string(buffer))
	}

	// render valid issue URLs
	test(URLJoin(setting.AppSubURL, "issues", "3333"),
		numericIssueLink(URLJoin(setting.AppSubURL, "issues"), 3333))

	// render external issue URLs
	tmp := "//1111/2222/ssss-issues/3333?param=blah&blahh=333"
	test("http:"+tmp,
		"<a href=\""+tmp+"\">#3333 <i class='comment icon'></i></a>")
	test("http://test.com/issues/33333", numericIssueLink("//test.com/issues", 33333))
	test("https://issues/333", numericIssueLink("//issues", 333))

	// render valid commit URLs
	tmp = URLJoin(SubURLNoProtocol, "commit", "d8a994ef243349f321568f9e36d5c3f444b99cae")
	test("http://"+tmp, "<a href=\""+tmp+"\">d8a994ef24</a>")
	tmp += "#diff-2"
	test("http://"+tmp, "<a href=\""+tmp+"\">d8a994ef24 (diff-2)</a>")

	// render other commit URLs
	tmp = "//external-link.gogs.io/gogs/gogs/commit/d8a994ef243349f321568f9e36d5c3f444b99cae#diff-2"
	test("https:"+tmp, "<a href=\""+tmp+"\">d8a994ef24 (diff-2)</a>")
}

func TestRegExp_MentionPattern(t *testing.T) {
	trueTestCases := []string{
		"@Unknwon",
		"@ANT_123",
		"@xxx-DiN0-z-A..uru..s-xxx",
		"   @lol   ",
		" @Te/st",
	}
	falseTestCases := []string{
		"@ 0",
		"@ ",
		"@",
		"",
		"ABC",
	}

	for _, testCase := range trueTestCases {
		res := MentionPattern.MatchString(testCase)
		if !res {
			println()
			println(testCase)
		}
		assert.True(t, res)
	}
	for _, testCase := range falseTestCases {
		res := MentionPattern.MatchString(testCase)
		if res {
			println()
			println(testCase)
		}
		assert.False(t, res)
	}
}

func TestRegExp_IssueNumericPattern(t *testing.T) {
	trueTestCases := []string{
		"#1234",
		"#0",
		"#1234567890987654321",
	}
	falseTestCases := []string{
		"# 1234",
		"# 0",
		"# ",
		"#",
		"#ABC",
		"#1A2B",
		"",
		"ABC",
	}

	for _, testCase := range trueTestCases {
		assert.True(t, IssueNumericPattern.MatchString(testCase))
	}
	for _, testCase := range falseTestCases {
		assert.False(t, IssueNumericPattern.MatchString(testCase))
	}
}

func TestRegExp_IssueAlphanumericPattern(t *testing.T) {
	trueTestCases := []string{
		"ABC-1234",
		"A-1",
		"RC-80",
		"ABCDEFGHIJ-1234567890987654321234567890",
	}
	falseTestCases := []string{
		"RC-08",
		"PR-0",
		"ABCDEFGHIJK-1",
		"PR_1",
		"",
		"#ABC",
		"",
		"ABC",
		"GG-",
		"rm-1",
	}

	for _, testCase := range trueTestCases {
		assert.True(t, IssueAlphanumericPattern.MatchString(testCase))
	}
	for _, testCase := range falseTestCases {
		assert.False(t, IssueAlphanumericPattern.MatchString(testCase))
	}
}

func TestRegExp_Sha1CurrentPattern(t *testing.T) {
	trueTestCases := []string{
		"d8a994ef243349f321568f9e36d5c3f444b99cae",
		"abcdefabcdefabcdefabcdefabcdefabcdefabcd",
	}
	falseTestCases := []string{
		"test",
		"abcdefg",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmn",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmO",
	}

	for _, testCase := range trueTestCases {
		assert.True(t, Sha1CurrentPattern.MatchString(testCase))
	}
	for _, testCase := range falseTestCases {
		assert.False(t, Sha1CurrentPattern.MatchString(testCase))
	}
}

func TestRegExp_ShortLinkPattern(t *testing.T) {
	trueTestCases := []string{
		"[[stuff]]",
		"[[]]",
		"[[stuff|title=Difficult name with spaces*!]]",
	}
	falseTestCases := []string{
		"test",
		"abcdefg",
		"[[]",
		"[[",
		"[]",
		"]]",
		"abcdefghijklmnopqrstuvwxyz",
	}

	for _, testCase := range trueTestCases {
		assert.True(t, ShortLinkPattern.MatchString(testCase))
	}
	for _, testCase := range falseTestCases {
		assert.False(t, ShortLinkPattern.MatchString(testCase))
	}
}

func TestRegExp_AnySHA1Pattern(t *testing.T) {
	testCases := map[string][]string{
		"https://github.com/jquery/jquery/blob/a644101ed04d0beacea864ce805e0c4f86ba1cd1/test/unit/event.js#L2703": []string{
			"github.com",
			"jquery",
			"jquery",
			"blob",
			"a644101ed04d0beacea864ce805e0c4f86ba1cd1",
			"test/unit/event.js",
			"L2703",
		},
		"https://github.com/jquery/jquery/blob/a644101ed04d0beacea864ce805e0c4f86ba1cd1/test/unit/event.js": []string{
			"github.com",
			"jquery",
			"jquery",
			"blob",
			"a644101ed04d0beacea864ce805e0c4f86ba1cd1",
			"test/unit/event.js",
			"",
		},
		"https://github.com/jquery/jquery/commit/0705be475092aede1eddae01319ec931fb9c65fc": []string{
			"github.com",
			"jquery",
			"jquery",
			"commit",
			"0705be475092aede1eddae01319ec931fb9c65fc",
			"",
			"",
		},
		"https://github.com/jquery/jquery/tree/0705be475092aede1eddae01319ec931fb9c65fc/src": []string{
			"github.com",
			"jquery",
			"jquery",
			"tree",
			"0705be475092aede1eddae01319ec931fb9c65fc",
			"src",
			"",
		},
		"https://try.gogs.io/gogs/gogs/commit/d8a994ef243349f321568f9e36d5c3f444b99cae#diff-2": []string{
			"try.gogs.io",
			"gogs",
			"gogs",
			"commit",
			"d8a994ef243349f321568f9e36d5c3f444b99cae",
			"",
			"diff-2",
		},
	}

	for k, v := range testCases {
		assert.Equal(t, AnySHA1Pattern.FindStringSubmatch(k)[1:], v)
	}
}

func TestRegExp_IssueFullPattern(t *testing.T) {
	testCases := map[string][]string{
		"https://github.com/gogits/gogs/pull/3244": []string{
			"github.com/gogits/gogs/pull/",
			"3244",
			"",
			"",
		},
		"https://github.com/gogits/gogs/issues/3247#issuecomment-231517079": []string{
			"github.com/gogits/gogs/issues/",
			"3247",
			"#issuecomment-231517079",
			"",
		},
		"https://try.gogs.io/gogs/gogs/issues/4#issue-685": []string{
			"try.gogs.io/gogs/gogs/issues/",
			"4",
			"#issue-685",
			"",
		},
		"https://youtrack.jetbrains.com/issue/JT-36485": []string{
			"youtrack.jetbrains.com/issue/",
			"JT-36485",
			"",
			"",
		},
		"https://youtrack.jetbrains.com/issue/JT-36485#comment=27-1508676": []string{
			"youtrack.jetbrains.com/issue/",
			"JT-36485",
			"#comment=27-1508676",
			"",
		},
	}

	for k, v := range testCases {
		assert.Equal(t, IssueFullPattern.FindStringSubmatch(k)[1:], v)
	}
}

func TestMisc_IsMarkdownFile(t *testing.T) {
	setting.Markdown.FileExtensions = []string{".md", ".markdown", ".mdown", ".mkd"}
	trueTestCases := []string{
		"test.md",
		"wow.MARKDOWN",
		"LOL.mDoWn",
	}
	falseTestCases := []string{
		"test",
		"abcdefg",
		"abcdefghijklmnopqrstuvwxyz",
		"test.md.test",
	}

	for _, testCase := range trueTestCases {
		assert.True(t, IsMarkdownFile(testCase))
	}
	for _, testCase := range falseTestCases {
		assert.False(t, IsMarkdownFile(testCase))
	}
}

func TestMisc_IsReadmeFile(t *testing.T) {
	trueTestCases := []string{
		"readme",
		"README",
		"readME.mdown",
		"README.md",
	}
	falseTestCases := []string{
		"test.md",
		"wow.MARKDOWN",
		"LOL.mDoWn",
		"test",
		"abcdefg",
		"abcdefghijklmnopqrstuvwxyz",
		"test.md.test",
	}

	for _, testCase := range trueTestCases {
		assert.True(t, IsReadmeFile(testCase))
	}
	for _, testCase := range falseTestCases {
		assert.False(t, IsReadmeFile(testCase))
	}
}

// Test cases without ambiguous links
var sameCases = []string{
	// dear imgui wiki markdown extract: special wiki syntax
	`Wiki! Enjoy :)
- [[Links, Language bindings, Engine bindings|Links]]
- [[Tips]]

Ideas and codes

- Bezier widget (by @r-lyeh) https://github.com/ocornut/imgui/issues/786
- Node graph editors https://github.com/ocornut/imgui/issues/306
- [[Memory Editor|memory_editor_example]]
- [[Plot var helper|plot_var_example]]`,
	// rendered
	`<p>Wiki! Enjoy :)</p>

<ul>
<li><a href="` + AppURL + `wiki/Links" rel="nofollow">Links, Language bindings, Engine bindings</a></li>
<li><a href="` + AppURL + `wiki/Tips" rel="nofollow">Tips</a></li>
</ul>

<p>Ideas and codes</p>

<ul>
<li>Bezier widget (by <a href="` + AppURL + `r-lyeh" rel="nofollow">@r-lyeh</a>)<a href="` + AppURL + `issues/786" rel="nofollow">#786</a></li>
<li>Node graph editors<a href="` + AppURL + `issues/306" rel="nofollow">#306</a></li>
<li><a href="` + AppURL + `wiki/memory_editor_example" rel="nofollow">Memory Editor</a></li>
<li><a href="` + AppURL + `wiki/plot_var_example" rel="nofollow">Plot var helper</a></li>
</ul>
`,
	// wine-staging wiki home extract: tables, special wiki syntax, images
	`## What is Wine Staging?
**Wine Staging** on website [wine-staging.com](http://wine-staging.com).

## Quick Links
Here are some links to the most important topics. You can find the full list of pages at the sidebar.

| [[images/icon-install.png]]    | [[Installation]]                                         |
|--------------------------------|----------------------------------------------------------|
| [[images/icon-usage.png]]      | [[Usage]]                                                |
| [[images/icon-config.png]]     | [[Configuration]]                                        |
| [[images/icon-bug.png]]        | [Bugs](http://bugs.wine-staging.com)                     |
`,
	// rendered
	`<h2>What is Wine Staging?</h2>

<p><strong>Wine Staging</strong> on website <a href="http://wine-staging.com" rel="nofollow">wine-staging.com</a>.</p>

<h2>Quick Links</h2>

<p>Here are some links to the most important topics. You can find the full list of pages at the sidebar.</p>

<table>
<thead>
<tr>
<th><a href="` + AppURL + `wiki/raw/images%2Ficon-install.png" rel="nofollow"><img src="` + AppURL + `wiki/raw/images%2Ficon-install.png" alt="images/icon-install.png" title="icon-install.png"/></a></th>
<th><a href="` + AppURL + `wiki/Installation" rel="nofollow">Installation</a></th>
</tr>
</thead>

<tbody>
<tr>
<td><a href="` + AppURL + `wiki/raw/images%2Ficon-usage.png" rel="nofollow"><img src="` + AppURL + `wiki/raw/images%2Ficon-usage.png" alt="images/icon-usage.png" title="icon-usage.png"/></a></td>
<td><a href="` + AppURL + `wiki/Usage" rel="nofollow">Usage</a></td>
</tr>

<tr>
<td><a href="` + AppURL + `wiki/raw/images%2Ficon-config.png" rel="nofollow"><img src="` + AppURL + `wiki/raw/images%2Ficon-config.png" alt="images/icon-config.png" title="icon-config.png"/></a></td>
<td><a href="` + AppURL + `wiki/Configuration" rel="nofollow">Configuration</a></td>
</tr>

<tr>
<td><a href="` + AppURL + `wiki/raw/images%2Ficon-bug.png" rel="nofollow"><img src="` + AppURL + `wiki/raw/images%2Ficon-bug.png" alt="images/icon-bug.png" title="icon-bug.png"/></a></td>
<td><a href="http://bugs.wine-staging.com" rel="nofollow">Bugs</a></td>
</tr>
</tbody>
</table>
`,
	// libgdx wiki page: inline images with special syntax
	`[Excelsior JET](http://www.excelsiorjet.com/) allows you to create native executables for Windows, Linux and Mac OS X.

1. [Package your libGDX application](https://github.com/libgdx/libgdx/wiki/Gradle-on-the-Commandline#packaging-for-the-desktop)
[[images/1.png]]
2. Perform a test run by hitting the Run! button.
[[images/2.png]]`,
	// rendered
	`<p><a href="http://www.excelsiorjet.com/" rel="nofollow">Excelsior JET</a> allows you to create native executables for Windows, Linux and Mac OS X.</p>

<ol>
<li><a href="https://github.com/libgdx/libgdx/wiki/Gradle-on-the-Commandline#packaging-for-the-desktop" rel="nofollow">Package your libGDX application</a>
<a href="` + AppURL + `wiki/raw/images%2F1.png" rel="nofollow"><img src="` + AppURL + `wiki/raw/images%2F1.png" alt="images/1.png" title="1.png"/></a></li>
<li>Perform a test run by hitting the Run! button.
<a href="` + AppURL + `wiki/raw/images%2F2.png" rel="nofollow"><img src="` + AppURL + `wiki/raw/images%2F2.png" alt="images/2.png" title="2.png"/></a></li>
</ol>
`,
}

func TestTotal_RenderString(t *testing.T) {
	for i := 0; i < len(sameCases); i += 2 {
		line := RenderString(sameCases[i], AppURL, map[string]string{})
		assert.Equal(t, sameCases[i+1], line)
	}

	testCases := []string{}

	for i := 0; i < len(testCases); i += 2 {
		line := RenderString(testCases[i], AppURL, map[string]string{})
		assert.Equal(t, testCases[i+1], line)
	}
}

func TestTotal_RenderWiki(t *testing.T) {
	for i := 0; i < len(sameCases); i += 2 {
		line := RenderWiki([]byte(sameCases[i]), AppURL, map[string]string{})
		assert.Equal(t, sameCases[i+1], line)
	}

	testCases := []string{
		// Guard wiki sidebar: special syntax
		`[[Guardfile-DSL / Configuring-Guard|Guardfile-DSL---Configuring-Guard]]`,
		// rendered
		`<p><a href="` + AppURL + `wiki/Guardfile-DSL---Configuring-Guard" rel="nofollow">Guardfile-DSL / Configuring-Guard</a></p>
`,
		// special syntax
		`[[Name|Link]]`,
		// rendered
		`<p><a href="` + AppURL + `wiki/Link" rel="nofollow">Name</a></p>
`,
	}

	for i := 0; i < len(testCases); i += 2 {
		line := RenderWiki([]byte(testCases[i]), AppURL, map[string]string{})
		assert.Equal(t, testCases[i+1], line)
	}
}
