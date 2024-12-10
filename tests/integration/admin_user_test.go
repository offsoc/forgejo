// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAdminViewUsers(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", "/admin/users")
	session.MakeRequest(t, req, http.StatusOK)

	session = loginUser(t, "user2")
	req = NewRequest(t, "GET", "/admin/users")
	session.MakeRequest(t, req, http.StatusForbidden)
}

func TestAdminViewUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", "/admin/users/1")
	session.MakeRequest(t, req, http.StatusOK)

	session = loginUser(t, "user2")
	req = NewRequest(t, "GET", "/admin/users/1")
	session.MakeRequest(t, req, http.StatusForbidden)
}

func TestAdminEditUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	testSuccessfullEdit(t, user_model.User{ID: 2, Name: "newusername", LoginName: "otherlogin", Email: "new@e-mail.gitea"})
}

func testSuccessfullEdit(t *testing.T, formData user_model.User) {
	makeRequest(t, formData, http.StatusSeeOther)
}

func makeRequest(t *testing.T, formData user_model.User, headerCode int) {
	session := loginUser(t, "user1")
	csrf := GetCSRF(t, session, "/admin/users/"+strconv.Itoa(int(formData.ID))+"/edit")
	req := NewRequestWithValues(t, "POST", "/admin/users/"+strconv.Itoa(int(formData.ID))+"/edit", map[string]string{
		"_csrf":      csrf,
		"user_name":  formData.Name,
		"login_name": formData.LoginName,
		"login_type": "0-0",
		"email":      formData.Email,
	})

	session.MakeRequest(t, req, headerCode)
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: formData.ID})
	assert.Equal(t, formData.Name, user.Name)
	assert.Equal(t, formData.LoginName, user.LoginName)
	assert.Equal(t, formData.Email, user.Email)
}

func TestAdminDeleteUser(t *testing.T) {
	defer tests.AddFixtures("tests/integration/fixtures/TestAdminDeleteUser/")()
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")

	userID := int64(1000)

	unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{PosterID: userID})

	csrf := GetCSRF(t, session, fmt.Sprintf("/admin/users/%d/edit", userID))
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/admin/users/%d/delete", userID), map[string]string{
		"_csrf": csrf,
		"purge": "true",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	assertUserDeleted(t, userID, true)
	unittest.CheckConsistencyFor(t, &user_model.User{})
}

func TestAdminViewUsersSorted(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadAdmin)

	variations := [][]string{
		{"alphabetically", "the_34-user.with.all.allowedChars", "user1", "user10", "user11"},
		{"reversealphabetically", "user9", "user8", "user5", "user40"},
		{"newest", "user40", "user39", "user38", "user37"},
		{"oldest", "user1", "user2", "user4", "user5"},
		{"recentupdate", "user2", "user1", "user40", "user39"},
		{"leastupdate", "user4", "user5", "user8", "user9"},
	}

	for _, variation := range variations {
		sortType := variation[0]
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/admin/users?sort=%s&limit=4", sortType)).AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)

		var users []api.User
		DecodeJSON(t, resp, &users)
		assert.Len(t, users, 4)
		for i, user := range users {
			// +1 because the first element is the sort type
			assert.Equalf(t, variation[i+1], user.UserName, "Sort type: %s, index %d", sortType, i+1)
		}
	}
}
