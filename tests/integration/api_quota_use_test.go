// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	quota_model "forgejo.org/models/quota"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/migration"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type quotaEnvUser struct {
	User    *user_model.User
	Session *TestSession
	Token   string
}

type quotaEnvOrgs struct {
	Unlimited api.Organization
	Limited   api.Organization
}

type quotaEnv struct {
	Admin quotaEnvUser
	User  quotaEnvUser
	Dummy quotaEnvUser

	Repo *repo_model.Repository
	Orgs quotaEnvOrgs

	cleanups []func()
}

func (e *quotaEnv) APIPathForRepo(uriFormat string, a ...any) string {
	path := fmt.Sprintf(uriFormat, a...)
	return fmt.Sprintf("/api/v1/repos/%s/%s%s", e.User.User.Name, e.Repo.Name, path)
}

func (e *quotaEnv) Cleanup() {
	for i := len(e.cleanups) - 1; i >= 0; i-- {
		e.cleanups[i]()
	}
}

func (e *quotaEnv) WithoutQuota(t *testing.T, task func(), rules ...string) {
	rule := "all"
	if rules != nil {
		rule = rules[0]
	}
	defer e.SetRuleLimit(t, rule, -1)()
	task()
}

func (e *quotaEnv) SetupWithSingleQuotaRule(t *testing.T) {
	t.Helper()

	cleaner := test.MockVariableValue(&setting.Quota.Enabled, true)
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())
	e.cleanups = append(e.cleanups, cleaner)

	// Create a default group
	cleaner = createQuotaGroup(t, "default")
	e.cleanups = append(e.cleanups, cleaner)

	// Create a single all-encompassing rule
	unlimited := int64(-1)
	ruleAll := api.CreateQuotaRuleOptions{
		Name:     "all",
		Limit:    &unlimited,
		Subjects: []string{"size:all"},
	}
	cleaner = createQuotaRule(t, ruleAll)
	e.cleanups = append(e.cleanups, cleaner)

	// Add these rules to the group
	cleaner = e.AddRuleToGroup(t, "default", "all")
	e.cleanups = append(e.cleanups, cleaner)

	// Add the user to the quota group
	cleaner = e.AddUserToGroup(t, "default", e.User.User.Name)
	e.cleanups = append(e.cleanups, cleaner)
}

func (e *quotaEnv) AddDummyUser(t *testing.T, username string) {
	t.Helper()

	userCleanup := apiCreateUser(t, username)
	e.Dummy.User = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
	e.Dummy.Session = loginUser(t, e.Dummy.User.Name)
	e.Dummy.Token = getTokenForLoggedInUser(t, e.Dummy.Session, auth_model.AccessTokenScopeAll)
	e.cleanups = append(e.cleanups, userCleanup)

	// Add the user to the "limited" group. See AddLimitedOrg
	cleaner := e.AddUserToGroup(t, "limited", username)
	e.cleanups = append(e.cleanups, cleaner)
}

func (e *quotaEnv) AddLimitedOrg(t *testing.T) {
	t.Helper()

	// Create the limited org
	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs", api.CreateOrgOption{
		UserName: "limited-org",
	}).AddTokenAuth(e.User.Token)
	resp := e.User.Session.MakeRequest(t, req, http.StatusCreated)
	DecodeJSON(t, resp, &e.Orgs.Limited)
	e.cleanups = append(e.cleanups, func() {
		req := NewRequest(t, "DELETE", "/api/v1/orgs/limited-org").
			AddTokenAuth(e.Admin.Token)
		e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
	})

	// Create a group for the org
	cleaner := createQuotaGroup(t, "limited")
	e.cleanups = append(e.cleanups, cleaner)

	// Create a single all-encompassing rule
	zero := int64(0)
	ruleDenyAll := api.CreateQuotaRuleOptions{
		Name:     "deny-all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}
	cleaner = createQuotaRule(t, ruleDenyAll)
	e.cleanups = append(e.cleanups, cleaner)

	// Add these rules to the group
	cleaner = e.AddRuleToGroup(t, "limited", "deny-all")
	e.cleanups = append(e.cleanups, cleaner)

	// Add the user to the quota group
	cleaner = e.AddUserToGroup(t, "limited", e.Orgs.Limited.UserName)
	e.cleanups = append(e.cleanups, cleaner)
}

func (e *quotaEnv) AddUnlimitedOrg(t *testing.T) {
	t.Helper()

	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs", api.CreateOrgOption{
		UserName: "unlimited-org",
	}).AddTokenAuth(e.User.Token)
	resp := e.User.Session.MakeRequest(t, req, http.StatusCreated)
	DecodeJSON(t, resp, &e.Orgs.Unlimited)
	e.cleanups = append(e.cleanups, func() {
		req := NewRequest(t, "DELETE", "/api/v1/orgs/unlimited-org").
			AddTokenAuth(e.Admin.Token)
		e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
	})
}

func (e *quotaEnv) SetupWithMultipleQuotaRules(t *testing.T) {
	t.Helper()

	cleaner := test.MockVariableValue(&setting.Quota.Enabled, true)
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())
	e.cleanups = append(e.cleanups, cleaner)

	// Create a default group
	cleaner = createQuotaGroup(t, "default")
	e.cleanups = append(e.cleanups, cleaner)

	// Create three rules: all, repo-size, and asset-size
	zero := int64(0)
	ruleAll := api.CreateQuotaRuleOptions{
		Name:     "all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}
	cleaner = createQuotaRule(t, ruleAll)
	e.cleanups = append(e.cleanups, cleaner)

	fifteenMb := int64(1024 * 1024 * 15)
	ruleRepoSize := api.CreateQuotaRuleOptions{
		Name:     "repo-size",
		Limit:    &fifteenMb,
		Subjects: []string{"size:repos:all"},
	}
	cleaner = createQuotaRule(t, ruleRepoSize)
	e.cleanups = append(e.cleanups, cleaner)

	ruleAssetSize := api.CreateQuotaRuleOptions{
		Name:     "asset-size",
		Limit:    &fifteenMb,
		Subjects: []string{"size:assets:all"},
	}
	cleaner = createQuotaRule(t, ruleAssetSize)
	e.cleanups = append(e.cleanups, cleaner)

	// Add these rules to the group
	cleaner = e.AddRuleToGroup(t, "default", "all")
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = e.AddRuleToGroup(t, "default", "repo-size")
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = e.AddRuleToGroup(t, "default", "asset-size")
	e.cleanups = append(e.cleanups, cleaner)

	// Add the user to the quota group
	cleaner = e.AddUserToGroup(t, "default", e.User.User.Name)
	e.cleanups = append(e.cleanups, cleaner)
}

func (e *quotaEnv) AddUserToGroup(t *testing.T, group, user string) func() {
	t.Helper()

	req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/%s/users/%s", group, user).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)

	return func() {
		req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/%s/users/%s", group, user).AddTokenAuth(e.Admin.Token)
		e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
	}
}

func (e *quotaEnv) SetRuleLimit(t *testing.T, rule string, limit int64) func() {
	t.Helper()

	originalRule, err := quota_model.GetRuleByName(db.DefaultContext, rule)
	require.NoError(t, err)
	assert.NotNil(t, originalRule)

	req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/admin/quota/rules/%s", rule), api.EditQuotaRuleOptions{
		Limit: &limit,
	}).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusOK)

	return func() {
		e.SetRuleLimit(t, rule, originalRule.Limit)
	}
}

func (e *quotaEnv) RemoveRuleFromGroup(t *testing.T, group, rule string) {
	t.Helper()

	req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/%s/rules/%s", group, rule).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
}

func (e *quotaEnv) AddRuleToGroup(t *testing.T, group, rule string) func() {
	t.Helper()

	req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/%s/rules/%s", group, rule).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)

	return func() {
		e.RemoveRuleFromGroup(t, group, rule)
	}
}

func prepareQuotaEnv(t *testing.T, username string) *quotaEnv {
	t.Helper()

	env := quotaEnv{}

	// Set up the admin user
	env.Admin.User = unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	env.Admin.Session = loginUser(t, env.Admin.User.Name)
	env.Admin.Token = getTokenForLoggedInUser(t, env.Admin.Session, auth_model.AccessTokenScopeAll)

	// Create a test user
	userCleanup := apiCreateUser(t, username)
	env.User.User = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
	env.User.Session = loginUser(t, env.User.User.Name)
	env.User.Token = getTokenForLoggedInUser(t, env.User.Session, auth_model.AccessTokenScopeAll)
	env.cleanups = append(env.cleanups, userCleanup)

	// Create a repository
	repo, _, repoCleanup := tests.CreateDeclarativeRepoWithOptions(t, env.User.User, tests.DeclarativeRepoOptions{})
	env.Repo = repo
	env.cleanups = append(env.cleanups, repoCleanup)

	return &env
}

func TestAPIQuotaUserCleanSlate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Quota.Enabled, true)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		env := prepareQuotaEnv(t, "qt-clean-slate")
		defer env.Cleanup()

		t.Run("branch creation", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create a branch
			req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
				BranchName: "branch-to-delete",
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusCreated)
		})
	})
}

func TestAPIQuotaEnforcement(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		testAPIQuotaEnforcement(t)
	})
}

func TestAPIQuotaCountsTowardsCorrectUser(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-correct-user-test")
		defer env.Cleanup()
		env.SetupWithSingleQuotaRule(t)

		// Create a new group, with size:all set to 0
		defer createQuotaGroup(t, "limited")()
		zero := int64(0)
		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "limited",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()
		defer env.AddRuleToGroup(t, "limited", "limited")()

		// Add the admin user to it
		defer env.AddUserToGroup(t, "limited", env.Admin.User.Name)()

		// Add the admin user as collaborator to our repo
		perm := "admin"
		req := NewRequestWithJSON(t, "PUT",
			env.APIPathForRepo("/collaborators/%s", env.Admin.User.Name),
			api.AddCollaboratorOption{
				Permission: &perm,
			}).AddTokenAuth(env.User.Token)
		env.User.Session.MakeRequest(t, req, http.StatusNoContent)

		// Now, try to push something as admin!
		req = NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
			BranchName: "admin-branch",
		}).AddTokenAuth(env.Admin.Token)
		env.Admin.Session.MakeRequest(t, req, http.StatusCreated)
	})
}

func TestAPIQuotaError(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-enforcement")
		defer env.Cleanup()
		env.SetupWithSingleQuotaRule(t)
		env.AddUnlimitedOrg(t)
		env.AddLimitedOrg(t)

		req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/forks"), api.CreateForkOption{
			Organization: &env.Orgs.Limited.UserName,
		}).AddTokenAuth(env.User.Token)
		resp := env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)

		var msg context.APIQuotaExceeded
		DecodeJSON(t, resp, &msg)

		assert.Equal(t, env.Orgs.Limited.ID, msg.UserID)
		assert.Equal(t, env.Orgs.Limited.UserName, msg.UserName)
	})
}

func testAPIQuotaEnforcement(t *testing.T) {
	env := prepareQuotaEnv(t, "quota-enforcement")
	defer env.Cleanup()
	env.SetupWithSingleQuotaRule(t)
	env.AddUnlimitedOrg(t)
	env.AddLimitedOrg(t)
	env.AddDummyUser(t, "qe-dummy")

	t.Run("#/user/repos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer env.SetRuleLimit(t, "all", 0)()

		t.Run("CREATE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", api.CreateRepoOption{
				Name:     "quota-exceeded",
				AutoInit: true,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("LIST", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/repos").AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
	})

	t.Run("#/orgs/{org}/repos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer env.SetRuleLimit(t, "all", 0)

		assertCreateRepo := func(t *testing.T, orgName, repoName string, expectedStatus int) func() {
			t.Helper()

			req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/repos", orgName), api.CreateRepoOption{
				Name: repoName,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, expectedStatus)

			return func() {
				req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s", orgName, repoName).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusNoContent)
			}
		}

		t.Run("limited", func(t *testing.T) {
			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestf(t, "GET", "/api/v1/orgs/%s/repos", env.Orgs.Unlimited.UserName).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})

			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				assertCreateRepo(t, env.Orgs.Limited.UserName, "test-repo", http.StatusRequestEntityTooLarge)
			})
		})

		t.Run("unlimited", func(t *testing.T) {
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				defer assertCreateRepo(t, env.Orgs.Unlimited.UserName, "test-repo", http.StatusCreated)()
			})
		})
	})

	t.Run("#/repos/migrate", func(t *testing.T) {
		t.Run("to:limited", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer env.SetRuleLimit(t, "all", 0)()

			req := NewRequestWithJSON(t, "POST", "/api/v1/repos/migrate", api.MigrateRepoOptions{
				CloneAddr: env.Repo.HTMLURL() + ".git",
				RepoName:  "quota-migrate",
				Service:   "forgejo",
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("to:unlimited", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer env.SetRuleLimit(t, "all", 0)()

			req := NewRequestWithJSON(t, "POST", "/api/v1/repos/migrate", api.MigrateRepoOptions{
				CloneAddr: "an-invalid-address",
				RepoName:  "quota-migrate",
				RepoOwner: env.Orgs.Unlimited.UserName,
				Service:   "forgejo",
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusUnprocessableEntity)
		})
	})

	t.Run("#/repos/{template_owner}/{template_repo}/generate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Create a template repository
		template, _, cleanup := tests.CreateDeclarativeRepoWithOptions(t, env.User.User, tests.DeclarativeRepoOptions{
			IsTemplate: optional.Some(true),
		})
		defer cleanup()

		// Drop the quota to 0
		defer env.SetRuleLimit(t, "all", 0)()

		t.Run("to: limited", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "POST", template.APIURL()+"/generate", api.GenerateRepoOption{
				Owner:      env.User.User.Name,
				Name:       "generated-repo",
				GitContent: true,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("to: unlimited", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "POST", template.APIURL()+"/generate", api.GenerateRepoOption{
				Owner:      env.Orgs.Unlimited.UserName,
				Name:       "generated-repo",
				GitContent: true,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusCreated)

			req = NewRequestf(t, "DELETE", "/api/v1/repos/%s/generated-repo", env.Orgs.Unlimited.UserName).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusNoContent)
		})
	})

	t.Run("#/repos/{username}/{reponame}", func(t *testing.T) {
		// Lets create a new repo to play with.
		repo, _, repoCleanup := tests.CreateDeclarativeRepoWithOptions(t, env.User.User, tests.DeclarativeRepoOptions{})
		defer repoCleanup()

		// Drop the quota to 0
		defer env.SetRuleLimit(t, "all", 0)()

		deleteRepo := func(t *testing.T, path string) {
			t.Helper()

			req := NewRequestf(t, "DELETE", "/api/v1/repos/%s", path).
				AddTokenAuth(env.Admin.Token)
			env.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
		}

		t.Run("GET", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s", env.User.User.Name, repo.Name).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
		t.Run("PATCH", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			desc := "Some description"
			req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s", env.User.User.Name, repo.Name), api.EditRepoOption{
				Description: &desc,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
		t.Run("DELETE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s", env.User.User.Name, repo.Name).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusNoContent)
		})

		t.Run("branches", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create a branch we can delete later
			env.WithoutQuota(t, func() {
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "to-delete",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusCreated)
			})

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/branches")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "quota-exceeded",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("{branch}", func(t *testing.T) {
				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/branches/to-delete")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/branches/to-delete")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})
			})
		})

		t.Run("contents", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			var fileSha string

			// Create a file to play with
			env.WithoutQuota(t, func() {
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents/plaything.txt"), api.CreateFileOptions{
					ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
				}).AddTokenAuth(env.User.Token)
				resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

				var r api.FileResponse
				DecodeJSON(t, resp, &r)

				fileSha = r.Content.SHA
			})

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/contents")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents"), api.ChangeFilesOptions{
					Files: []*api.ChangeFileOperation{
						{
							Operation: "create",
							Path:      "quota-exceeded.txt",
						},
					},
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("{filepath}", func(t *testing.T) {
				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/contents/plaything.txt")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("CREATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents/plaything.txt"), api.CreateFileOptions{
						ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
				t.Run("UPDATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "PUT", env.APIPathForRepo("/contents/plaything.txt"), api.UpdateFileOptions{
						ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
						DeleteFileOptions: api.DeleteFileOptions{
							SHA: fileSha,
						},
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					// Deleting a file fails, because it creates a new commit,
					// which would increase the quota use.
					req := NewRequestWithJSON(t, "DELETE", env.APIPathForRepo("/contents/plaything.txt"), api.DeleteFileOptions{
						SHA: fileSha,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
			})
		})

		t.Run("diffpatch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "PUT", env.APIPathForRepo("/contents/README.md"), api.UpdateFileOptions{
				ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
				DeleteFileOptions: api.DeleteFileOptions{
					SHA: "c0ffeebabe",
				},
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("forks", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Run("as: limited user", func(t *testing.T) {
				// Our current user (env.User) is already limited here.

				t.Run("into: limited org", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/forks"), api.CreateForkOption{
						Organization: &env.Orgs.Limited.UserName,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})

				t.Run("into: unlimited org", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/forks"), api.CreateForkOption{
						Organization: &env.Orgs.Unlimited.UserName,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusAccepted)

					deleteRepo(t, env.Orgs.Unlimited.UserName+"/"+env.Repo.Name)
				})
			})
			t.Run("as: unlimited user", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Lift the quota limits on our current user temporarily
				defer env.SetRuleLimit(t, "all", -1)()

				t.Run("into: limited org", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/forks"), api.CreateForkOption{
						Organization: &env.Orgs.Limited.UserName,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})

				t.Run("into: unlimited org", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/forks"), api.CreateForkOption{
						Organization: &env.Orgs.Unlimited.UserName,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusAccepted)

					deleteRepo(t, env.Orgs.Unlimited.UserName+"/"+env.Repo.Name)
				})
			})
		})

		t.Run("mirror-sync", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			var mirrorRepo *repo_model.Repository
			env.WithoutQuota(t, func() {
				// Create a mirror repo
				opts := migration.MigrateOptions{
					RepoName:    "test_mirror",
					Description: "Test mirror",
					Private:     false,
					Mirror:      true,
					CloneAddr:   repo_model.RepoPath(env.User.User.Name, env.Repo.Name),
					Wiki:        true,
					Releases:    false,
				}

				repo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, env.User.User, env.User.User, repo_service.CreateRepoOptions{
					Name:        opts.RepoName,
					Description: opts.Description,
					IsPrivate:   opts.Private,
					IsMirror:    opts.Mirror,
					Status:      repo_model.RepositoryBeingMigrated,
				})
				require.NoError(t, err)

				mirrorRepo = repo
			})

			req := NewRequestf(t, "POST", "/api/v1/repos/%s/mirror-sync", mirrorRepo.FullName()).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("issues", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create an issue play with
			req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/issues"), api.CreateIssueOption{
				Title: "quota test issue",
			}).AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

			var issue api.Issue
			DecodeJSON(t, resp, &issue)

			createAsset := func(filename string) (*bytes.Buffer, string) {
				buff := generateImg()
				body := &bytes.Buffer{}

				// Setup multi-part
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("attachment", filename)
				io.Copy(part, &buff)
				writer.Close()

				return body, writer.FormDataContentType()
			}

			t.Run("{index}/assets", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				t.Run("LIST", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/issues/%d/assets", issue.Index)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("CREATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					body, contentType := createAsset("overquota.png")
					req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/issues/%d/assets", issue.Index), body).
						AddTokenAuth(env.User.Token)
					req.Header.Add("Content-Type", contentType)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})

				t.Run("{attachment_id}", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					var issueAsset api.Attachment
					env.WithoutQuota(t, func() {
						body, contentType := createAsset("test.png")
						req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/issues/%d/assets", issue.Index), body).
							AddTokenAuth(env.User.Token)
						req.Header.Add("Content-Type", contentType)
						resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

						DecodeJSON(t, resp, &issueAsset)
					})

					t.Run("GET", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "GET", env.APIPathForRepo("/issues/%d/assets/%d", issue.Index, issueAsset.ID)).
							AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusOK)
					})
					t.Run("UPDATE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/issues/%d/assets/%d", issue.Index, issueAsset.ID), api.EditAttachmentOptions{
							Name: "new-name.png",
						}).AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusCreated)
					})
					t.Run("DELETE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "DELETE", env.APIPathForRepo("/issues/%d/assets/%d", issue.Index, issueAsset.ID)).
							AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusNoContent)
					})
				})
			})

			t.Run("comments/{id}/assets", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Create a new comment!
				var comment api.Comment
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/issues/%d/comments", issue.Index), api.CreateIssueCommentOption{
					Body: "This is a comment",
				}).AddTokenAuth(env.User.Token)
				resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)
				DecodeJSON(t, resp, &comment)

				t.Run("LIST", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/issues/comments/%d/assets", comment.ID)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("CREATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					body, contentType := createAsset("overquota.png")
					req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/issues/comments/%d/assets", comment.ID), body).
						AddTokenAuth(env.User.Token)
					req.Header.Add("Content-Type", contentType)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})

				t.Run("{attachment_id}", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					var attachment api.Attachment
					env.WithoutQuota(t, func() {
						body, contentType := createAsset("test.png")
						req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/issues/comments/%d/assets", comment.ID), body).
							AddTokenAuth(env.User.Token)
						req.Header.Add("Content-Type", contentType)
						resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)
						DecodeJSON(t, resp, &attachment)
					})

					t.Run("GET", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "GET", env.APIPathForRepo("/issues/comments/%d/assets/%d", comment.ID, attachment.ID)).
							AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusOK)
					})
					t.Run("UPDATE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/issues/comments/%d/assets/%d", comment.ID, attachment.ID), api.EditAttachmentOptions{
							Name: "new-name.png",
						}).AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusCreated)
					})
					t.Run("DELETE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "DELETE", env.APIPathForRepo("/issues/comments/%d/assets/%d", comment.ID, attachment.ID)).
							AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusNoContent)
					})
				})
			})
		})

		t.Run("pulls", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Fork the repository into the unlimited org first
			req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/forks"), api.CreateForkOption{
				Organization: &env.Orgs.Unlimited.UserName,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusAccepted)

			defer deleteRepo(t, env.Orgs.Unlimited.UserName+"/"+env.Repo.Name)

			// Create a pull request!
			//
			// Creating a pull request this way does not increase the space of
			// the base repo, so is not subject to quota enforcement.

			req = NewRequestWithJSON(t, "POST", env.APIPathForRepo("/pulls"), api.CreatePullRequestOption{
				Base:  "main",
				Title: "test-pr",
				Head:  fmt.Sprintf("%s:main", env.Orgs.Unlimited.UserName),
			}).AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

			var pr api.PullRequest
			DecodeJSON(t, resp, &pr)

			t.Run("{index}", func(t *testing.T) {
				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/pulls/%d", pr.Index)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("UPDATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/pulls/%d", pr.Index), api.EditPullRequestOption{
						Title: "Updated title",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusCreated)
				})

				t.Run("merge", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/pulls/%d/merge", pr.Index), forms.MergePullRequestForm{
						Do: "merge",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
			})
		})

		t.Run("releases", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			var releaseID int64

			// Create a release so that there's something to play with.
			env.WithoutQuota(t, func() {
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
					TagName: "play-release-tag",
					Title:   "play-release",
				}).AddTokenAuth(env.User.Token)
				resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

				var q api.Release
				DecodeJSON(t, resp, &q)

				releaseID = q.ID
			})

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/releases")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
					TagName: "play-release-tag-two",
					Title:   "play-release-two",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("tags/{tag}", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Create a release for our subtests
				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
						TagName: "play-release-tag-subtest",
						Title:   "play-release-subtest",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusCreated)
				})

				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/releases/tags/play-release-tag-subtest")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/releases/tags/play-release-tag-subtest")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})
			})

			t.Run("{id}", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				var tmpReleaseID int64

				// Create a release so that there's something to play with.
				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
						TagName: "tmp-tag",
						Title:   "tmp-release",
					}).AddTokenAuth(env.User.Token)
					resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

					var q api.Release
					DecodeJSON(t, resp, &q)

					tmpReleaseID = q.ID
				})

				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/releases/%d", tmpReleaseID)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("UPDATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/releases/%d", tmpReleaseID), api.EditReleaseOption{
						TagName: "tmp-tag-two",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/releases/%d", tmpReleaseID)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})

				t.Run("assets", func(t *testing.T) {
					t.Run("LIST", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "GET", env.APIPathForRepo("/releases/%d/assets", releaseID)).
							AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusOK)
					})
					t.Run("CREATE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						body := strings.NewReader("hello world")
						req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/releases/%d/assets?name=bar.txt", releaseID), body).
							AddTokenAuth(env.User.Token)
						req.Header.Add("Content-Type", "text/plain")
						env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
					})

					t.Run("{attachment_id}", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						var attachmentID int64

						// Create an attachment to play with
						env.WithoutQuota(t, func() {
							body := strings.NewReader("hello world")
							req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/releases/%d/assets?name=foo.txt", releaseID), body).
								AddTokenAuth(env.User.Token)
							req.Header.Add("Content-Type", "text/plain")
							resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

							var q api.Attachment
							DecodeJSON(t, resp, &q)

							attachmentID = q.ID
						})

						t.Run("GET", func(t *testing.T) {
							defer tests.PrintCurrentTest(t)()

							req := NewRequest(t, "GET", env.APIPathForRepo("/releases/%d/assets/%d", releaseID, attachmentID)).
								AddTokenAuth(env.User.Token)
							env.User.Session.MakeRequest(t, req, http.StatusOK)
						})
						t.Run("UPDATE", func(t *testing.T) {
							defer tests.PrintCurrentTest(t)()

							req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/releases/%d/assets/%d", releaseID, attachmentID), api.EditAttachmentOptions{
								Name: "new-name.txt",
							}).AddTokenAuth(env.User.Token)
							env.User.Session.MakeRequest(t, req, http.StatusCreated)
						})
						t.Run("DELETE", func(t *testing.T) {
							defer tests.PrintCurrentTest(t)()

							req := NewRequest(t, "DELETE", env.APIPathForRepo("/releases/%d/assets/%d", releaseID, attachmentID)).
								AddTokenAuth(env.User.Token)
							env.User.Session.MakeRequest(t, req, http.StatusNoContent)
						})
					})
				})
			})
		})

		t.Run("tags", func(t *testing.T) {
			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/tags")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/tags"), api.CreateTagOption{
					TagName: "tag-quota-test",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("{tag}", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/tags"), api.CreateTagOption{
						TagName: "tag-quota-test-2",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusCreated)
				})

				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/tags/tag-quota-test-2")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/tags/tag-quota-test-2")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})
			})
		})

		t.Run("transfer", func(t *testing.T) {
			t.Run("to: limited", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Create a repository to transfer
				repo, _, cleanup := tests.CreateDeclarativeRepoWithOptions(t, env.User.User, tests.DeclarativeRepoOptions{})
				defer cleanup()

				// Initiate repo transfer
				req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/transfer", env.User.User.Name, repo.Name), api.TransferRepoOption{
					NewOwner: env.Dummy.User.Name,
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)

				// Initiate it outside of quotas, so we can test accept/reject.
				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/transfer", env.User.User.Name, repo.Name), api.TransferRepoOption{
						NewOwner: env.Dummy.User.Name,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusCreated)
				}, "deny-all") // a bit of a hack, sorry!

				// Try to accept the repo transfer
				req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/transfer/accept", env.User.User.Name, repo.Name)).
					AddTokenAuth(env.Dummy.Token)
				env.Dummy.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)

				// Then reject it.
				req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/transfer/reject", env.User.User.Name, repo.Name)).
					AddTokenAuth(env.Dummy.Token)
				env.Dummy.Session.MakeRequest(t, req, http.StatusOK)
			})

			t.Run("to: unlimited", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Disable the quota for the dummy user
				defer env.SetRuleLimit(t, "deny-all", -1)()

				// Create a repository to transfer
				repo, _, cleanup := tests.CreateDeclarativeRepoWithOptions(t, env.User.User, tests.DeclarativeRepoOptions{})
				defer cleanup()

				// Initiate repo transfer
				req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/transfer", env.User.User.Name, repo.Name), api.TransferRepoOption{
					NewOwner: env.Dummy.User.Name,
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusCreated)

				// Accept the repo transfer
				req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/transfer/accept", env.User.User.Name, repo.Name)).
					AddTokenAuth(env.Dummy.Token)
				env.Dummy.Session.MakeRequest(t, req, http.StatusAccepted)
			})
		})
	})

	t.Run("#/packages/{owner}/{type}/{name}/{version}", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer env.SetRuleLimit(t, "all", 0)()

		// Create a generic package to play with
		env.WithoutQuota(t, func() {
			body := strings.NewReader("forgejo is awesome")
			req := NewRequestWithBody(t, "PUT", fmt.Sprintf("/api/packages/%s/generic/quota-test/1.0.0/test.txt", env.User.User.Name), body).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusCreated)
		})

		t.Run("CREATE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			body := strings.NewReader("forgejo is awesome")
			req := NewRequestWithBody(t, "PUT", fmt.Sprintf("/api/packages/%s/generic/quota-test/1.0.0/overquota.txt", env.User.User.Name), body).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("GET", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestf(t, "GET", "/api/v1/packages/%s/generic/quota-test/1.0.0", env.User.User.Name).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
		t.Run("DELETE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestf(t, "DELETE", "/api/v1/packages/%s/generic/quota-test/1.0.0", env.User.User.Name).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusNoContent)
		})
	})
}

func TestAPIQuotaOrgQuotaQuery(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-enforcement")
		defer env.Cleanup()

		env.SetupWithSingleQuotaRule(t)
		env.AddUnlimitedOrg(t)
		env.AddLimitedOrg(t)

		// Look at the quota use of our user, and the unlimited org, for later
		// comparison.
		var userInfo api.QuotaInfo
		req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(env.User.Token)
		resp := env.User.Session.MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &userInfo)

		var orgInfo api.QuotaInfo
		req = NewRequestf(t, "GET", "/api/v1/orgs/%s/quota", env.Orgs.Unlimited.Name).
			AddTokenAuth(env.User.Token)
		resp = env.User.Session.MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &orgInfo)

		assert.Positive(t, userInfo.Used.Size.Repos.Public)
		assert.EqualValues(t, 0, orgInfo.Used.Size.Repos.Public)
	})
}

func TestAPIQuotaUserBasics(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-enforcement")
		defer env.Cleanup()

		env.SetupWithMultipleQuotaRules(t)

		t.Run("quota usage change", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaInfo
			DecodeJSON(t, resp, &q)

			assert.Positive(t, q.Used.Size.Repos.Public)
			assert.Empty(t, q.Groups[0].Name)
			assert.Empty(t, q.Groups[0].Rules[0].Name)

			t.Run("admin view", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestf(t, "GET", "/api/v1/admin/users/%s/quota", env.User.User.Name).AddTokenAuth(env.Admin.Token)
				resp := env.Admin.Session.MakeRequest(t, req, http.StatusOK)

				var q api.QuotaInfo
				DecodeJSON(t, resp, &q)

				assert.Positive(t, q.Used.Size.Repos.Public)

				assert.NotEmpty(t, q.Groups[0].Name)
				assert.NotEmpty(t, q.Groups[0].Rules[0].Name)
			})
		})

		t.Run("quota check passing", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/check?subject=size:repos:all").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q bool
			DecodeJSON(t, resp, &q)

			assert.True(t, q)
		})

		t.Run("quota check failing after limit change", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer env.SetRuleLimit(t, "repo-size", 0)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/check?subject=size:repos:all").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q bool
			DecodeJSON(t, resp, &q)

			assert.False(t, q)
		})

		t.Run("quota enforcement", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer env.SetRuleLimit(t, "repo-size", 0)()

			t.Run("repoCreateFile", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents/new-file.txt"), api.CreateFileOptions{
					ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("repoCreateBranch", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "new-branch",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("repoDeleteBranch", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Temporarily disable quota checking
				defer env.SetRuleLimit(t, "repo-size", -1)()
				defer env.SetRuleLimit(t, "all", -1)()

				// Create a branch
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "branch-to-delete",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusCreated)

				// Set the limit back. No need to defer, the first one will set it
				// back to the correct value.
				env.SetRuleLimit(t, "all", 0)
				env.SetRuleLimit(t, "repo-size", 0)

				// Deleting a branch does not incur quota enforcement
				req = NewRequest(t, "DELETE", env.APIPathForRepo("/branches/branch-to-delete")).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusNoContent)
			})
		})
	})
}
