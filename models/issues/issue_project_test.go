// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/project"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivateIssueProjects(t *testing.T) {
	defer tests.AddFixtures("models/fixtures/PrivateIssueProjects/")()
	require.NoError(t, unittest.PrepareTestDatabase())

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	t.Run("Organization project", func(t *testing.T) {
		org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
		orgProject := unittest.AssertExistsAndLoadBean(t, &project.Project{ID: 1001, OwnerID: org.ID})
		column := unittest.AssertExistsAndLoadBean(t, &project.Column{ID: 1001, ProjectID: orgProject.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, user2, org, optional.None[bool]())
			require.NoError(t, err)
			assert.Len(t, issueList, 1)
			assert.EqualValues(t, 6, issueList[0].ID)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.None[bool]())
			require.NoError(t, err)
			assert.EqualValues(t, 1, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.Some(true))
			require.NoError(t, err)
			assert.EqualValues(t, 0, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, orgProject, user2, org, optional.Some(false))
			require.NoError(t, err)
			assert.EqualValues(t, 1, issuesNum)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, nil, org, optional.None[bool]())
			require.NoError(t, err)
			assert.Empty(t, issueList)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, orgProject, nil, org, optional.None[bool]())
			require.NoError(t, err)
			assert.EqualValues(t, 0, issuesNum)
		})
	})

	t.Run("User project", func(t *testing.T) {
		userProject := unittest.AssertExistsAndLoadBean(t, &project.Project{ID: 1002, OwnerID: user2.ID})
		column := unittest.AssertExistsAndLoadBean(t, &project.Column{ID: 1002, ProjectID: userProject.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, user2, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.Len(t, issueList, 1)
			assert.EqualValues(t, 7, issueList[0].ID)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, userProject, user2, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.EqualValues(t, 1, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, userProject, user2, nil, optional.Some(true))
			require.NoError(t, err)
			assert.EqualValues(t, 0, issuesNum)

			issuesNum, err = issues.NumIssuesInProject(db.DefaultContext, userProject, user2, nil, optional.Some(false))
			require.NoError(t, err)
			assert.EqualValues(t, 1, issuesNum)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			issueList, err := issues.LoadIssuesFromColumn(db.DefaultContext, column, nil, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.Empty(t, issueList)

			issuesNum, err := issues.NumIssuesInProject(db.DefaultContext, userProject, nil, nil, optional.None[bool]())
			require.NoError(t, err)
			assert.EqualValues(t, 0, issuesNum)
		})
	})
}
