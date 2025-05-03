// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package integration

import (
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/translation/i18n"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestMissingTranslationHandling(t *testing.T) {
	// Currently new languages can only be added to localestore via AddLocaleByIni
	// so this line is here to make the other one work. When INI locales are removed,
	// it will not be needed by this test.
	i18n.DefaultLocales.AddLocaleByIni("fun", "Funlang", nil, nil, []byte(""), nil)

	// Add a testing locale to the store
	i18n.DefaultLocales.AddToLocaleFromJSON("fun", []byte(`{
		"meta.last_line": "This language only has one line that is never used by the UI. It will never have a translation for incorrect_root_url"
	}`))

	// Get "fun" locale, make sure it's available
	funLocale, found := i18n.DefaultLocales.Locale("fun")
	assert.True(t, found)

	// Get translation for a string that this locale doesn't have
	s := funLocale.TrString("incorrect_root_url")

	// Verify fallback to English
	assert.True(t, strings.HasPrefix(s, "This Forgejo instance"))
}

// TestDataSizeTranslation is a test for usage of TrSize in file size display
func TestDataSizeTranslation(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		testUser := "user2"
		testRepoName := "data_size_test"
		noDigits := regexp.MustCompile("[0-9]+")
		longString100 := `testRepoMigrate(t, session, "https://code.forgejo.org/forgejo/test_repo.git", testRepoName, struct)` + "\n"

		// Login user
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: testUser})
		session := loginUser(t, testUser)

		// Create test repo
		testRepo, _, f := tests.CreateDeclarativeRepo(t, user2, testRepoName, nil, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "137byteFile.txt",
					ContentReader: strings.NewReader(longString100 + strings.Repeat("1", 36) + "\n"),
				},
				{
					Operation:     "create",
					TreePath:      "1.5kibFile.txt",
					ContentReader: strings.NewReader(strings.Repeat(longString100, 15) + strings.Repeat("1", 35) + "\n"),
				},
				{
					Operation:     "create",
					TreePath:      "1.25mibFile.txt",
					ContentReader: strings.NewReader(strings.Repeat(longString100, 13107) + strings.Repeat("1", 19) + "\n"),
				},
			})
		defer f()

		// Change language from English to catch regressions that make translated sizes fall back to
		// not translated, like to raw output of FileSize() or humanize.IBytes()
		lang := session.GetCookie("lang")
		lang.Value = "ru-RU"
		session.SetCookie(lang)

		// Go to /user/settings/repos
		req := NewRequest(t, "GET", "user/settings/repos")
		resp := session.MakeRequest(t, req, http.StatusOK)

		// Check if repo size is translated
		repos := NewHTMLParser(t, resp.Body).Find(".user-setting-content .list .item .content")
		assert.Positive(t, repos.Length())
		repos.Each(func(i int, repo *goquery.Selection) {
			repoName := repo.Find("a.name").Text()
			if repoName == path.Join(testUser, testRepo.Name) {
				repoSize := repo.Find("span").Text()
				repoSize = noDigits.ReplaceAllString(repoSize, "")
				assert.Equal(t, " КиБ", repoSize)
			}
		})

		// Go to /user2/repo1
		req = NewRequest(t, "GET", path.Join(testUser, testRepoName))
		resp = session.MakeRequest(t, req, http.StatusOK)

		// Check if repo size in repo summary is translated
		repo := NewHTMLParser(t, resp.Body).Find(".repository-summary span")
		repoSize := strings.TrimSpace(repo.Text())
		repoSize = noDigits.ReplaceAllString(repoSize, "")
		assert.Equal(t, " КиБ", repoSize)

		// Check if repo sizes in the tooltip are translated
		fullSize, exists := repo.Attr("data-tooltip-content")
		assert.True(t, exists)
		fullSize = noDigits.ReplaceAllString(fullSize, "")
		assert.Equal(t, "git:  КиБ; lfs:  Б", fullSize)

		// Check if file sizes are correctly translated
		testFileSizeTranslated(t, session, path.Join(testUser, testRepoName, "src/branch/main/137byteFile.txt"), "137 Б")
		testFileSizeTranslated(t, session, path.Join(testUser, testRepoName, "src/branch/main/1.5kibFile.txt"), "1,5 КиБ")
		testFileSizeTranslated(t, session, path.Join(testUser, testRepoName, "src/branch/main/1.25mibFile.txt"), "1,3 МиБ")
	})
}

func testFileSizeTranslated(t *testing.T, session *TestSession, filePath, correctSize string) {
	// Go to specified file page
	req := NewRequest(t, "GET", filePath)
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Check if file size is translated
	sizeCorrect := false
	fileInfo := NewHTMLParser(t, resp.Body).Find(".file-info .file-info-entry")
	fileInfo.Each(func(i int, info *goquery.Selection) {
		infoText := strings.TrimSpace(info.Text())
		if infoText == correctSize {
			sizeCorrect = true
		}
	})

	assert.True(t, sizeCorrect)
}
