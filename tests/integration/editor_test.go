// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/translation"
	gitea_context "forgejo.org/services/context"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFileOnProtectedBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")

		csrf := GetCSRF(t, session, "/user2/repo1/settings/branches")
		// Change master branch to protected
		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/branches/edit", map[string]string{
			"_csrf":       csrf,
			"rule_name":   "master",
			"enable_push": "true",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		// Check if master branch has been locked successfully
		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Equal(t, "success%3DBranch%2Bprotection%2Bfor%2Brule%2B%2522master%2522%2Bhas%2Bbeen%2Bupdated.", flashCookie.Value)

		// Request editor page
		req = NewRequest(t, "GET", "/user2/repo1/_new/master/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		lastCommit := doc.GetInputValueByName("last_commit")
		assert.NotEmpty(t, lastCommit)

		// Save new file to master branch
		req = NewRequestWithValues(t, "POST", "/user2/repo1/_new/master/", map[string]string{
			"_csrf":          doc.GetCSRF(),
			"last_commit":    lastCommit,
			"tree_path":      "test.txt",
			"content":        "Content",
			"commit_choice":  "direct",
			"commit_mail_id": "3",
		})

		resp = session.MakeRequest(t, req, http.StatusOK)
		// Check body for error message
		assert.Contains(t, resp.Body.String(), "Cannot commit to protected branch &#34;master&#34;.")

		// remove the protected branch
		csrf = GetCSRF(t, session, "/user2/repo1/settings/branches")

		// Change master branch to protected
		req = NewRequestWithValues(t, "POST", "/user2/repo1/settings/branches/1/delete", map[string]string{
			"_csrf": csrf,
		})

		resp = session.MakeRequest(t, req, http.StatusOK)

		res := make(map[string]string)
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
		assert.Equal(t, "/user2/repo1/settings/branches", res["redirect"])

		// Check if master branch has been locked successfully
		flashCookie = session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Equal(t, "error%3DRemoving%2Bbranch%2Bprotection%2Brule%2B%25221%2522%2Bfailed.", flashCookie.Value)
	})
}

func testEditFile(t *testing.T, session *TestSession, user, repo, branch, filePath, newContent string) *httptest.ResponseRecorder {
	// Get to the 'edit this file' page
	req := NewRequest(t, "GET", path.Join(user, repo, "_edit", branch, filePath))
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	lastCommit := htmlDoc.GetInputValueByName("last_commit")
	assert.NotEmpty(t, lastCommit)

	// Submit the edits
	req = NewRequestWithValues(t, "POST", path.Join(user, repo, "_edit", branch, filePath),
		map[string]string{
			"_csrf":          htmlDoc.GetCSRF(),
			"last_commit":    lastCommit,
			"tree_path":      filePath,
			"content":        newContent,
			"commit_choice":  "direct",
			"commit_mail_id": "-1",
		},
	)
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify the change
	req = NewRequest(t, "GET", path.Join(user, repo, "raw/branch", branch, filePath))
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, newContent, resp.Body.String())

	return resp
}

func testEditFileToNewBranch(t *testing.T, session *TestSession, user, repo, branch, targetBranch, filePath, newContent string) *httptest.ResponseRecorder {
	// Get to the 'edit this file' page
	req := NewRequest(t, "GET", path.Join(user, repo, "_edit", branch, filePath))
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	lastCommit := htmlDoc.GetInputValueByName("last_commit")
	assert.NotEmpty(t, lastCommit)

	// Submit the edits
	req = NewRequestWithValues(t, "POST", path.Join(user, repo, "_edit", branch, filePath),
		map[string]string{
			"_csrf":           htmlDoc.GetCSRF(),
			"last_commit":     lastCommit,
			"tree_path":       filePath,
			"content":         newContent,
			"commit_choice":   "commit-to-new-branch",
			"new_branch_name": targetBranch,
			"commit_mail_id":  "-1",
		},
	)
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify the change
	req = NewRequest(t, "GET", path.Join(user, repo, "raw/branch", targetBranch, filePath))
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, newContent, resp.Body.String())

	return resp
}

func TestEditFile(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")
		testEditFile(t, session, "user2", "repo1", "master", "README.md", "Hello, World (Edited)\n")
	})
}

func TestEditFileToNewBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")
		testEditFileToNewBranch(t, session, "user2", "repo1", "master", "feature/test", "README.md", "Hello, World (Edited)\n")
	})
}

func TestEditorAddTranslation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/user2/repo1/_new/master")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	placeholder, ok := htmlDoc.Find("input[name='commit_summary']").Attr("placeholder")
	assert.True(t, ok)
	assert.Equal(t, `Add "<filename>"`, placeholder)
}

func TestCommitMail(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		// Require that the user has KeepEmailPrivate enabled, because it needs
		// to be tested that even with this setting enabled, it will use the
		// provided mail and not revert to the placeholder one.
		assert.True(t, user.KeepEmailPrivate)

		inactivatedMail := unittest.AssertExistsAndLoadBean(t, &user_model.EmailAddress{ID: 35, UID: user.ID})
		assert.False(t, inactivatedMail.IsActivated)

		otherEmail := unittest.AssertExistsAndLoadBean(t, &user_model.EmailAddress{ID: 1, IsActivated: true})
		assert.NotEqual(t, otherEmail.UID, user.ID)

		primaryEmail := unittest.AssertExistsAndLoadBean(t, &user_model.EmailAddress{ID: 3, UID: user.ID, IsActivated: true})

		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		gitRepo, _ := git.OpenRepository(git.DefaultContext, repo1.RepoPath())
		defer gitRepo.Close()

		session := loginUser(t, user.Name)

		lastCommitAndCSRF := func(t *testing.T, link string, skipLastCommit bool) (string, string) {
			t.Helper()

			req := NewRequest(t, "GET", link)
			resp := session.MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			lastCommit := htmlDoc.GetInputValueByName("last_commit")
			if !skipLastCommit {
				assert.NotEmpty(t, lastCommit)
			}

			return lastCommit, htmlDoc.GetCSRF()
		}

		type caseOpts struct {
			link           string
			fileName       string
			base           map[string]string
			skipLastCommit bool
		}

		// Base2 should have different content, so we can test two 'correct' operations
		// without the second becoming a noop because no content was changed. If needed,
		// link2 can point to a new file that's used with base2.
		assertCase := func(t *testing.T, case1, case2 caseOpts) {
			t.Helper()

			t.Run("Not activated", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				lastCommit, csrf := lastCommitAndCSRF(t, case1.link, case1.skipLastCommit)
				baseCopy := case1.base
				baseCopy["_csrf"] = csrf
				baseCopy["last_commit"] = lastCommit
				baseCopy["commit_mail_id"] = fmt.Sprintf("%d", inactivatedMail.ID)

				req := NewRequestWithValues(t, "POST", case1.link, baseCopy)
				resp := session.MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				assert.Contains(t,
					htmlDoc.doc.Find(".ui.negative.message").Text(),
					translation.NewLocale("en-US").Tr("repo.editor.invalid_commit_mail"),
				)
			})

			t.Run("Not belong to user", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				lastCommit, csrf := lastCommitAndCSRF(t, case1.link, case1.skipLastCommit)
				baseCopy := case1.base
				baseCopy["_csrf"] = csrf
				baseCopy["last_commit"] = lastCommit
				baseCopy["commit_mail_id"] = fmt.Sprintf("%d", otherEmail.ID)

				req := NewRequestWithValues(t, "POST", case1.link, baseCopy)
				resp := session.MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				assert.Contains(t,
					htmlDoc.doc.Find(".ui.negative.message").Text(),
					translation.NewLocale("en-US").Tr("repo.editor.invalid_commit_mail"),
				)
			})

			t.Run("Placeholder mail", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				lastCommit, csrf := lastCommitAndCSRF(t, case1.link, case1.skipLastCommit)
				baseCopy := case1.base
				baseCopy["_csrf"] = csrf
				baseCopy["last_commit"] = lastCommit
				baseCopy["commit_mail_id"] = "-1"

				req := NewRequestWithValues(t, "POST", case1.link, baseCopy)
				session.MakeRequest(t, req, http.StatusSeeOther)
				if !case2.skipLastCommit {
					newlastCommit, _ := lastCommitAndCSRF(t, case1.link, false)
					assert.NotEqual(t, newlastCommit, lastCommit)
				}

				commit, err := gitRepo.GetCommitByPath(case1.fileName)
				require.NoError(t, err)

				assert.Equal(t, "user2", commit.Author.Name)
				assert.Equal(t, "user2@noreply.example.org", commit.Author.Email)
				assert.Equal(t, "user2", commit.Committer.Name)
				assert.Equal(t, "user2@noreply.example.org", commit.Committer.Email)
			})

			t.Run("Normal", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				lastCommit, csrf := lastCommitAndCSRF(t, case2.link, case2.skipLastCommit)
				baseCopy := case2.base
				baseCopy["_csrf"] = csrf
				baseCopy["last_commit"] = lastCommit
				baseCopy["commit_mail_id"] = fmt.Sprintf("%d", primaryEmail.ID)

				req := NewRequestWithValues(t, "POST", case2.link, baseCopy)
				session.MakeRequest(t, req, http.StatusSeeOther)
				if !case2.skipLastCommit {
					newlastCommit, _ := lastCommitAndCSRF(t, case2.link, false)
					assert.NotEqual(t, newlastCommit, lastCommit)
				}

				commit, err := gitRepo.GetCommitByPath(case2.fileName)
				require.NoError(t, err)

				assert.Equal(t, "user2", commit.Author.Name)
				assert.Equal(t, primaryEmail.Email, commit.Author.Email)
				assert.Equal(t, "user2", commit.Committer.Name)
				assert.Equal(t, primaryEmail.Email, commit.Committer.Email)
			})
		}

		t.Run("New", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			assertCase(t, caseOpts{
				fileName: "new_file",
				link:     "user2/repo1/_new/master",
				base: map[string]string{
					"tree_path":     "new_file",
					"content":       "new_content",
					"commit_choice": "direct",
				},
			}, caseOpts{
				fileName: "new_file_2",
				link:     "user2/repo1/_new/master",
				base: map[string]string{
					"tree_path":     "new_file_2",
					"content":       "new_content",
					"commit_choice": "direct",
				},
			},
			)
		})

		t.Run("Edit", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			assertCase(t, caseOpts{
				fileName: "README.md",
				link:     "user2/repo1/_edit/master/README.md",
				base: map[string]string{
					"tree_path":     "README.md",
					"content":       "Edit content",
					"commit_choice": "direct",
				},
			}, caseOpts{
				fileName: "README.md",
				link:     "user2/repo1/_edit/master/README.md",
				base: map[string]string{
					"tree_path":     "README.md",
					"content":       "Other content",
					"commit_choice": "direct",
				},
			},
			)
		})

		t.Run("Delete", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			assertCase(t, caseOpts{
				fileName: "new_file",
				link:     "user2/repo1/_delete/master/new_file",
				base: map[string]string{
					"commit_choice": "direct",
				},
			}, caseOpts{
				fileName: "new_file_2",
				link:     "user2/repo1/_delete/master/new_file_2",
				base: map[string]string{
					"commit_choice": "direct",
				},
			},
			)
		})

		t.Run("Upload", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Upload two separate times, so we have two different 'uploads' that can
			// be used independently of each other.
			uploadFile := func(t *testing.T, name, content string) string {
				t.Helper()

				body := &bytes.Buffer{}
				mpForm := multipart.NewWriter(body)
				err := mpForm.WriteField("_csrf", GetCSRF(t, session, "/user2/repo1/_upload/master"))
				require.NoError(t, err)

				file, err := mpForm.CreateFormFile("file", name)
				require.NoError(t, err)

				io.Copy(file, bytes.NewBufferString(content))
				require.NoError(t, mpForm.Close())

				req := NewRequestWithBody(t, "POST", "/user2/repo1/upload-file", body)
				req.Header.Add("Content-Type", mpForm.FormDataContentType())
				resp := session.MakeRequest(t, req, http.StatusOK)

				respMap := map[string]string{}
				DecodeJSON(t, resp, &respMap)
				return respMap["uuid"]
			}

			file1UUID := uploadFile(t, "upload_file_1", "Uploaded a file!")
			file2UUID := uploadFile(t, "upload_file_2", "Uploaded another file!")
			file1UUIDFullpathKey := fmt.Sprintf("files_fullpath[%s]", file1UUID)
			file2UUIDFullpathKey := fmt.Sprintf("files_fullpath[%s]", file2UUID)

			assertCase(t, caseOpts{
				fileName:       "upload_file_1",
				link:           "user2/repo1/_upload/master",
				skipLastCommit: true,
				base: map[string]string{
					"commit_choice":      "direct",
					"files":              file1UUID,
					file1UUIDFullpathKey: "upload_file_1",
				},
			}, caseOpts{
				fileName:       "upload_file_2",
				link:           "user2/repo1/_upload/master",
				skipLastCommit: true,
				base: map[string]string{
					"commit_choice":      "direct",
					"files":              file2UUID,
					file2UUIDFullpathKey: "upload_file_2",
				},
			},
			)
		})

		t.Run("Apply patch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			assertCase(t, caseOpts{
				fileName: "diff-file-1.txt",
				link:     "user2/repo1/_diffpatch/master",
				base: map[string]string{
					"tree_path":     "patch",
					"commit_choice": "direct",
					"content": `diff --git a/diff-file-1.txt b/diff-file-1.txt
new file mode 100644
index 0000000000..50fcd26d6c
--- /dev/null
+++ b/diff-file-1.txt
@@ -0,0 +1 @@
+File 1
`,
				},
			}, caseOpts{
				fileName: "diff-file-2.txt",
				link:     "user2/repo1/_diffpatch/master",
				base: map[string]string{
					"tree_path":     "patch",
					"commit_choice": "direct",
					"content": `diff --git a/diff-file-2.txt b/diff-file-2.txt
new file mode 100644
index 0000000000..4475433e27
--- /dev/null
+++ b/diff-file-2.txt
@@ -0,0 +1 @@
+File 2
`,
				},
			})
		})

		t.Run("Cherry pick", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			commitID1, err := gitRepo.GetCommitByPath("diff-file-1.txt")
			require.NoError(t, err)
			commitID2, err := gitRepo.GetCommitByPath("diff-file-2.txt")
			require.NoError(t, err)

			assertCase(t, caseOpts{
				fileName: "diff-file-1.txt",
				link:     "user2/repo1/_cherrypick/" + commitID1.ID.String() + "/master",
				base: map[string]string{
					"commit_choice": "direct",
					"revert":        "true",
				},
			}, caseOpts{
				fileName: "diff-file-2.txt",
				link:     "user2/repo1/_cherrypick/" + commitID2.ID.String() + "/master",
				base: map[string]string{
					"commit_choice": "direct",
					"revert":        "true",
				},
			})
		})
	})
}
