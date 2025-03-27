// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"os"
	"path"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckUnadoptedRepositories_Add(t *testing.T) {
	start := 10
	end := 20
	unadopted := &unadoptedRepositories{
		start: start,
		end:   end,
		index: 0,
	}

	total := 30
	for i := 0; i < total; i++ {
		unadopted.add("something")
	}

	assert.Equal(t, total, unadopted.index)
	assert.Len(t, unadopted.repositories, end-start)
}

func TestCheckUnadoptedRepositories(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	//
	// Non existent user
	//
	unadopted := &unadoptedRepositories{start: 0, end: 100}
	err := checkUnadoptedRepositories(db.DefaultContext, "notauser", []string{"repo"}, unadopted)
	require.NoError(t, err)
	assert.Empty(t, unadopted.repositories)
	//
	// Unadopted repository is returned
	// Existing (adopted) repository is not returned
	//
	userName := "user2"
	repoName := "repo2"
	unadoptedRepoName := "unadopted"
	unadopted = &unadoptedRepositories{start: 0, end: 100}
	err = checkUnadoptedRepositories(db.DefaultContext, userName, []string{repoName, unadoptedRepoName}, unadopted)
	require.NoError(t, err)
	assert.Equal(t, []string{path.Join(userName, unadoptedRepoName)}, unadopted.repositories)
	//
	// Existing (adopted) repository is not returned
	//
	unadopted = &unadoptedRepositories{start: 0, end: 100}
	err = checkUnadoptedRepositories(db.DefaultContext, userName, []string{repoName}, unadopted)
	require.NoError(t, err)
	assert.Empty(t, unadopted.repositories)
	assert.Equal(t, 0, unadopted.index)
}

func TestListUnadoptedRepositories_ListOptions(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	username := "user2"
	unadoptedList := []string{path.Join(username, "unadopted1"), path.Join(username, "unadopted2")}
	for _, unadopted := range unadoptedList {
		_ = os.Mkdir(path.Join(setting.RepoRootPath, unadopted+".git"), 0o755)
	}

	opts := db.ListOptions{Page: 1, PageSize: 1}
	repoNames, count, err := ListUnadoptedRepositories(db.DefaultContext, "", &opts)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, unadoptedList[0], repoNames[0])

	opts = db.ListOptions{Page: 2, PageSize: 1}
	repoNames, count, err = ListUnadoptedRepositories(db.DefaultContext, "", &opts)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, unadoptedList[1], repoNames[0])
}

func TestAdoptRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	username := "user2"

	unadopted := "unadopted"
	require.NoError(t, unittest.CopyDir(
		"../../modules/git/tests/repos/repo1_bare",
		path.Join(setting.RepoRootPath, username, unadopted+".git"),
	))

	opts := db.ListOptions{Page: 1, PageSize: 1}
	repoNames, _, err := ListUnadoptedRepositories(db.DefaultContext, "", &opts)
	require.NoError(t, err)
	require.Contains(t, repoNames, path.Join(username, unadopted))

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo, err := AdoptRepository(db.DefaultContext, doer, owner, CreateRepoOptions{
		Name:        unadopted,
		Description: "description",
		IsPrivate:   false,
		AutoInit:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, git.Sha1ObjectFormat.Name(), repo.ObjectFormatName)
}
