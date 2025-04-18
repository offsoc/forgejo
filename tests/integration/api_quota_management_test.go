// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	quota_model "forgejo.org/models/quota"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIQuotaDisabled(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	session := loginUser(t, user.Name)

	req := NewRequest(t, "GET", "/api/v1/user/quota")
	session.MakeRequest(t, req, http.StatusNotFound)
}

func apiCreateUser(t *testing.T, username string) func() {
	t.Helper()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	session := loginUser(t, admin.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	mustChangePassword := false
	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/users", api.CreateUserOption{
		Email:              "api+" + username + "@example.com",
		Username:           username,
		Password:           "password",
		MustChangePassword: &mustChangePassword,
	}).AddTokenAuth(token)
	session.MakeRequest(t, req, http.StatusCreated)

	return func() {
		req := NewRequest(t, "DELETE", "/api/v1/admin/users/"+username+"?purge=true").AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusNoContent)
	}
}

func TestAPIQuotaCreateGroupWithRules(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	// Create two rules in advance
	unlimited := int64(-1)
	defer createQuotaRule(t, api.CreateQuotaRuleOptions{
		Name:     "unlimited",
		Limit:    &unlimited,
		Subjects: []string{"size:all"},
	})()
	zero := int64(0)
	defer createQuotaRule(t, api.CreateQuotaRuleOptions{
		Name:     "deny-git-lfs",
		Limit:    &zero,
		Subjects: []string{"size:git:lfs"},
	})()

	// Log in as admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// Create a new group, with rules specified
	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", api.CreateQuotaGroupOptions{
		Name: "group-with-rules",
		Rules: []api.CreateQuotaRuleOptions{
			// First: an existing group, unlimited, name only
			{
				Name: "unlimited",
			},
			// Second: an existing group, deny-git-lfs, with different params
			{
				Name:  "deny-git-lfs",
				Limit: &unlimited,
			},
			// Third: an entirely new group
			{
				Name:     "new-rule",
				Subjects: []string{"size:assets:all"},
			},
		},
	}).AddTokenAuth(adminToken)
	resp := adminSession.MakeRequest(t, req, http.StatusCreated)
	defer func() {
		req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/group-with-rules").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "DELETE", "/api/v1/admin/quota/rules/new-rule").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)
	}()

	// Verify that we created a group with rules included
	var q api.QuotaGroup
	DecodeJSON(t, resp, &q)

	assert.Equal(t, "group-with-rules", q.Name)
	assert.Len(t, q.Rules, 3)

	// Verify that the previously existing rules are unchanged
	rule, err := quota_model.GetRuleByName(db.DefaultContext, "unlimited")
	require.NoError(t, err)
	assert.NotNil(t, rule)
	assert.EqualValues(t, -1, rule.Limit)
	assert.Equal(t, quota_model.LimitSubjects{quota_model.LimitSubjectSizeAll}, rule.Subjects)

	rule, err = quota_model.GetRuleByName(db.DefaultContext, "deny-git-lfs")
	require.NoError(t, err)
	assert.NotNil(t, rule)
	assert.EqualValues(t, 0, rule.Limit)
	assert.Equal(t, quota_model.LimitSubjects{quota_model.LimitSubjectSizeGitLFS}, rule.Subjects)

	// Verify that the new rule was also created
	rule, err = quota_model.GetRuleByName(db.DefaultContext, "new-rule")
	require.NoError(t, err)
	assert.NotNil(t, rule)
	assert.EqualValues(t, 0, rule.Limit)
	assert.Equal(t, quota_model.LimitSubjects{quota_model.LimitSubjectSizeAssetsAll}, rule.Subjects)

	t.Run("invalid rule spec", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", api.CreateQuotaGroupOptions{
			Name: "group-with-invalid-rule-spec",
			Rules: []api.CreateQuotaRuleOptions{
				{
					Name:     "rule-with-wrong-spec",
					Subjects: []string{"valid:false"},
				},
			},
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusUnprocessableEntity)
	})
}

func TestAPIQuotaEmptyState(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	username := "quota-empty-user"
	defer apiCreateUser(t, username)()
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	t.Run("#/admin/users/quota-empty-user/quota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
		adminSession := loginUser(t, admin.Name)
		adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

		req := NewRequest(t, "GET", "/api/v1/admin/users/quota-empty-user/quota").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaInfo
		DecodeJSON(t, resp, &q)

		assert.Equal(t, api.QuotaUsed{}, q.Used)
		assert.Empty(t, q.Groups)
	})

	t.Run("#/user/quota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaInfo
		DecodeJSON(t, resp, &q)

		assert.Equal(t, api.QuotaUsed{}, q.Used)
		assert.Empty(t, q.Groups)

		t.Run("#/user/quota/artifacts", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/artifacts").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedArtifactList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})

		t.Run("#/user/quota/attachments", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/attachments").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedAttachmentList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})

		t.Run("#/user/quota/packages", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/packages").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedPackageList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})
	})
}

func createQuotaRule(t *testing.T, opts api.CreateQuotaRuleOptions) func() {
	t.Helper()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/rules", opts).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)

	return func() {
		req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/rules/%s", opts.Name).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)
	}
}

func createQuotaGroup(t *testing.T, name string) func() {
	t.Helper()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", api.CreateQuotaGroupOptions{
		Name: name,
	}).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)

	return func() {
		req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/%s", name).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)
	}
}

func TestAPIQuotaAdminRoutesRules(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	zero := int64(0)
	oneKb := int64(1024)

	t.Run("adminCreateQuotaRule", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/rules", api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		}).AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusCreated)
		defer func() {
			req := NewRequest(t, "DELETE", "/api/v1/admin/quota/rules/deny-all").AddTokenAuth(adminToken)
			adminSession.MakeRequest(t, req, http.StatusNoContent)
		}()

		var q api.QuotaRuleInfo
		DecodeJSON(t, resp, &q)

		assert.Equal(t, "deny-all", q.Name)
		assert.EqualValues(t, 0, q.Limit)
		assert.Equal(t, []string{"size:all"}, q.Subjects)

		rule, err := quota_model.GetRuleByName(db.DefaultContext, "deny-all")
		require.NoError(t, err)
		assert.EqualValues(t, 0, rule.Limit)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("missing options", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/rules", nil).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusUnprocessableEntity)
			})

			t.Run("invalid subjects", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/rules", api.CreateQuotaRuleOptions{
					Name:     "invalid-subjects",
					Limit:    &zero,
					Subjects: []string{"valid:false"},
				}).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusUnprocessableEntity)
			})

			t.Run("trying to add an existing rule", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				rule := api.CreateQuotaRuleOptions{
					Name:  "double-rule",
					Limit: &zero,
				}

				defer createQuotaRule(t, rule)()

				req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/rules", rule).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusConflict)
			})
		})
	})

	t.Run("adminDeleteQuotaRule", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})

		req := NewRequest(t, "DELETE", "/api/v1/admin/quota/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		rule, err := quota_model.GetRuleByName(db.DefaultContext, "deny-all")
		require.NoError(t, err)
		assert.Nil(t, rule)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("nonexistent rule", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/rules/does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminEditQuotaRule", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()

		req := NewRequestWithJSON(t, "PATCH", "/api/v1/admin/quota/rules/deny-all", api.EditQuotaRuleOptions{
			Limit: &oneKb,
		}).AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaRuleInfo
		DecodeJSON(t, resp, &q)
		assert.EqualValues(t, 1024, q.Limit)

		rule, err := quota_model.GetRuleByName(db.DefaultContext, "deny-all")
		require.NoError(t, err)
		assert.EqualValues(t, 1024, rule.Limit)

		t.Run("no options", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "PATCH", "/api/v1/admin/quota/rules/deny-all", nil).AddTokenAuth(adminToken)
			adminSession.MakeRequest(t, req, http.StatusOK)
		})

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("nonexistent rule", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "PATCH", "/api/v1/admin/quota/rules/does-not-exist", api.EditQuotaRuleOptions{
					Limit: &oneKb,
				}).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("invalid subjects", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "PATCH", "/api/v1/admin/quota/rules/deny-all", api.EditQuotaRuleOptions{
					Subjects: &[]string{"valid:false"},
				}).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusUnprocessableEntity)
			})
		})
	})

	t.Run("adminListQuotaRules", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()

		req := NewRequest(t, "GET", "/api/v1/admin/quota/rules").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var rules []api.QuotaRuleInfo
		DecodeJSON(t, resp, &rules)

		assert.Len(t, rules, 1)
		assert.Equal(t, "deny-all", rules[0].Name)
		assert.EqualValues(t, 0, rules[0].Limit)
	})
}

func TestAPIQuotaAdminRoutesGroups(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	zero := int64(0)

	ruleDenyAll := api.CreateQuotaRuleOptions{
		Name:     "deny-all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}

	username := "quota-test-user"
	defer apiCreateUser(t, username)()

	t.Run("adminCreateQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", api.CreateQuotaGroupOptions{
			Name: "default",
		}).AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusCreated)
		defer func() {
			req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default").AddTokenAuth(adminToken)
			adminSession.MakeRequest(t, req, http.StatusNoContent)
		}()

		var q api.QuotaGroup
		DecodeJSON(t, resp, &q)

		assert.Equal(t, "default", q.Name)
		assert.Empty(t, q.Rules)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		require.NoError(t, err)
		assert.Equal(t, "default", group.Name)
		assert.Empty(t, group.Rules)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("missing options", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", nil).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusUnprocessableEntity)
			})

			t.Run("trying to add an existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				defer createQuotaGroup(t, "duplicate")()

				req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", api.CreateQuotaGroupOptions{
					Name: "duplicate",
				}).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusConflict)
			})
		})
	})

	t.Run("adminDeleteQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		createQuotaGroup(t, "default")

		req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		require.NoError(t, err)
		assert.Nil(t, group)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminAddRuleToQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		require.NoError(t, err)
		assert.Len(t, group.Rules, 1)
		assert.Equal(t, "deny-all", group.Rules[0].Name)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/does-not-exist/rules/deny-all").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("non-existing rule", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/rules/does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminRemoveRuleFromQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		require.NoError(t, err)
		assert.Equal(t, "default", group.Name)
		assert.Empty(t, group.Rules)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/does-not-exist/rules/deny-all").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("non-existing rule", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default/rules/does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("rule not in group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				defer createQuotaRule(t, api.CreateQuotaRuleOptions{
					Name:  "rule-not-in-group",
					Limit: &zero,
				})()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default/rules/rule-not-in-group").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminGetQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "GET", "/api/v1/admin/quota/groups/default").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaGroup
		DecodeJSON(t, resp, &q)

		assert.Equal(t, "default", q.Name)
		assert.Len(t, q.Rules, 1)
		assert.Equal(t, "deny-all", q.Rules[0].Name)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/api/v1/admin/quota/groups/does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminListQuotaGroups", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "GET", "/api/v1/admin/quota/groups").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaGroupList
		DecodeJSON(t, resp, &q)

		assert.Len(t, q, 1)
		assert.Equal(t, "default", q[0].Name)
		assert.Len(t, q[0].Rules, 1)
		assert.Equal(t, "deny-all", q[0].Rules[0].Name)
	})

	t.Run("adminAddUserToQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()

		req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/default/users/%s", username).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})

		groups, err := quota_model.GetGroupsForUser(db.DefaultContext, user.ID)
		require.NoError(t, err)
		assert.Len(t, groups, 1)
		assert.Equal(t, "default", groups[0].Name)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/does-not-exist/users/%s", username).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("non-existing user", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/users/this-user-does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("user already added", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/users/user1").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNoContent)

				req = NewRequest(t, "PUT", "/api/v1/admin/quota/groups/default/users/user1").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusConflict)
			})
		})
	})

	t.Run("adminRemoveUserFromQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()

		req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/default/users/%s", username).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		req = NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/default/users/%s", username).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
		groups, err := quota_model.GetGroupsForUser(db.DefaultContext, user.ID)
		require.NoError(t, err)
		assert.Empty(t, groups)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/does-not-exist/users/%s", username).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("non-existing user", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default/users/does-not-exist").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("user not in group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default/users/user1").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminListUsersInQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()

		req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/default/users/%s", username).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "GET", "/api/v1/admin/quota/groups/default/users").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q []api.User
		DecodeJSON(t, resp, &q)

		assert.Len(t, q, 1)
		assert.Equal(t, username, q[0].UserName)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/api/v1/admin/quota/groups/does-not-exist/users").AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("adminSetUserQuotaGroups", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaGroup(t, "test-1")()
		defer createQuotaGroup(t, "test-2")()

		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/admin/users/%s/quota/groups", username), api.SetUserQuotaGroupsOptions{
			Groups: &[]string{"default", "test-1", "test-2"},
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})

		groups, err := quota_model.GetGroupsForUser(db.DefaultContext, user.ID)
		require.NoError(t, err)
		assert.Len(t, groups, 3)

		t.Run("unhappy path", func(t *testing.T) {
			t.Run("non-existing user", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", "/api/v1/admin/users/does-not-exist/quota/groups", api.SetUserQuotaGroupsOptions{
					Groups: &[]string{"default", "test-1", "test-2"},
				}).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("non-existing group", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/admin/users/%s/quota/groups", username), api.SetUserQuotaGroupsOptions{
					Groups: &[]string{"default", "test-1", "test-2", "this-group-does-not-exist"},
				}).AddTokenAuth(adminToken)
				adminSession.MakeRequest(t, req, http.StatusUnprocessableEntity)
			})
		})
	})
}

func TestAPIQuotaUserRoutes(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// Create a test user
	username := "quota-test-user-routes"
	defer apiCreateUser(t, username)()
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Set up rules & groups for the user
	defer createQuotaGroup(t, "user-routes-deny")()
	defer createQuotaGroup(t, "user-routes-1kb")()

	zero := int64(0)
	ruleDenyAll := api.CreateQuotaRuleOptions{
		Name:     "user-routes-deny-all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}
	defer createQuotaRule(t, ruleDenyAll)()
	oneKb := int64(1024)
	rule1KbStuff := api.CreateQuotaRuleOptions{
		Name:     "user-routes-1kb",
		Limit:    &oneKb,
		Subjects: []string{"size:assets:attachments:releases", "size:assets:packages:all", "size:git:lfs"},
	}
	defer createQuotaRule(t, rule1KbStuff)()

	req := NewRequest(t, "PUT", "/api/v1/admin/quota/groups/user-routes-deny/rules/user-routes-deny-all").AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusNoContent)
	req = NewRequest(t, "PUT", "/api/v1/admin/quota/groups/user-routes-1kb/rules/user-routes-1kb").AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusNoContent)

	req = NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/user-routes-deny/users/%s", username).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusNoContent)
	req = NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/user-routes-1kb/users/%s", username).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusNoContent)

	t.Run("userGetQuota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaInfo
		DecodeJSON(t, resp, &q)

		assert.Len(t, q.Groups, 2)
		assert.Len(t, q.Groups[0].Rules, 1)
		assert.Len(t, q.Groups[1].Rules, 1)
	})
}
