// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"slices"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/test"

	"gitea.com/go-chi/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_ContributorsGraph(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	require.NoError(t, repo.LoadOwner(db.DefaultContext))
	mockCache, err := cache.NewCacher(cache.Options{
		Adapter:  "memory",
		Interval: 24 * 60,
	})
	require.NoError(t, err)

	lc, cleanup := test.NewLogChecker(log.DEFAULT, log.INFO)
	lc.StopMark(`getExtendedCommitStats[repo="user2/repo2" revision="404ref"]: object does not exist [id: 404ref, rel_path: ]`)
	defer cleanup()

	generateContributorStats(nil, mockCache, "key", repo, "404ref")
	assert.False(t, mockCache.IsExist("key"))
	_, stopped := lc.Check(100 * time.Millisecond)
	assert.True(t, stopped)

	generateContributorStats(nil, mockCache, "key2", repo, "master")
	dataString, isData := mockCache.Get("key2").(string)
	assert.True(t, isData)
	// Verify that JSON is actually stored in the cache.
	assert.EqualValues(t, `{"ethantkoenig@gmail.com":{"name":"Ethan Koenig","login":"","avatar_link":"https://secure.gravatar.com/avatar/b42fb195faa8c61b8d88abfefe30e9e3?d=identicon","home_link":"","total_commits":1,"weeks":{"1511654400000":{"week":1511654400000,"additions":3,"deletions":0,"commits":1}}},"jimmy.praet@telenet.be":{"name":"Jimmy Praet","login":"","avatar_link":"https://secure.gravatar.com/avatar/93c49b7c89eb156971d11161c9b52795?d=identicon","home_link":"","total_commits":1,"weeks":{"1624752000000":{"week":1624752000000,"additions":2,"deletions":0,"commits":1}}},"jon@allspice.io":{"name":"Jon","login":"","avatar_link":"https://secure.gravatar.com/avatar/00388ce725e6886f3e07c3733007289b?d=identicon","home_link":"","total_commits":1,"weeks":{"1607817600000":{"week":1607817600000,"additions":10,"deletions":0,"commits":1}}},"total":{"name":"Total","login":"","avatar_link":"","home_link":"","total_commits":3,"weeks":{"1511654400000":{"week":1511654400000,"additions":3,"deletions":0,"commits":1},"1607817600000":{"week":1607817600000,"additions":10,"deletions":0,"commits":1},"1624752000000":{"week":1624752000000,"additions":2,"deletions":0,"commits":1}}}}`, dataString)

	var data map[string]*ContributorData
	require.NoError(t, json.Unmarshal([]byte(dataString), &data))

	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	assert.EqualValues(t, []string{
		"ethantkoenig@gmail.com",
		"jimmy.praet@telenet.be",
		"jon@allspice.io",
		"total", // generated summary
	}, keys)

	assert.EqualValues(t, &ContributorData{
		Name:         "Ethan Koenig",
		AvatarLink:   "https://secure.gravatar.com/avatar/b42fb195faa8c61b8d88abfefe30e9e3?d=identicon",
		TotalCommits: 1,
		Weeks: map[int64]*WeekData{
			1511654400000: {
				Week:      1511654400000, // sunday 2017-11-26
				Additions: 3,
				Deletions: 0,
				Commits:   1,
			},
		},
	}, data["ethantkoenig@gmail.com"])
	assert.EqualValues(t, &ContributorData{
		Name:         "Total",
		AvatarLink:   "",
		TotalCommits: 3,
		Weeks: map[int64]*WeekData{
			1511654400000: {
				Week:      1511654400000, // sunday 2017-11-26 (2017-11-26 20:31:18 -0800)
				Additions: 3,
				Deletions: 0,
				Commits:   1,
			},
			1607817600000: {
				Week:      1607817600000, // sunday 2020-12-13 (2020-12-15 15:23:11 -0500)
				Additions: 10,
				Deletions: 0,
				Commits:   1,
			},
			1624752000000: {
				Week:      1624752000000, // sunday 2021-06-27 (2021-06-29 21:54:09 +0200)
				Additions: 2,
				Deletions: 0,
				Commits:   1,
			},
		},
	}, data["total"])
}
