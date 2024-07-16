// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_IsOwnedBy(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	for _, testCase := range []struct {
		OrgID         int64
		UserID        int64
		ExpectedOwner bool
	}{
		{3, 2, true},
		{3, 1, false},
		{3, 3, false},
		{3, 4, false},
		{2, 2, false}, // user2 is not an organization
		{2, 3, false},
	} {
		org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: testCase.OrgID})
		isOwner, err := org.IsOwnedBy(db.DefaultContext, testCase.UserID)
		require.NoError(t, err)
		assert.Equal(t, testCase.ExpectedOwner, isOwner)
	}
}

func TestUser_IsOrgMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	for _, testCase := range []struct {
		OrgID          int64
		UserID         int64
		ExpectedMember bool
	}{
		{3, 2, true},
		{3, 4, true},
		{3, 1, false},
		{3, 3, false},
		{2, 2, false}, // user2 is not an organization
		{2, 3, false},
	} {
		org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: testCase.OrgID})
		isMember, err := org.IsOrgMember(db.DefaultContext, testCase.UserID)
		require.NoError(t, err)
		assert.Equal(t, testCase.ExpectedMember, isMember)
	}
}

func TestUser_GetTeam(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	team, err := org.GetTeam(db.DefaultContext, "team1")
	require.NoError(t, err)
	assert.Equal(t, org.ID, team.OrgID)
	assert.Equal(t, "team1", team.LowerName)

	_, err = org.GetTeam(db.DefaultContext, "does not exist")
	assert.True(t, organization.IsErrTeamNotExist(err))

	nonOrg := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 2})
	_, err = nonOrg.GetTeam(db.DefaultContext, "team")
	assert.True(t, organization.IsErrTeamNotExist(err))
}

func TestUser_GetOwnerTeam(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	team, err := org.GetOwnerTeam(db.DefaultContext)
	require.NoError(t, err)
	assert.Equal(t, org.ID, team.OrgID)

	nonOrg := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 2})
	_, err = nonOrg.GetOwnerTeam(db.DefaultContext)
	assert.True(t, organization.IsErrTeamNotExist(err))
}

func TestUser_GetTeams(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	teams, err := org.LoadTeams(db.DefaultContext)
	require.NoError(t, err)
	if assert.Len(t, teams, 5) {
		assert.Equal(t, int64(1), teams[0].ID)
		assert.Equal(t, int64(2), teams[1].ID)
		assert.Equal(t, int64(12), teams[2].ID)
		assert.Equal(t, int64(14), teams[3].ID)
		assert.Equal(t, int64(7), teams[4].ID)
	}
}

func TestUser_GetMembers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	members, _, err := org.GetMembers(db.DefaultContext)
	require.NoError(t, err)
	if assert.Len(t, members, 3) {
		assert.Equal(t, int64(2), members[0].ID)
		assert.Equal(t, int64(28), members[1].ID)
		assert.Equal(t, int64(4), members[2].ID)
	}
}

func TestGetOrgByName(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	org, err := organization.GetOrgByName(db.DefaultContext, "org3")
	require.NoError(t, err)
	assert.EqualValues(t, 3, org.ID)
	assert.Equal(t, "org3", org.Name)

	_, err = organization.GetOrgByName(db.DefaultContext, "user2") // user2 is an individual
	assert.True(t, organization.IsErrOrgNotExist(err))

	_, err = organization.GetOrgByName(db.DefaultContext, "") // corner case
	assert.True(t, organization.IsErrOrgNotExist(err))
}

func TestCountOrganizations(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	expected, err := db.GetEngine(db.DefaultContext).Where("type=?", user_model.UserTypeOrganization).Count(&organization.Organization{})
	require.NoError(t, err)
	cnt, err := db.Count[organization.Organization](db.DefaultContext, organization.FindOrgOptions{IncludePrivate: true})
	require.NoError(t, err)
	assert.Equal(t, expected, cnt)
}

func TestIsOrganizationOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(orgID, userID int64, expected bool) {
		isOwner, err := organization.IsOrganizationOwner(db.DefaultContext, orgID, userID)
		require.NoError(t, err)
		assert.EqualValues(t, expected, isOwner)
	}
	test(3, 2, true)
	test(3, 3, false)
	test(6, 5, true)
	test(6, 4, false)
	test(unittest.NonexistentID, unittest.NonexistentID, false)
}

func TestIsOrganizationMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(orgID, userID int64, expected bool) {
		isMember, err := organization.IsOrganizationMember(db.DefaultContext, orgID, userID)
		require.NoError(t, err)
		assert.EqualValues(t, expected, isMember)
	}
	test(3, 2, true)
	test(3, 3, false)
	test(3, 4, true)
	test(6, 5, true)
	test(6, 4, false)
	test(unittest.NonexistentID, unittest.NonexistentID, false)
}

func TestIsPublicMembership(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(orgID, userID int64, expected bool) {
		isMember, err := organization.IsPublicMembership(db.DefaultContext, orgID, userID)
		require.NoError(t, err)
		assert.EqualValues(t, expected, isMember)
	}
	test(3, 2, true)
	test(3, 3, false)
	test(3, 4, false)
	test(6, 5, true)
	test(6, 4, false)
	test(unittest.NonexistentID, unittest.NonexistentID, false)
}

func TestFindOrgs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	orgs, err := db.Find[organization.Organization](db.DefaultContext, organization.FindOrgOptions{
		UserID:         4,
		IncludePrivate: true,
	})
	require.NoError(t, err)
	if assert.Len(t, orgs, 1) {
		assert.EqualValues(t, 3, orgs[0].ID)
	}

	orgs, err = db.Find[organization.Organization](db.DefaultContext, organization.FindOrgOptions{
		UserID:         4,
		IncludePrivate: false,
	})
	require.NoError(t, err)
	assert.Empty(t, orgs)

	total, err := db.Count[organization.Organization](db.DefaultContext, organization.FindOrgOptions{
		UserID:         4,
		IncludePrivate: true,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
}

func TestGetOrgUsersByOrgID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	orgUsers, err := organization.GetOrgUsersByOrgID(db.DefaultContext, &organization.FindOrgMembersOpts{
		ListOptions: db.ListOptions{},
		OrgID:       3,
		PublicOnly:  false,
	})
	require.NoError(t, err)
	if assert.Len(t, orgUsers, 3) {
		assert.Equal(t, organization.OrgUser{
			ID:       orgUsers[0].ID,
			OrgID:    3,
			UID:      2,
			IsPublic: true,
		}, *orgUsers[0])
		assert.Equal(t, organization.OrgUser{
			ID:       orgUsers[1].ID,
			OrgID:    3,
			UID:      4,
			IsPublic: false,
		}, *orgUsers[1])
		assert.Equal(t, organization.OrgUser{
			ID:       orgUsers[2].ID,
			OrgID:    3,
			UID:      28,
			IsPublic: true,
		}, *orgUsers[2])
	}

	orgUsers, err = organization.GetOrgUsersByOrgID(db.DefaultContext, &organization.FindOrgMembersOpts{
		ListOptions: db.ListOptions{},
		OrgID:       unittest.NonexistentID,
		PublicOnly:  false,
	})
	require.NoError(t, err)
	assert.Empty(t, orgUsers)
}

func TestChangeOrgUserStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(orgID, userID int64, public bool) {
		require.NoError(t, organization.ChangeOrgUserStatus(db.DefaultContext, orgID, userID, public))
		orgUser := unittest.AssertExistsAndLoadBean(t, &organization.OrgUser{OrgID: orgID, UID: userID})
		assert.Equal(t, public, orgUser.IsPublic)
	}

	testSuccess(3, 2, false)
	testSuccess(3, 2, false)
	testSuccess(3, 4, true)
	require.NoError(t, organization.ChangeOrgUserStatus(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID, true))
}

func TestUser_GetUserTeamIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	testSuccess := func(userID int64, expected []int64) {
		teamIDs, err := org.GetUserTeamIDs(db.DefaultContext, userID)
		require.NoError(t, err)
		assert.Equal(t, expected, teamIDs)
	}
	testSuccess(2, []int64{1, 2, 14})
	testSuccess(4, []int64{2})
	testSuccess(unittest.NonexistentID, []int64{})
}

func TestAccessibleReposEnv_CountRepos(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	testSuccess := func(userID, expectedCount int64) {
		env, err := organization.AccessibleReposEnv(db.DefaultContext, org, userID)
		require.NoError(t, err)
		count, err := env.CountRepos()
		require.NoError(t, err)
		assert.EqualValues(t, expectedCount, count)
	}
	testSuccess(2, 3)
	testSuccess(4, 2)
}

func TestAccessibleReposEnv_RepoIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	testSuccess := func(userID int64, expectedRepoIDs []int64) {
		env, err := organization.AccessibleReposEnv(db.DefaultContext, org, userID)
		require.NoError(t, err)
		repoIDs, err := env.RepoIDs(1, 100)
		require.NoError(t, err)
		assert.Equal(t, expectedRepoIDs, repoIDs)
	}
	testSuccess(2, []int64{3, 5, 32})
	testSuccess(4, []int64{3, 32})
}

func TestAccessibleReposEnv_Repos(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	testSuccess := func(userID int64, expectedRepoIDs []int64) {
		env, err := organization.AccessibleReposEnv(db.DefaultContext, org, userID)
		require.NoError(t, err)
		repos, err := env.Repos(1, 100)
		require.NoError(t, err)
		expectedRepos := make(repo_model.RepositoryList, len(expectedRepoIDs))
		for i, repoID := range expectedRepoIDs {
			expectedRepos[i] = unittest.AssertExistsAndLoadBean(t,
				&repo_model.Repository{ID: repoID})
		}
		assert.Equal(t, expectedRepos, repos)
	}
	testSuccess(2, []int64{3, 5, 32})
	testSuccess(4, []int64{3, 32})
}

func TestAccessibleReposEnv_MirrorRepos(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	testSuccess := func(userID int64, expectedRepoIDs []int64) {
		env, err := organization.AccessibleReposEnv(db.DefaultContext, org, userID)
		require.NoError(t, err)
		repos, err := env.MirrorRepos()
		require.NoError(t, err)
		expectedRepos := make(repo_model.RepositoryList, len(expectedRepoIDs))
		for i, repoID := range expectedRepoIDs {
			expectedRepos[i] = unittest.AssertExistsAndLoadBean(t,
				&repo_model.Repository{ID: repoID})
		}
		assert.Equal(t, expectedRepos, repos)
	}
	testSuccess(2, []int64{5})
	testSuccess(4, []int64{})
}

func TestHasOrgVisibleTypePublic(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	const newOrgName = "test-org-public"
	org := &organization.Organization{
		Name:       newOrgName,
		Visibility: structs.VisibleTypePublic,
	}

	unittest.AssertNotExistsBean(t, &user_model.User{Name: org.Name, Type: user_model.UserTypeOrganization})
	require.NoError(t, organization.CreateOrganization(db.DefaultContext, org, owner))
	org = unittest.AssertExistsAndLoadBean(t,
		&organization.Organization{Name: org.Name, Type: user_model.UserTypeOrganization})
	test1 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), owner)
	test2 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), org3)
	test3 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), nil)
	assert.True(t, test1) // owner of org
	assert.True(t, test2) // user not a part of org
	assert.True(t, test3) // logged out user
}

func TestHasOrgVisibleTypeLimited(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	const newOrgName = "test-org-limited"
	org := &organization.Organization{
		Name:       newOrgName,
		Visibility: structs.VisibleTypeLimited,
	}

	unittest.AssertNotExistsBean(t, &user_model.User{Name: org.Name, Type: user_model.UserTypeOrganization})
	require.NoError(t, organization.CreateOrganization(db.DefaultContext, org, owner))
	org = unittest.AssertExistsAndLoadBean(t,
		&organization.Organization{Name: org.Name, Type: user_model.UserTypeOrganization})
	test1 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), owner)
	test2 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), org3)
	test3 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), nil)
	assert.True(t, test1)  // owner of org
	assert.True(t, test2)  // user not a part of org
	assert.False(t, test3) // logged out user
}

func TestHasOrgVisibleTypePrivate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	const newOrgName = "test-org-private"
	org := &organization.Organization{
		Name:       newOrgName,
		Visibility: structs.VisibleTypePrivate,
	}

	unittest.AssertNotExistsBean(t, &user_model.User{Name: org.Name, Type: user_model.UserTypeOrganization})
	require.NoError(t, organization.CreateOrganization(db.DefaultContext, org, owner))
	org = unittest.AssertExistsAndLoadBean(t,
		&organization.Organization{Name: org.Name, Type: user_model.UserTypeOrganization})
	test1 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), owner)
	test2 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), org3)
	test3 := organization.HasOrgOrUserVisible(db.DefaultContext, org.AsUser(), nil)
	assert.True(t, test1)  // owner of org
	assert.False(t, test2) // user not a part of org
	assert.False(t, test3) // logged out user
}

func TestGetUsersWhoCanCreateOrgRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	users, err := organization.GetUsersWhoCanCreateOrgRepo(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	var ids []int64
	for i := range users {
		ids = append(ids, users[i].ID)
	}
	assert.ElementsMatch(t, ids, []int64{2, 28})

	users, err = organization.GetUsersWhoCanCreateOrgRepo(db.DefaultContext, 7)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.NotNil(t, users[5])
}

func TestUser_RemoveOrgRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: org.ID})

	// remove a repo that does belong to org
	unittest.AssertExistsAndLoadBean(t, &organization.TeamRepo{RepoID: repo.ID, OrgID: org.ID})
	require.NoError(t, organization.RemoveOrgRepo(db.DefaultContext, org.ID, repo.ID))
	unittest.AssertNotExistsBean(t, &organization.TeamRepo{RepoID: repo.ID, OrgID: org.ID})
	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repo.ID}) // repo should still exist

	// remove a repo that does not belong to org
	require.NoError(t, organization.RemoveOrgRepo(db.DefaultContext, org.ID, repo.ID))
	unittest.AssertNotExistsBean(t, &organization.TeamRepo{RepoID: repo.ID, OrgID: org.ID})

	require.NoError(t, organization.RemoveOrgRepo(db.DefaultContext, org.ID, unittest.NonexistentID))

	unittest.CheckConsistencyFor(t,
		&user_model.User{ID: org.ID},
		&organization.Team{OrgID: org.ID},
		&repo_model.Repository{ID: repo.ID})
}

func TestCreateOrganization(t *testing.T) {
	// successful creation of org
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	const newOrgName = "neworg"
	org := &organization.Organization{
		Name: newOrgName,
	}

	unittest.AssertNotExistsBean(t, &user_model.User{Name: newOrgName, Type: user_model.UserTypeOrganization})
	require.NoError(t, organization.CreateOrganization(db.DefaultContext, org, owner))
	org = unittest.AssertExistsAndLoadBean(t,
		&organization.Organization{Name: newOrgName, Type: user_model.UserTypeOrganization})
	ownerTeam := unittest.AssertExistsAndLoadBean(t,
		&organization.Team{Name: organization.OwnerTeamName, OrgID: org.ID})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{UID: owner.ID, TeamID: ownerTeam.ID})
	unittest.CheckConsistencyFor(t, &user_model.User{}, &organization.Team{})
}

func TestCreateOrganization2(t *testing.T) {
	// unauthorized creation of org
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	const newOrgName = "neworg"
	org := &organization.Organization{
		Name: newOrgName,
	}

	unittest.AssertNotExistsBean(t, &organization.Organization{Name: newOrgName, Type: user_model.UserTypeOrganization})
	err := organization.CreateOrganization(db.DefaultContext, org, owner)
	require.Error(t, err)
	assert.True(t, organization.IsErrUserNotAllowedCreateOrg(err))
	unittest.AssertNotExistsBean(t, &organization.Organization{Name: newOrgName, Type: user_model.UserTypeOrganization})
	unittest.CheckConsistencyFor(t, &organization.Organization{}, &organization.Team{})
}

func TestCreateOrganization3(t *testing.T) {
	// create org with same name as existent org
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := &organization.Organization{Name: "org3"}                       // should already exist
	unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: org.Name}) // sanity check
	err := organization.CreateOrganization(db.DefaultContext, org, owner)
	require.Error(t, err)
	assert.True(t, user_model.IsErrUserAlreadyExist(err))
	unittest.CheckConsistencyFor(t, &user_model.User{}, &organization.Team{})
}

func TestCreateOrganization4(t *testing.T) {
	// create org with unusable name
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	err := organization.CreateOrganization(db.DefaultContext, &organization.Organization{Name: "assets"}, owner)
	require.Error(t, err)
	assert.True(t, db.IsErrNameReserved(err))
	unittest.CheckConsistencyFor(t, &organization.Organization{}, &organization.Team{})
}
