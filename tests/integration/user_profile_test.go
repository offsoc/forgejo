// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestUserProfile(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		checkReadme := func(t *testing.T, title, readmeFilename string, expectedCount int) {
			t.Run(title, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Prepare the test repository
				user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

				var ops []*files_service.ChangeRepoFile
				op := "create"
				if readmeFilename != "README.md" {
					ops = append(ops, &files_service.ChangeRepoFile{
						Operation: "delete",
						TreePath:  "README.md",
					})
				} else {
					op = "update"
				}
				if readmeFilename != "" {
					ops = append(ops, &files_service.ChangeRepoFile{
						Operation:     op,
						TreePath:      readmeFilename,
						ContentReader: strings.NewReader("# Hi!\n"),
					})
				}

				_, _, f := tests.CreateDeclarativeRepo(t, user2, ".profile", nil, nil, ops)
				defer f()

				// Perform the test
				req := NewRequest(t, "GET", "/user2")
				resp := MakeRequest(t, req, http.StatusOK)

				doc := NewHTMLParser(t, resp.Body)
				readmeCount := doc.Find("#readme_profile").Length()

				assert.Equal(t, expectedCount, readmeCount)
			})
		}

		checkReadme(t, "No readme", "", 0)
		checkReadme(t, "README.md", "README.md", 1)
		checkReadme(t, "readme.md", "readme.md", 1)
		checkReadme(t, "ReadMe.mD", "ReadMe.mD", 1)
		checkReadme(t, "readme.org does not render", "README.org", 0)
	})
}
