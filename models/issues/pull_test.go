// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"fmt"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequest_LoadAttributes(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pr.LoadAttributes(db.DefaultContext))
	assert.NotNil(t, pr.Merger)
	assert.Equal(t, pr.MergerID, pr.Merger.ID)
}

func TestPullRequest_LoadIssue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pr.LoadIssue(db.DefaultContext))
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
	require.NoError(t, pr.LoadIssue(db.DefaultContext))
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
}

func TestPullRequest_LoadBaseRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pr.LoadBaseRepo(db.DefaultContext))
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
	require.NoError(t, pr.LoadBaseRepo(db.DefaultContext))
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
}

func TestPullRequest_LoadHeadRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pr.LoadHeadRepo(db.DefaultContext))
	assert.NotNil(t, pr.HeadRepo)
	assert.Equal(t, pr.HeadRepoID, pr.HeadRepo.ID)
}

// TODO TestMerge

// TODO TestNewPullRequest

func TestPullRequestsNewest(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	prs, count, err := issues_model.PullRequests(db.DefaultContext, 1, &issues_model.PullRequestsOptions{
		ListOptions: db.ListOptions{
			Page: 1,
		},
		State:    "open",
		SortType: "newest",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)
	if assert.Len(t, prs, 3) {
		assert.EqualValues(t, 5, prs[0].ID)
		assert.EqualValues(t, 2, prs[1].ID)
		assert.EqualValues(t, 1, prs[2].ID)
	}
}

func TestPullRequests_Closed_RecentSortType(t *testing.T) {
	// Issue ID | Closed At.  | Updated At
	//    2     | 1707270001  | 1707270001
	//    3     | 1707271000  | 1707279999
	//    11    | 1707279999  | 1707275555
	tests := []struct {
		sortType             string
		expectedIssueIDOrder []int64
	}{
		{"recentupdate", []int64{3, 11, 2}},
		{"recentclose", []int64{11, 3, 2}},
	}

	assert.NoError(t, unittest.PrepareTestDatabase())
	_, err := db.Exec(db.DefaultContext, "UPDATE issue SET closed_unix = 1707270001, updated_unix = 1707270001, is_closed = true WHERE id = 2")
	require.NoError(t, err)
	_, err = db.Exec(db.DefaultContext, "UPDATE issue SET closed_unix = 1707271000, updated_unix = 1707279999, is_closed = true WHERE id = 3")
	require.NoError(t, err)
	_, err = db.Exec(db.DefaultContext, "UPDATE issue SET closed_unix = 1707279999, updated_unix = 1707275555, is_closed = true WHERE id = 11")
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.sortType, func(t *testing.T) {
			prs, _, err := issues_model.PullRequests(db.DefaultContext, 1, &issues_model.PullRequestsOptions{
				ListOptions: db.ListOptions{
					Page: 1,
				},
				State:    "closed",
				SortType: test.sortType,
			})
			require.NoError(t, err)

			if assert.Len(t, prs, len(test.expectedIssueIDOrder)) {
				for i := range test.expectedIssueIDOrder {
					assert.Equal(t, test.expectedIssueIDOrder[i], prs[i].IssueID)
				}
			}
		})
	}
}

func TestLoadRequestedReviewers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pull.LoadIssue(db.DefaultContext))
	issue := pull.Issue
	require.NoError(t, issue.LoadRepo(db.DefaultContext))
	assert.Empty(t, pull.RequestedReviewers)

	user1, err := user_model.GetUserByID(db.DefaultContext, 1)
	require.NoError(t, err)

	comment, err := issues_model.AddReviewRequest(db.DefaultContext, issue, user1, &user_model.User{})
	require.NoError(t, err)
	assert.NotNil(t, comment)

	require.NoError(t, pull.LoadRequestedReviewers(db.DefaultContext))
	assert.Len(t, pull.RequestedReviewers, 1)

	comment, err = issues_model.RemoveReviewRequest(db.DefaultContext, issue, user1, &user_model.User{})
	require.NoError(t, err)
	assert.NotNil(t, comment)

	pull.RequestedReviewers = nil
	require.NoError(t, pull.LoadRequestedReviewers(db.DefaultContext))
	assert.Empty(t, pull.RequestedReviewers)
}

func TestPullRequestsOldest(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	prs, count, err := issues_model.PullRequests(db.DefaultContext, 1, &issues_model.PullRequestsOptions{
		ListOptions: db.ListOptions{
			Page: 1,
		},
		State:    "open",
		SortType: "oldest",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)
	if assert.Len(t, prs, 3) {
		assert.EqualValues(t, 1, prs[0].ID)
		assert.EqualValues(t, 2, prs[1].ID)
		assert.EqualValues(t, 5, prs[2].ID)
	}
}

func TestGetUnmergedPullRequest(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetUnmergedPullRequest(db.DefaultContext, 1, 1, "branch2", "master", issues_model.PullRequestFlowGithub)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pr.ID)

	_, err = issues_model.GetUnmergedPullRequest(db.DefaultContext, 1, 9223372036854775807, "branch1", "master", issues_model.PullRequestFlowGithub)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestHasUnmergedPullRequestsByHeadInfo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	exist, err := issues_model.HasUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "branch2")
	require.NoError(t, err)
	assert.True(t, exist)

	exist, err = issues_model.HasUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "not_exist_branch")
	require.NoError(t, err)
	assert.False(t, exist)
}

func TestGetUnmergedPullRequestsByHeadInfo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	prs, err := issues_model.GetUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "branch2")
	require.NoError(t, err)
	assert.Len(t, prs, 1)
	for _, pr := range prs {
		assert.Equal(t, int64(1), pr.HeadRepoID)
		assert.Equal(t, "branch2", pr.HeadBranch)
	}
}

func TestGetUnmergedPullRequestsByHeadInfoMax(t *testing.T) {
	defer unittest.OverrideFixtures("models/fixtures/TestGetUnmergedPullRequestsByHeadInfoMax")()
	require.NoError(t, unittest.PrepareTestDatabase())

	repoID := int64(1)
	olderThan := int64(0)

	// for NULL created field the olderThan condition is ignored
	prs, err := issues_model.GetUnmergedPullRequestsByHeadInfoMax(db.DefaultContext, repoID, olderThan, "branch2")
	require.NoError(t, err)
	assert.Equal(t, int64(1), prs[0].HeadRepoID)

	// test for when the created field is set
	branch := "branchmax"
	prs, err = issues_model.GetUnmergedPullRequestsByHeadInfoMax(db.DefaultContext, repoID, olderThan, branch)
	require.NoError(t, err)
	assert.Empty(t, prs)
	olderThan = time.Now().UnixNano()
	require.NoError(t, err)
	prs, err = issues_model.GetUnmergedPullRequestsByHeadInfoMax(db.DefaultContext, repoID, olderThan, branch)
	require.NoError(t, err)
	assert.Len(t, prs, 1)
	for _, pr := range prs {
		assert.Equal(t, int64(1), pr.HeadRepoID)
		assert.Equal(t, branch, pr.HeadBranch)
	}
	pr := prs[0]

	for _, testCase := range []struct {
		table   string
		field   string
		id      int64
		match   any
		nomatch any
	}{
		{
			table:   "issue",
			field:   "is_closed",
			id:      pr.IssueID,
			match:   false,
			nomatch: true,
		},
		{
			table:   "pull_request",
			field:   "flow",
			id:      pr.ID,
			match:   issues_model.PullRequestFlowGithub,
			nomatch: issues_model.PullRequestFlowAGit,
		},
		{
			table:   "pull_request",
			field:   "head_repo_id",
			id:      pr.ID,
			match:   pr.HeadRepoID,
			nomatch: 0,
		},
		{
			table:   "pull_request",
			field:   "head_branch",
			id:      pr.ID,
			match:   pr.HeadBranch,
			nomatch: "something else",
		},
		{
			table:   "pull_request",
			field:   "has_merged",
			id:      pr.ID,
			match:   false,
			nomatch: true,
		},
	} {
		t.Run(testCase.field, func(t *testing.T) {
			update := fmt.Sprintf("UPDATE `%s` SET `%s` = ? WHERE `id` = ?", testCase.table, testCase.field)

			// expect no match
			_, err = db.GetEngine(db.DefaultContext).Exec(update, testCase.nomatch, testCase.id)
			require.NoError(t, err)
			prs, err = issues_model.GetUnmergedPullRequestsByHeadInfoMax(db.DefaultContext, repoID, olderThan, branch)
			require.NoError(t, err)
			assert.Empty(t, prs)

			// expect one match
			_, err = db.GetEngine(db.DefaultContext).Exec(update, testCase.match, testCase.id)
			require.NoError(t, err)
			prs, err = issues_model.GetUnmergedPullRequestsByHeadInfoMax(db.DefaultContext, repoID, olderThan, branch)
			require.NoError(t, err)
			assert.Len(t, prs, 1)

			// identical to the known PR
			assert.Equal(t, pr.ID, prs[0].ID)
		})
	}
}

func TestGetUnmergedPullRequestsByBaseInfo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	prs, err := issues_model.GetUnmergedPullRequestsByBaseInfo(db.DefaultContext, 1, "master")
	require.NoError(t, err)
	assert.Len(t, prs, 1)
	pr := prs[0]
	assert.Equal(t, int64(2), pr.ID)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, "master", pr.BaseBranch)
}

func TestGetPullRequestByIndex(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByIndex(db.DefaultContext, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, int64(2), pr.Index)

	_, err = issues_model.GetPullRequestByIndex(db.DefaultContext, 9223372036854775807, 9223372036854775807)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))

	_, err = issues_model.GetPullRequestByIndex(db.DefaultContext, 1, 0)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pr.ID)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = issues_model.GetPullRequestByID(db.DefaultContext, 9223372036854775807)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByIssueID(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = issues_model.GetPullRequestByIssueID(db.DefaultContext, 9223372036854775807)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestPullRequest_Update(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	pr.BaseBranch = "baseBranch"
	pr.HeadBranch = "headBranch"
	pr.Update(db.DefaultContext)

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
	assert.Equal(t, "baseBranch", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
	unittest.CheckConsistencyFor(t, pr)
}

func TestPullRequest_UpdateCols(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := &issues_model.PullRequest{
		ID:         1,
		BaseBranch: "baseBranch",
		HeadBranch: "headBranch",
	}
	require.NoError(t, pr.UpdateCols(db.DefaultContext, "head_branch"))

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	assert.Equal(t, "master", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
	unittest.CheckConsistencyFor(t, pr)
}

func TestPullRequestList_LoadAttributes(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	prs := []*issues_model.PullRequest{
		unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}),
		unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}),
	}
	require.NoError(t, issues_model.PullRequestList(prs).LoadAttributes(db.DefaultContext))
	for _, pr := range prs {
		assert.NotNil(t, pr.Issue)
		assert.Equal(t, pr.IssueID, pr.Issue.ID)
	}

	require.NoError(t, issues_model.PullRequestList([]*issues_model.PullRequest{}).LoadAttributes(db.DefaultContext))
}

// TODO TestAddTestPullRequestTask

func TestPullRequest_IsWorkInProgress(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	pr.LoadIssue(db.DefaultContext)

	assert.False(t, pr.IsWorkInProgress(db.DefaultContext))

	pr.Issue.Title = "WIP: " + pr.Issue.Title
	assert.True(t, pr.IsWorkInProgress(db.DefaultContext))

	pr.Issue.Title = "[wip]: " + pr.Issue.Title
	assert.True(t, pr.IsWorkInProgress(db.DefaultContext))
}

func TestPullRequest_GetWorkInProgressPrefixWorkInProgress(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	pr.LoadIssue(db.DefaultContext)

	assert.Empty(t, pr.GetWorkInProgressPrefix(db.DefaultContext))

	original := pr.Issue.Title
	pr.Issue.Title = "WIP: " + original
	assert.Equal(t, "WIP:", pr.GetWorkInProgressPrefix(db.DefaultContext))

	pr.Issue.Title = "[wip] " + original
	assert.Equal(t, "[wip]", pr.GetWorkInProgressPrefix(db.DefaultContext))
}

func TestDeleteOrphanedObjects(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	countBefore, err := db.GetEngine(db.DefaultContext).Count(&issues_model.PullRequest{})
	require.NoError(t, err)

	_, err = db.GetEngine(db.DefaultContext).Insert(&issues_model.PullRequest{IssueID: 1000}, &issues_model.PullRequest{IssueID: 1001}, &issues_model.PullRequest{IssueID: 1003})
	require.NoError(t, err)

	orphaned, err := db.CountOrphanedObjects(db.DefaultContext, "pull_request", "issue", "pull_request.issue_id=issue.id")
	require.NoError(t, err)
	assert.EqualValues(t, 3, orphaned)

	err = db.DeleteOrphanedObjects(db.DefaultContext, "pull_request", "issue", "pull_request.issue_id=issue.id")
	require.NoError(t, err)

	countAfter, err := db.GetEngine(db.DefaultContext).Count(&issues_model.PullRequest{})
	require.NoError(t, err)
	assert.Equal(t, countBefore, countAfter)
}

func TestParseCodeOwnersLine(t *testing.T) {
	type CodeOwnerTest struct {
		Line   string
		Tokens []string
	}

	given := []CodeOwnerTest{
		{Line: "", Tokens: nil},
		{Line: "# comment", Tokens: []string{}},
		{Line: "!.* @user1 @org1/team1", Tokens: []string{"!.*", "@user1", "@org1/team1"}},
		{Line: `.*\\.js @user2 #comment`, Tokens: []string{`.*\.js`, "@user2"}},
		{Line: `docs/(aws|google|azure)/[^/]*\\.(md|txt) @org3 @org2/team2`, Tokens: []string{`docs/(aws|google|azure)/[^/]*\.(md|txt)`, "@org3", "@org2/team2"}},
		{Line: `\#path @org3`, Tokens: []string{`#path`, "@org3"}},
		{Line: `path\ with\ spaces/ @org3`, Tokens: []string{`path with spaces/`, "@org3"}},
	}

	for _, g := range given {
		tokens := issues_model.TokenizeCodeOwnersLine(g.Line)
		assert.Equal(t, g.Tokens, tokens, "Codeowners tokenizer failed")
	}
}

func TestGetApprovers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5})
	// Official reviews are already deduplicated. Allow unofficial reviews
	// to assert that there are no duplicated approvers.
	setting.Repository.PullRequest.DefaultMergeMessageOfficialApproversOnly = false
	approvers := pr.GetApprovers(db.DefaultContext)
	expected := "Reviewed-by: User Five <user5@example.com>\nReviewed-by: Org Six <org6@example.com>\n"
	assert.Equal(t, expected, approvers)
}

func TestGetPullRequestByMergedCommit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByMergedCommit(db.DefaultContext, 1, "1a8823cd1a9549fde083f992f6b9b87a7ab74fb3")
	require.NoError(t, err)
	assert.EqualValues(t, 1, pr.ID)

	_, err = issues_model.GetPullRequestByMergedCommit(db.DefaultContext, 0, "1a8823cd1a9549fde083f992f6b9b87a7ab74fb3")
	require.ErrorAs(t, err, &issues_model.ErrPullRequestNotExist{})
	_, err = issues_model.GetPullRequestByMergedCommit(db.DefaultContext, 1, "")
	require.ErrorAs(t, err, &issues_model.ErrPullRequestNotExist{})
}

func TestMigrate_InsertPullRequests(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	reponame := "repo1"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{Name: reponame})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	i := &issues_model.Issue{
		RepoID:   repo.ID,
		Repo:     repo,
		Title:    "title1",
		Content:  "issuecontent1",
		IsPull:   true,
		PosterID: owner.ID,
		Poster:   owner,
	}

	p := &issues_model.PullRequest{
		Issue: i,
	}

	err := issues_model.InsertPullRequests(db.DefaultContext, p)
	require.NoError(t, err)

	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{IssueID: i.ID})

	unittest.CheckConsistencyFor(t, &issues_model.Issue{}, &issues_model.PullRequest{})
}
