// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/migration"
	mirror_service "forgejo.org/services/mirror"
	release_service "forgejo.org/services/release"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMirrorPull(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repoPath := repo_model.RepoPath(user.Name, repo.Name)

	opts := migration.MigrateOptions{
		RepoName:    "test_mirror",
		Description: "Test mirror",
		Private:     false,
		Mirror:      true,
		CloneAddr:   repoPath,
		Wiki:        true,
		Releases:    false,
	}

	mirrorRepo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name:        opts.RepoName,
		Description: opts.Description,
		IsPrivate:   opts.Private,
		IsMirror:    opts.Mirror,
		Status:      repo_model.RepositoryBeingMigrated,
	})
	require.NoError(t, err)
	assert.True(t, mirrorRepo.IsMirror, "expected pull-mirror repo to be marked as a mirror immediately after its creation")

	ctx := t.Context()

	mirror, err := repo_service.MigrateRepositoryGitData(ctx, user, mirrorRepo, opts, nil)
	require.NoError(t, err)

	gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo)
	require.NoError(t, err)
	defer gitRepo.Close()

	findOptions := repo_model.FindReleasesOptions{
		IncludeDrafts: true,
		IncludeTags:   true,
		RepoID:        mirror.ID,
	}
	initCount, err := db.Count[repo_model.Release](db.DefaultContext, findOptions)
	require.NoError(t, err)

	require.NoError(t, release_service.CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.2",
		Target:       "master",
		Title:        "v0.2 is released",
		Note:         "v0.2 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        true,
	}, "", []*release_service.AttachmentChange{}))

	_, err = repo_model.GetMirrorByRepoID(ctx, mirror.ID)
	require.NoError(t, err)

	ok := mirror_service.SyncPullMirror(ctx, mirror.ID)
	assert.True(t, ok)

	count, err := db.Count[repo_model.Release](db.DefaultContext, findOptions)
	require.NoError(t, err)
	assert.Equal(t, initCount+1, count)

	release, err := repo_model.GetRelease(db.DefaultContext, repo.ID, "v0.2")
	require.NoError(t, err)
	require.NoError(t, release_service.DeleteReleaseByID(ctx, repo, release, user, true))

	ok = mirror_service.SyncPullMirror(ctx, mirror.ID)
	assert.True(t, ok)

	count, err = db.Count[repo_model.Release](db.DefaultContext, findOptions)
	require.NoError(t, err)
	assert.Equal(t, initCount, count)
}
