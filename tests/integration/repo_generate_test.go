// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/translation"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func assertRepoCreateForm(t *testing.T, htmlDoc *HTMLDoc, owner *user_model.User, templateID string) {
	_, exists := htmlDoc.doc.Find("form.ui.form[action^='/repo/create']").Attr("action")
	assert.True(t, exists, "Expected the repo creation form")
	locale := translation.NewLocale("en-US")

	// Verify page title
	title := htmlDoc.doc.Find("title").Text()
	assert.Contains(t, title, locale.TrString("new_repo.title"))

	// Verify form header
	header := strings.TrimSpace(htmlDoc.doc.Find(".form[action='/repo/create'] .header").Text())
	assert.EqualValues(t, locale.TrString("new_repo.title"), header)

	htmlDoc.AssertDropdownHasSelectedOption(t, "uid", strconv.FormatInt(owner.ID, 10))

	// the template menu is loaded client-side, so don't assert the option exists
	assert.Equal(t, templateID, htmlDoc.GetInputValueByName("repo_template"), "Unexpected repo_template selection")

	for _, name := range []string{"issue_labels", "gitignores", "license", "object_format_name"} {
		htmlDoc.AssertDropdownHasOptions(t, name)
	}
}

func testRepoGenerate(t *testing.T, session *TestSession, templateID, templateOwnerName, templateRepoName string, user, generateOwner *user_model.User, generateRepoName string) {
	// Step0: check the existence of the generated repo
	req := NewRequestf(t, "GET", "/%s/%s", generateOwner.Name, generateRepoName)
	session.MakeRequest(t, req, http.StatusNotFound)

	// Step1: go to the main page of template repo
	req = NewRequestf(t, "GET", "/%s/%s", templateOwnerName, templateRepoName)
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Step2: click the "Use this template" button
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("a.ui.button[href^=\"/repo/create\"]").Attr("href")
	assert.True(t, exists, "The template has changed")
	req = NewRequest(t, "GET", link)
	resp = session.MakeRequest(t, req, http.StatusOK)

	// Step3: test and submit form
	htmlDoc = NewHTMLParser(t, resp.Body)
	assertRepoCreateForm(t, htmlDoc, user, templateID)
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf":         htmlDoc.GetCSRF(),
		"uid":           fmt.Sprintf("%d", generateOwner.ID),
		"repo_name":     generateRepoName,
		"repo_template": templateID,
		"git_content":   "true",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Step4: check the existence of the generated repo
	req = NewRequestf(t, "GET", "/%s/%s", generateOwner.Name, generateRepoName)
	session.MakeRequest(t, req, http.StatusOK)
}

func testRepoGenerateWithFixture(t *testing.T, session *TestSession, templateID, templateOwnerName, templateRepoName string, user, generateOwner *user_model.User, generateRepoName string) {
	testRepoGenerate(t, session, templateID, templateOwnerName, templateRepoName, user, generateOwner, generateRepoName)

	// check substituted values in Readme
	req := NewRequestf(t, "GET", "/%s/%s/raw/branch/master/README.md", generateOwner.Name, generateRepoName)
	resp := session.MakeRequest(t, req, http.StatusOK)
	body := fmt.Sprintf(`# %s Readme
Owner: %s
Link: /%s/%s
Clone URL: %s%s/%s.git`,
		generateRepoName,
		strings.ToUpper(generateOwner.Name),
		generateOwner.Name,
		generateRepoName,
		setting.AppURL,
		generateOwner.Name,
		generateRepoName)
	assert.Equal(t, body, resp.Body.String())

	// Step6: check substituted values in substituted file path ${REPO_NAME}
	req = NewRequestf(t, "GET", "/%s/%s/raw/branch/master/%s.log", generateOwner.Name, generateRepoName, generateRepoName)
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, generateRepoName, resp.Body.String())
}

// test form elements before and after POST error response
func TestRepoCreateForm(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	userName := "user1"
	session := loginUser(t, userName)
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})

	req := NewRequest(t, "GET", "/repo/create")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assertRepoCreateForm(t, htmlDoc, user, "")

	req = NewRequestWithValues(t, "POST", "/repo/create", map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
	})
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assertRepoCreateForm(t, htmlDoc, user, "")
}

func TestRepoGenerate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	userName := "user1"
	session := loginUser(t, userName)
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})

	testRepoGenerateWithFixture(t, session, "44", "user27", "template1", user, user, "generated1")
}

func TestRepoGenerateToOrg(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	userName := "user2"
	session := loginUser(t, userName)
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "org3"})

	testRepoGenerateWithFixture(t, session, "44", "user27", "template1", user, org, "generated2")
}

func TestRepoCreateFormTrimSpace(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)

	req := NewRequestWithValues(t, "POST", "/repo/create", map[string]string{
		"_csrf":     GetCSRF(t, session, "/repo/create"),
		"uid":       "2",
		"repo_name": " spaced-name ",
	})
	resp := session.MakeRequest(t, req, http.StatusSeeOther)

	assert.EqualValues(t, "/user2/spaced-name", test.RedirectURL(resp))
	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: 2, Name: "spaced-name"})
}

func TestRepoGenerateTemplating(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		input := `# $REPO_NAME
	This is a Repo By $REPO_OWNER
	ThisIsThe${REPO_NAME}InAnInlineWay`
		expected := `# %s
	This is a Repo By %s
	ThisIsThe%sInAnInlineWay`
		templateName := "my_template"
		generatedName := "my_generated"

		userName := "user1"
		session := loginUser(t, userName)
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})

		template, _, f := tests.CreateDeclarativeRepoWithOptions(t, user, tests.DeclarativeRepoOptions{
			Name:       optional.Some(templateName),
			IsTemplate: optional.Some(true),
			Files: optional.Some([]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      ".forgejo/template",
					ContentReader: strings.NewReader("Readme.md"),
				},
				{
					Operation:     "create",
					TreePath:      "Readme.md",
					ContentReader: strings.NewReader(input),
				},
			}),
		})
		defer f()

		// The repo.TemplateID field is not initialized. Luckily, the ID field holds the expected value
		templateID := strconv.FormatInt(template.ID, 10)

		testRepoGenerate(
			t,
			session,
			templateID,
			user.Name,
			templateName,
			user,
			user,
			generatedName,
		)

		req := NewRequestf(
			t,
			"GET", "/%s/%s/raw/branch/%s/Readme.md",
			user.Name,
			generatedName,
			template.DefaultBranch,
		)
		resp := session.MakeRequest(t, req, http.StatusOK)
		body := fmt.Sprintf(expected,
			generatedName,
			user.Name,
			generatedName)
		assert.Equal(t, body, resp.Body.String())
	})
}
