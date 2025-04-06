// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/templates"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchivedIssues(t *testing.T) {
	// Arrange
	setting.UI.IssuePagingNum = 1
	require.NoError(t, unittest.LoadFixtures())

	ctx, _ := contexttest.MockContext(t, "issues")
	contexttest.LoadUser(t, ctx, 30)
	ctx.Req.Form.Set("state", "open")
	ctx.Req.Form.Set("type", "your_repositories")

	// Assume: User 30 has access to two Repos with Issues, one of the Repos being archived.
	repos, _, _ := repo_model.GetUserRepositories(db.DefaultContext, &repo_model.SearchRepoOptions{Actor: ctx.Doer})
	assert.Len(t, repos, 3)
	IsArchived := make(map[int64]bool)
	NumIssues := make(map[int64]int)
	for _, repo := range repos {
		IsArchived[repo.ID] = repo.IsArchived
		NumIssues[repo.ID] = repo.NumIssues
	}
	assert.False(t, IsArchived[50])
	assert.Equal(t, 1, NumIssues[50])
	assert.True(t, IsArchived[51])
	assert.Equal(t, 1, NumIssues[51])

	// Act
	Issues(ctx)

	// Assert: One Issue (ID 30) from one Repo (ID 50) is retrieved, while nothing from archived Repo 51 is retrieved
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())

	assert.Len(t, ctx.Data["Issues"], 1)
}

func TestIssues(t *testing.T) {
	setting.UI.IssuePagingNum = 1
	require.NoError(t, unittest.LoadFixtures())

	ctx, _ := contexttest.MockContext(t, "issues")
	contexttest.LoadUser(t, ctx, 2)
	ctx.Req.Form.Set("state", "closed")
	Issues(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())

	assert.EqualValues(t, true, ctx.Data["IsShowClosed"])
	assert.Len(t, ctx.Data["Issues"], 1)
}

func TestPulls(t *testing.T) {
	setting.UI.IssuePagingNum = 20
	require.NoError(t, unittest.LoadFixtures())

	ctx, _ := contexttest.MockContext(t, "pulls")
	contexttest.LoadUser(t, ctx, 2)
	ctx.Req.Form.Set("state", "open")
	ctx.Req.Form.Set("type", "your_repositories")
	Pulls(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())

	assert.Len(t, ctx.Data["Issues"], 5)
}

func TestMilestones(t *testing.T) {
	setting.UI.IssuePagingNum = 1
	require.NoError(t, unittest.LoadFixtures())

	ctx, _ := contexttest.MockContext(t, "milestones")
	contexttest.LoadUser(t, ctx, 2)
	ctx.SetParams("sort", "issues")
	ctx.Req.Form.Set("state", "closed")
	ctx.Req.Form.Set("sort", "furthestduedate")
	Milestones(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assert.EqualValues(t, map[int64]int64{1: 1}, ctx.Data["Counts"])
	assert.EqualValues(t, true, ctx.Data["IsShowClosed"])
	assert.EqualValues(t, "furthestduedate", ctx.Data["SortType"])
	assert.EqualValues(t, 1, ctx.Data["Total"])
	assert.Len(t, ctx.Data["Milestones"], 1)
	assert.Len(t, ctx.Data["Repos"], 2) // both repo 42 and 1 have milestones and both are owned by user 2
	assert.Equal(t, "user2/glob", ctx.Data["Repos"].(repo_model.RepositoryList)[0].FullName())
	assert.Equal(t, "user2/repo1", ctx.Data["Repos"].(repo_model.RepositoryList)[1].FullName())
}

func TestMilestonesForSpecificRepo(t *testing.T) {
	setting.UI.IssuePagingNum = 1
	require.NoError(t, unittest.LoadFixtures())

	ctx, _ := contexttest.MockContext(t, "milestones")
	contexttest.LoadUser(t, ctx, 2)
	ctx.SetParams("sort", "issues")
	ctx.SetParams("repo", "1")
	ctx.Req.Form.Set("state", "closed")
	ctx.Req.Form.Set("sort", "furthestduedate")
	Milestones(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assert.EqualValues(t, map[int64]int64{1: 1}, ctx.Data["Counts"])
	assert.EqualValues(t, true, ctx.Data["IsShowClosed"])
	assert.EqualValues(t, "furthestduedate", ctx.Data["SortType"])
	assert.EqualValues(t, 1, ctx.Data["Total"])
	assert.Len(t, ctx.Data["Milestones"], 1)
	assert.Len(t, ctx.Data["Repos"], 2) // both repo 42 and 1 have milestones and both are owned by user 2
}

func TestDashboardPagination(t *testing.T) {
	ctx, _ := contexttest.MockContext(t, "/", contexttest.MockContextOption{Render: templates.HTMLRenderer()})
	page := context.NewPagination(10, 3, 1, 3)

	setting.AppSubURL = "/SubPath"
	out, err := ctx.RenderToHTML("base/paginate", map[string]any{"Link": setting.AppSubURL, "Page": page})
	require.NoError(t, err)
	assert.Contains(t, out, `<a class=" item navigation" href="/SubPath/?page=2">`)

	setting.AppSubURL = ""
	out, err = ctx.RenderToHTML("base/paginate", map[string]any{"Link": setting.AppSubURL, "Page": page})
	require.NoError(t, err)
	assert.Contains(t, out, `<a class=" item navigation" href="/?page=2">`)
}

func TestOrgLabels(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	ctx, _ := contexttest.MockContext(t, "org/org3/issues")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadOrganization(t, ctx, 3)
	Issues(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())

	assert.True(t, ctx.Data["PageIsOrgIssues"].(bool))

	orgLabels := []struct {
		ID    int64
		OrgID int64
		Name  string
	}{
		{3, 3, "orglabel3"},
		{4, 3, "orglabel4"},
	}

	labels, ok := ctx.Data["Labels"].([]*issues_model.Label)

	assert.True(t, ok)

	if assert.Len(t, labels, len(orgLabels)) {
		for i, label := range labels {
			assert.Equal(t, orgLabels[i].OrgID, label.OrgID)
			assert.Equal(t, orgLabels[i].ID, label.ID)
			assert.Equal(t, orgLabels[i].Name, label.Name)
		}
	}
}
