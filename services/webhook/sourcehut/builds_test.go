// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package sourcehut

import (
	"context"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	unit_model "code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	webhook_module "code.gitea.io/gitea/modules/webhook"
	repo_service "code.gitea.io/gitea/services/repository"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/services/webhook/shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitInit(t testing.TB) {
	if setting.Git.HomePath != "" {
		return
	}
	t.Cleanup(test.MockVariableValue(&setting.Git.HomePath, t.TempDir()))
	require.NoError(t, git.InitSimple(context.Background()))
}

func TestSourcehutBuildsPayload(t *testing.T) {
	gitInit(t)
	defer test.MockVariableValue(&setting.RepoRootPath, ".")()
	defer test.MockVariableValue(&setting.AppURL, "https://example.forgejo.org/")()

	repo := &api.Repository{
		HTMLURL:  "http://localhost:3000/testdata/repo",
		Name:     "repo",
		FullName: "testdata/repo",
		Owner: &api.User{
			UserName: "testdata",
		},
		CloneURL: "http://localhost:3000/testdata/repo.git",
	}

	pc := sourcehutConvertor{
		ctx: git.DefaultContext,
		meta: BuildsMeta{
			ManifestPath: "adjust me in each test",
			Visibility:   "UNLISTED",
			Secrets:      true,
		},
	}
	t.Run("Create/branch", func(t *testing.T) {
		p := &api.CreatePayload{
			Sha:     "58771003157b81abc6bf41df0c5db4147a3e3c83",
			Ref:     "refs/heads/test",
			RefType: "branch",
			Repo:    repo,
		}

		pc.meta.ManifestPath = "simple.yml"
		pl, err := pc.Create(p)
		require.NoError(t, err)

		assert.Equal(t, `sources:
    - http://localhost:3000/testdata/repo.git#58771003157b81abc6bf41df0c5db4147a3e3c83
environment:
    BUILD_SUBMITTER: forgejo
    BUILD_SUBMITTER_URL: https://example.forgejo.org/
    GIT_REF: refs/heads/test
image: alpine/edge
tasks:
    - say-hello: |
        echo hello
    - say-world: echo world
`, pl.Variables.Manifest)
		assert.Equal(t, buildsVariables{
			Manifest:   pl.Variables.Manifest, // the manifest correctness is checked above, for nicer diff on error
			Note:       "branch test created",
			Tags:       []string{"testdata/repo", "branch/test", "simple.yml"},
			Secrets:    true,
			Execute:    true,
			Visibility: "UNLISTED",
		}, pl.Variables)
	})
	t.Run("Create/tag", func(t *testing.T) {
		p := &api.CreatePayload{
			Sha:     "58771003157b81abc6bf41df0c5db4147a3e3c83",
			Ref:     "refs/tags/v1.0.0",
			RefType: "tag",
			Repo:    repo,
		}

		pc.meta.ManifestPath = "simple.yml"
		pl, err := pc.Create(p)
		require.NoError(t, err)

		assert.Equal(t, `sources:
    - http://localhost:3000/testdata/repo.git#58771003157b81abc6bf41df0c5db4147a3e3c83
environment:
    BUILD_SUBMITTER: forgejo
    BUILD_SUBMITTER_URL: https://example.forgejo.org/
    GIT_REF: refs/tags/v1.0.0
image: alpine/edge
tasks:
    - say-hello: |
        echo hello
    - say-world: echo world
`, pl.Variables.Manifest)
		assert.Equal(t, buildsVariables{
			Manifest:   pl.Variables.Manifest, // the manifest correctness is checked above, for nicer diff on error
			Note:       "tag v1.0.0 created",
			Tags:       []string{"testdata/repo", "tag/v1.0.0", "simple.yml"},
			Secrets:    true,
			Execute:    true,
			Visibility: "UNLISTED",
		}, pl.Variables)
	})

	t.Run("Delete", func(t *testing.T) {
		p := &api.DeletePayload{}

		pl, err := pc.Delete(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Fork", func(t *testing.T) {
		p := &api.ForkPayload{}

		pl, err := pc.Fork(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Push/simple", func(t *testing.T) {
		p := &api.PushPayload{
			Ref: "refs/heads/main",
			HeadCommit: &api.PayloadCommit{
				ID:      "58771003157b81abc6bf41df0c5db4147a3e3c83",
				Message: "add simple",
			},
			Repo: repo,
		}

		pc.meta.ManifestPath = "simple.yml"
		pl, err := pc.Push(p)
		require.NoError(t, err)

		assert.Equal(t, `sources:
    - http://localhost:3000/testdata/repo.git#58771003157b81abc6bf41df0c5db4147a3e3c83
environment:
    BUILD_SUBMITTER: forgejo
    BUILD_SUBMITTER_URL: https://example.forgejo.org/
    GIT_REF: refs/heads/main
image: alpine/edge
tasks:
    - say-hello: |
        echo hello
    - say-world: echo world
`, pl.Variables.Manifest)
		assert.Equal(t, buildsVariables{
			Manifest:   pl.Variables.Manifest, // the manifest correctness is checked above, for nicer diff on error
			Note:       "add simple",
			Tags:       []string{"testdata/repo", "branch/main", "simple.yml"},
			Secrets:    true,
			Execute:    true,
			Visibility: "UNLISTED",
		}, pl.Variables)
	})
	t.Run("Push/complex", func(t *testing.T) {
		p := &api.PushPayload{
			Ref: "refs/heads/main",
			HeadCommit: &api.PayloadCommit{
				ID:      "b0404943256a1f5a50c3726f4378756b4c1e5704",
				Message: "replace simple with complex",
			},
			Repo: repo,
		}

		pc.meta.ManifestPath = "complex.yaml"
		pc.meta.Visibility = "PRIVATE"
		pc.meta.Secrets = false
		pl, err := pc.Push(p)
		require.NoError(t, err)

		assert.Equal(t, `sources:
    - http://localhost:3000/testdata/repo.git#b0404943256a1f5a50c3726f4378756b4c1e5704
environment:
    BUILD_SUBMITTER: forgejo
    BUILD_SUBMITTER_URL: https://example.forgejo.org/
    GIT_REF: refs/heads/main
    deploy: synapse@synapse-bt.org
image: archlinux
packages:
    - nodejs
    - npm
    - rsync
secrets:
    - 7ebab768-e5e4-4c9d-ba57-ec41a72c5665
tasks: []
triggers:
    - condition: failure
      action: email
      to: Jim Jimson <jim@example.org>
    # report back the status
    - condition: always
      action: webhook
      url: https://hook.example.org
`, pl.Variables.Manifest)
		assert.Equal(t, buildsVariables{
			Manifest:   pl.Variables.Manifest, // the manifest correctness is checked above, for nicer diff on error
			Note:       "replace simple with complex",
			Tags:       []string{"testdata/repo", "branch/main", "complex.yaml"},
			Secrets:    false,
			Execute:    true,
			Visibility: "PRIVATE",
		}, pl.Variables)
	})

	t.Run("Push/error", func(t *testing.T) {
		p := &api.PushPayload{
			Ref: "refs/heads/main",
			HeadCommit: &api.PayloadCommit{
				ID:      "58771003157b81abc6bf41df0c5db4147a3e3c83",
				Message: "add simple",
			},
			Repo: repo,
		}

		pc.meta.ManifestPath = "non-existing.yml"
		pl, err := pc.Push(p)
		require.NoError(t, err)

		assert.Equal(t, graphqlPayload[buildsVariables]{
			Error: "testdata/repo:refs/heads/main could not open manifest \"non-existing.yml\"",
		}, pl)
	})

	t.Run("Issue", func(t *testing.T) {
		p := &api.IssuePayload{}

		p.Action = api.HookIssueOpened
		pl, err := pc.Issue(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)

		p.Action = api.HookIssueClosed
		pl, err = pc.Issue(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("IssueComment", func(t *testing.T) {
		p := &api.IssueCommentPayload{}

		pl, err := pc.IssueComment(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("PullRequest", func(t *testing.T) {
		p := &api.PullRequestPayload{}

		pl, err := pc.PullRequest(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("PullRequestComment", func(t *testing.T) {
		p := &api.IssueCommentPayload{
			IsPull: true,
		}

		pl, err := pc.IssueComment(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Review", func(t *testing.T) {
		p := &api.PullRequestPayload{}
		p.Action = api.HookIssueReviewed

		pl, err := pc.Review(p, webhook_module.HookEventPullRequestReviewApproved)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Repository", func(t *testing.T) {
		p := &api.RepositoryPayload{}

		pl, err := pc.Repository(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Package", func(t *testing.T) {
		p := &api.PackagePayload{}

		pl, err := pc.Package(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Wiki", func(t *testing.T) {
		p := &api.WikiPayload{}

		p.Action = api.HookWikiCreated
		pl, err := pc.Wiki(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)

		p.Action = api.HookWikiEdited
		pl, err = pc.Wiki(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)

		p.Action = api.HookWikiDeleted
		pl, err = pc.Wiki(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})

	t.Run("Release", func(t *testing.T) {
		p := &api.ReleasePayload{}

		pl, err := pc.Release(p)
		require.Equal(t, shared.ErrPayloadTypeNotSupported, err)
		require.Equal(t, graphqlPayload[buildsVariables]{}, pl)
	})
}

func TestSourcehutJSONPayload(t *testing.T) {
	gitInit(t)
	defer test.MockVariableValue(&setting.RepoRootPath, ".")()
	defer test.MockVariableValue(&setting.AppURL, "https://example.forgejo.org/")()

	repo := &api.Repository{
		HTMLURL:  "http://localhost:3000/testdata/repo",
		Name:     "repo",
		FullName: "testdata/repo",
		Owner: &api.User{
			UserName: "testdata",
		},
		CloneURL: "http://localhost:3000/testdata/repo.git",
	}

	p := &api.PushPayload{
		Ref: "refs/heads/main",
		HeadCommit: &api.PayloadCommit{
			ID:      "58771003157b81abc6bf41df0c5db4147a3e3c83",
			Message: "json test",
		},
		Repo: repo,
	}
	data, err := p.JSONPayload()
	require.NoError(t, err)

	hook := &webhook_model.Webhook{
		RepoID:   3,
		IsActive: true,
		Type:     webhook_module.MATRIX,
		URL:      "https://sourcehut.example.com/api/jobs",
		Meta:     `{"manifest_path":"simple.yml"}`,
	}
	task := &webhook_model.HookTask{
		HookID:         hook.ID,
		EventType:      webhook_module.HookEventPush,
		PayloadContent: string(data),
		PayloadVersion: 2,
	}

	req, reqBody, err := BuildsHandler{}.NewRequest(context.Background(), hook, task)
	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, reqBody)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/api/jobs", req.URL.Path)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	var body graphqlPayload[buildsVariables]
	err = json.NewDecoder(req.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "json test", body.Variables.Note)
}

func CreateDeclarativeRepo(t *testing.T, owner *user_model.User, name string, enabledUnits, disabledUnits []unit_model.Type, files []*files_service.ChangeRepoFile) (*repo_model.Repository, string) {
	t.Helper()

	// Create a new repository
	repo, err := repo_service.CreateRepository(db.DefaultContext, owner, owner, repo_service.CreateRepoOptions{
		Name:          name,
		Description:   "Temporary Repo",
		AutoInit:      true,
		Gitignores:    "",
		License:       "WTFPL",
		Readme:        "Default",
		DefaultBranch: "main",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, repo)
	t.Cleanup(func() {
		repo_service.DeleteRepository(db.DefaultContext, owner, repo, false)
	})

	if enabledUnits != nil || disabledUnits != nil {
		units := make([]repo_model.RepoUnit, len(enabledUnits))
		for i, unitType := range enabledUnits {
			units[i] = repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unitType,
			}
		}

		err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, units, disabledUnits)
		require.NoError(t, err)
	}

	var sha string
	if len(files) > 0 {
		resp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, owner, &files_service.ChangeRepoFilesOptions{
			Files:     files,
			Message:   "add files",
			OldBranch: "main",
			NewBranch: "main",
			Author: &files_service.IdentityOptions{
				Name:  owner.Name,
				Email: owner.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  owner.Name,
				Email: owner.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp)

		sha = resp.Commit.SHA
	}

	return repo, sha
}
