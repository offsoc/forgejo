// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package release

import (
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/services/attachment"

	_ "code.gitea.io/gitea/models/actions"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestRelease_Create(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo)
	require.NoError(t, err)
	defer gitRepo.Close()

	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.1",
		Target:       "master",
		Title:        "v0.1 is released",
		Note:         "v0.1 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        false,
	}, nil, ""))

	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.1.1",
		Target:       "65f1bf27bc3bf70f64657658635e66094edbcb4d",
		Title:        "v0.1.1 is released",
		Note:         "v0.1.1 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        false,
	}, nil, ""))

	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.1.2",
		Target:       "65f1bf2",
		Title:        "v0.1.2 is released",
		Note:         "v0.1.2 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        false,
	}, nil, ""))

	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.1.3",
		Target:       "65f1bf2",
		Title:        "v0.1.3 is released",
		Note:         "v0.1.3 is released",
		IsDraft:      true,
		IsPrerelease: false,
		IsTag:        false,
	}, nil, ""))

	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.1.4",
		Target:       "65f1bf2",
		Title:        "v0.1.4 is released",
		Note:         "v0.1.4 is released",
		IsDraft:      false,
		IsPrerelease: true,
		IsTag:        false,
	}, nil, ""))

	testPlayload := "testtest"

	attach, err := attachment.NewAttachment(db.DefaultContext, &repo_model.Attachment{
		RepoID:     repo.ID,
		UploaderID: user.ID,
		Name:       "test.txt",
	}, strings.NewReader(testPlayload), int64(len([]byte(testPlayload))))
	require.NoError(t, err)

	release := repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v0.1.5",
		Target:       "65f1bf2",
		Title:        "v0.1.5 is released",
		Note:         "v0.1.5 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        true,
	}
	require.NoError(t, CreateRelease(gitRepo, &release, []string{attach.UUID}, "test"))
}

func TestRelease_Update(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo)
	require.NoError(t, err)
	defer gitRepo.Close()

	// Test a changed release
	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v1.1.1",
		Target:       "master",
		Title:        "v1.1.1 is released",
		Note:         "v1.1.1 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        false,
	}, nil, ""))
	release, err := repo_model.GetRelease(db.DefaultContext, repo.ID, "v1.1.1")
	require.NoError(t, err)
	releaseCreatedUnix := release.CreatedUnix
	time.Sleep(2 * time.Second) // sleep 2 seconds to ensure a different timestamp
	release.Note = "Changed note"
	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, nil, nil, nil, false))
	release, err = repo_model.GetReleaseByID(db.DefaultContext, release.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(releaseCreatedUnix), int64(release.CreatedUnix))

	// Test a changed draft
	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v1.2.1",
		Target:       "65f1bf2",
		Title:        "v1.2.1 is draft",
		Note:         "v1.2.1 is draft",
		IsDraft:      true,
		IsPrerelease: false,
		IsTag:        false,
	}, nil, ""))
	release, err = repo_model.GetRelease(db.DefaultContext, repo.ID, "v1.2.1")
	require.NoError(t, err)
	releaseCreatedUnix = release.CreatedUnix
	time.Sleep(2 * time.Second) // sleep 2 seconds to ensure a different timestamp
	release.Title = "Changed title"
	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, nil, nil, nil, false))
	release, err = repo_model.GetReleaseByID(db.DefaultContext, release.ID)
	require.NoError(t, err)
	assert.Less(t, int64(releaseCreatedUnix), int64(release.CreatedUnix))

	// Test a changed pre-release
	require.NoError(t, CreateRelease(gitRepo, &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v1.3.1",
		Target:       "65f1bf2",
		Title:        "v1.3.1 is pre-released",
		Note:         "v1.3.1 is pre-released",
		IsDraft:      false,
		IsPrerelease: true,
		IsTag:        false,
	}, nil, ""))
	release, err = repo_model.GetRelease(db.DefaultContext, repo.ID, "v1.3.1")
	require.NoError(t, err)
	releaseCreatedUnix = release.CreatedUnix
	time.Sleep(2 * time.Second) // sleep 2 seconds to ensure a different timestamp
	release.Title = "Changed title"
	release.Note = "Changed note"
	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, nil, nil, nil, false))
	release, err = repo_model.GetReleaseByID(db.DefaultContext, release.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(releaseCreatedUnix), int64(release.CreatedUnix))

	// Test create release
	release = &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v1.1.2",
		Target:       "master",
		Title:        "v1.1.2 is released",
		Note:         "v1.1.2 is released",
		IsDraft:      true,
		IsPrerelease: false,
		IsTag:        false,
	}
	require.NoError(t, CreateRelease(gitRepo, release, nil, ""))
	assert.Positive(t, release.ID)

	release.IsDraft = false
	tagName := release.TagName

	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, nil, nil, nil, false))
	release, err = repo_model.GetReleaseByID(db.DefaultContext, release.ID)
	require.NoError(t, err)
	assert.Equal(t, tagName, release.TagName)

	// Add new attachments
	samplePayload := "testtest"
	attach, err := attachment.NewAttachment(db.DefaultContext, &repo_model.Attachment{
		RepoID:     repo.ID,
		UploaderID: user.ID,
		Name:       "test.txt",
	}, strings.NewReader(samplePayload), int64(len([]byte(samplePayload))))
	require.NoError(t, err)

	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, []string{attach.UUID}, nil, nil, false))
	require.NoError(t, repo_model.GetReleaseAttachments(db.DefaultContext, release))
	assert.Len(t, release.Attachments, 1)
	assert.EqualValues(t, attach.UUID, release.Attachments[0].UUID)
	assert.EqualValues(t, release.ID, release.Attachments[0].ReleaseID)
	assert.EqualValues(t, attach.Name, release.Attachments[0].Name)

	// update the attachment name
	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, nil, nil, map[string]string{
		attach.UUID: "test2.txt",
	}, false))
	release.Attachments = nil
	require.NoError(t, repo_model.GetReleaseAttachments(db.DefaultContext, release))
	assert.Len(t, release.Attachments, 1)
	assert.EqualValues(t, attach.UUID, release.Attachments[0].UUID)
	assert.EqualValues(t, release.ID, release.Attachments[0].ReleaseID)
	assert.EqualValues(t, "test2.txt", release.Attachments[0].Name)

	// delete the attachment
	require.NoError(t, UpdateRelease(db.DefaultContext, user, gitRepo, release, nil, []string{attach.UUID}, nil, false))
	release.Attachments = nil
	require.NoError(t, repo_model.GetReleaseAttachments(db.DefaultContext, release))
	assert.Empty(t, release.Attachments)
}

func TestRelease_createTag(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo)
	require.NoError(t, err)
	defer gitRepo.Close()

	// Test a changed release
	release := &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v2.1.1",
		Target:       "master",
		Title:        "v2.1.1 is released",
		Note:         "v2.1.1 is released",
		IsDraft:      false,
		IsPrerelease: false,
		IsTag:        false,
	}
	_, err = createTag(db.DefaultContext, gitRepo, release, "")
	require.NoError(t, err)
	assert.NotEmpty(t, release.CreatedUnix)
	releaseCreatedUnix := release.CreatedUnix
	time.Sleep(2 * time.Second) // sleep 2 seconds to ensure a different timestamp
	release.Note = "Changed note"
	_, err = createTag(db.DefaultContext, gitRepo, release, "")
	require.NoError(t, err)
	assert.Equal(t, int64(releaseCreatedUnix), int64(release.CreatedUnix))

	// Test a changed draft
	release = &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v2.2.1",
		Target:       "65f1bf2",
		Title:        "v2.2.1 is draft",
		Note:         "v2.2.1 is draft",
		IsDraft:      true,
		IsPrerelease: false,
		IsTag:        false,
	}
	_, err = createTag(db.DefaultContext, gitRepo, release, "")
	require.NoError(t, err)
	releaseCreatedUnix = release.CreatedUnix
	time.Sleep(2 * time.Second) // sleep 2 seconds to ensure a different timestamp
	release.Title = "Changed title"
	_, err = createTag(db.DefaultContext, gitRepo, release, "")
	require.NoError(t, err)
	assert.Less(t, int64(releaseCreatedUnix), int64(release.CreatedUnix))

	// Test a changed pre-release
	release = &repo_model.Release{
		RepoID:       repo.ID,
		Repo:         repo,
		PublisherID:  user.ID,
		Publisher:    user,
		TagName:      "v2.3.1",
		Target:       "65f1bf2",
		Title:        "v2.3.1 is pre-released",
		Note:         "v2.3.1 is pre-released",
		IsDraft:      false,
		IsPrerelease: true,
		IsTag:        false,
	}
	_, err = createTag(db.DefaultContext, gitRepo, release, "")
	require.NoError(t, err)
	releaseCreatedUnix = release.CreatedUnix
	time.Sleep(2 * time.Second) // sleep 2 seconds to ensure a different timestamp
	release.Title = "Changed title"
	release.Note = "Changed note"
	_, err = createTag(db.DefaultContext, gitRepo, release, "")
	require.NoError(t, err)
	assert.Equal(t, int64(releaseCreatedUnix), int64(release.CreatedUnix))
}

func TestCreateNewTag(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	require.NoError(t, CreateNewTag(git.DefaultContext, user, repo, "master", "v2.0",
		"v2.0 is released \n\n BUGFIX: .... \n\n 123"))
}
