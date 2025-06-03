// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	pull_service "forgejo.org/services/pull"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestListPullCommits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user5")
	req := NewRequest(t, "GET", "/user2/repo1/pulls/3/commits/list")
	resp := session.MakeRequest(t, req, http.StatusOK)

	var pullCommitList struct {
		Commits             []pull_service.CommitInfo `json:"commits"`
		LastReviewCommitSha string                    `json:"last_review_commit_sha"`
	}
	DecodeJSON(t, resp, &pullCommitList)

	if assert.Len(t, pullCommitList.Commits, 2) {
		assert.Equal(t, "5f22f7d0d95d614d25a5b68592adb345a4b5c7fd", pullCommitList.Commits[0].ID)
		assert.Equal(t, "4a357436d925b5c974181ff12a994538ddc5a269", pullCommitList.Commits[1].ID)
	}
	assert.Equal(t, "4a357436d925b5c974181ff12a994538ddc5a269", pullCommitList.LastReviewCommitSha)
}

func TestPullCommitLinks(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/pulls/3/commits")
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)

	commitSha := htmlDoc.Find(".commit-list td.sha a.sha.label").First()
	commitShaHref, _ := commitSha.Attr("href")
	assert.Equal(t, "/user2/repo1/pulls/3/commits/5f22f7d0d95d614d25a5b68592adb345a4b5c7fd", commitShaHref)

	commitLink := htmlDoc.Find(".commit-list td.message a").First()
	commitLinkHref, _ := commitLink.Attr("href")
	assert.Equal(t, "/user2/repo1/pulls/3/commits/5f22f7d0d95d614d25a5b68592adb345a4b5c7fd", commitLinkHref)
}
