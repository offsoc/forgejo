// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages_test

import (
	"testing"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/require"
)

func prepareExamplePackage(t *testing.T) *packages_model.Package {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})

	p0 := &packages_model.Package{
		OwnerID:   owner.ID,
		RepoID:    repo.ID,
		LowerName: "package",
		Type:      packages_model.TypeGeneric,
	}

	p, err := packages_model.TryInsertPackage(db.DefaultContext, p0)
	require.NotNil(t, p)
	require.NoError(t, err)
	require.Equal(t, *p0, *p)
	return p
}

func deletePackage(t *testing.T, p *packages_model.Package) {
	err := packages_model.DeletePackageByID(db.DefaultContext, p.ID)
	require.NoError(t, err)
}

func TestTryInsertPackage(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	p0 := &packages_model.Package{
		OwnerID:   owner.ID,
		LowerName: "package",
	}

	// Insert package should return the package and yield no error
	p, err := packages_model.TryInsertPackage(db.DefaultContext, p0)
	require.NotNil(t, p)
	require.NoError(t, err)
	require.Equal(t, *p0, *p)

	// Insert same package again should return the same package and yield ErrDuplicatePackage
	p, err = packages_model.TryInsertPackage(db.DefaultContext, p0)
	require.NotNil(t, p)
	require.IsType(t, packages_model.ErrDuplicatePackage, err)
	require.Equal(t, *p0, *p)

	err = packages_model.DeletePackageByID(db.DefaultContext, p0.ID)
	require.NoError(t, err)
}

func TestGetPackageByID(t *testing.T) {
	p0 := prepareExamplePackage(t)

	// Get package should return package and yield no error
	p, err := packages_model.GetPackageByID(db.DefaultContext, p0.ID)
	require.NotNil(t, p)
	require.Equal(t, *p0, *p)
	require.NoError(t, err)

	// Get package with non-existng ID should yield ErrPackageNotExist
	p, err = packages_model.GetPackageByID(db.DefaultContext, 999)
	require.Nil(t, p)
	require.Error(t, err)
	require.IsType(t, packages_model.ErrPackageNotExist, err)

	deletePackage(t, p0)
}

func TestDeletePackageByID(t *testing.T) {
	p0 := prepareExamplePackage(t)

	// Delete existing package should yield no error
	err := packages_model.DeletePackageByID(db.DefaultContext, p0.ID)
	require.NoError(t, err)

	// Delete (now) non-existing package should yield ErrPackageNotExist
	err = packages_model.DeletePackageByID(db.DefaultContext, p0.ID)
	require.Error(t, err)
	require.IsType(t, packages_model.ErrPackageNotExist, err)
}

func TestSetRepositoryLink(t *testing.T) {
	p0 := prepareExamplePackage(t)

	// Set repository link to package should yield no error and package RepoID should be updated
	err := packages_model.SetRepositoryLink(db.DefaultContext, p0.ID, 5)
	require.NoError(t, err)

	p, err := packages_model.GetPackageByID(db.DefaultContext, p0.ID)
	require.NoError(t, err)
	require.EqualValues(t, 5, p.RepoID)

	// Set repository link to non-existing package should yied ErrPackageNotExist
	err = packages_model.SetRepositoryLink(db.DefaultContext, 999, 5)
	require.Error(t, err)
	require.IsType(t, packages_model.ErrPackageNotExist, err)

	deletePackage(t, p0)
}

func TestUnlinkRepositoryFromAllPackages(t *testing.T) {
	p0 := prepareExamplePackage(t)

	// Unlink repository from all packages should yield no error and package with p0.ID should have RepoID 0
	err := packages_model.UnlinkRepositoryFromAllPackages(db.DefaultContext, p0.RepoID)
	require.NoError(t, err)

	p, err := packages_model.GetPackageByID(db.DefaultContext, p0.ID)
	require.NoError(t, err)
	require.EqualValues(t, 0, p.RepoID)

	// Unlink repository again from all packages should also yield no error
	err = packages_model.UnlinkRepositoryFromAllPackages(db.DefaultContext, p0.RepoID)
	require.NoError(t, err)

	deletePackage(t, p0)
}

func TestGetPackageByName(t *testing.T) {
	p0 := prepareExamplePackage(t)

	// Get package should return package and yield no error
	p, err := packages_model.GetPackageByName(db.DefaultContext, p0.OwnerID, p0.Type, p0.LowerName)
	require.NotNil(t, p)
	require.Equal(t, *p0, *p)
	require.NoError(t, err)

	// Get package with uppercase name should return package and yield no error
	p, err = packages_model.GetPackageByName(db.DefaultContext, p0.OwnerID, p0.Type, "Package")
	require.NotNil(t, p)
	require.Equal(t, *p0, *p)
	require.NoError(t, err)

	// Get package with wrong owner ID, type or name should return no package and yield ErrPackageNotExist
	p, err = packages_model.GetPackageByName(db.DefaultContext, 999, p0.Type, p0.LowerName)
	require.Nil(t, p)
	require.Error(t, err)
	require.IsType(t, packages_model.ErrPackageNotExist, err)
	p, err = packages_model.GetPackageByName(db.DefaultContext, p0.OwnerID, packages_model.TypeDebian, p0.LowerName)
	require.Nil(t, p)
	require.Error(t, err)
	require.IsType(t, packages_model.ErrPackageNotExist, err)
	p, err = packages_model.GetPackageByName(db.DefaultContext, p0.OwnerID, p0.Type, "package1")
	require.Nil(t, p)
	require.Error(t, err)
	require.IsType(t, packages_model.ErrPackageNotExist, err)

	deletePackage(t, p0)
}

func TestHasCountPackages(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})

	p, err := packages_model.TryInsertPackage(db.DefaultContext, &packages_model.Package{
		OwnerID:   owner.ID,
		RepoID:    repo.ID,
		LowerName: "package",
	})
	require.NotNil(t, p)
	require.NoError(t, err)

	// A package without package versions gets automatically cleaned up and should return false for owner
	has, err := packages_model.HasOwnerPackages(db.DefaultContext, owner.ID)
	require.False(t, has)
	require.NoError(t, err)
	count, err := packages_model.CountOwnerPackages(db.DefaultContext, owner.ID)
	require.EqualValues(t, 0, count)
	require.NoError(t, err)

	// A package without package versions gets automatically cleaned up and should return false for repository
	has, err = packages_model.HasRepositoryPackages(db.DefaultContext, repo.ID)
	require.False(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountRepositoryPackages(db.DefaultContext, repo.ID)
	require.EqualValues(t, 0, count)
	require.NoError(t, err)

	pv, err := packages_model.GetOrInsertVersion(db.DefaultContext, &packages_model.PackageVersion{
		PackageID:    p.ID,
		LowerVersion: "internal",
		IsInternal:   true,
	})
	require.NotNil(t, pv)
	require.NoError(t, err)

	// A package with an internal package version gets automatically cleaned up and should return false
	has, err = packages_model.HasOwnerPackages(db.DefaultContext, owner.ID)
	require.False(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountOwnerPackages(db.DefaultContext, owner.ID)
	require.EqualValues(t, 0, count)
	require.NoError(t, err)
	has, err = packages_model.HasRepositoryPackages(db.DefaultContext, repo.ID)
	require.False(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountRepositoryPackages(db.DefaultContext, repo.ID)
	require.EqualValues(t, 0, count)
	require.NoError(t, err)

	pv, err = packages_model.GetOrInsertVersion(db.DefaultContext, &packages_model.PackageVersion{
		PackageID:    p.ID,
		LowerVersion: "normal",
		IsInternal:   false,
	})
	require.NotNil(t, pv)
	require.NoError(t, err)

	// A package with a normal package version should return true
	has, err = packages_model.HasOwnerPackages(db.DefaultContext, owner.ID)
	require.True(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountOwnerPackages(db.DefaultContext, owner.ID)
	require.EqualValues(t, 1, count)
	require.NoError(t, err)
	has, err = packages_model.HasRepositoryPackages(db.DefaultContext, repo.ID)
	require.True(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountRepositoryPackages(db.DefaultContext, repo.ID)
	require.EqualValues(t, 1, count)
	require.NoError(t, err)

	pv2, err := packages_model.GetOrInsertVersion(db.DefaultContext, &packages_model.PackageVersion{
		PackageID:    p.ID,
		LowerVersion: "normal2",
		IsInternal:   false,
	})
	require.NotNil(t, pv2)
	require.NoError(t, err)

	// A package withmultiple package versions should be counted only once
	has, err = packages_model.HasOwnerPackages(db.DefaultContext, owner.ID)
	require.True(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountOwnerPackages(db.DefaultContext, owner.ID)
	require.EqualValues(t, 1, count)
	require.NoError(t, err)
	has, err = packages_model.HasRepositoryPackages(db.DefaultContext, repo.ID)
	require.True(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountRepositoryPackages(db.DefaultContext, repo.ID)
	require.EqualValues(t, 1, count)
	require.NoError(t, err)

	// For owner ID 0 there should be no packages
	has, err = packages_model.HasOwnerPackages(db.DefaultContext, 0)
	require.False(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountOwnerPackages(db.DefaultContext, 0)
	require.EqualValues(t, 0, count)
	require.NoError(t, err)

	// For repo ID 0 there should be no packages
	has, err = packages_model.HasRepositoryPackages(db.DefaultContext, 0)
	require.False(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountRepositoryPackages(db.DefaultContext, 0)
	require.EqualValues(t, 0, count)
	require.NoError(t, err)

	p1, err := packages_model.TryInsertPackage(db.DefaultContext, &packages_model.Package{
		OwnerID:   owner.ID,
		LowerName: "package0",
	})
	require.NotNil(t, p1)
	require.NoError(t, err)
	p1v, err := packages_model.GetOrInsertVersion(db.DefaultContext, &packages_model.PackageVersion{
		PackageID:    p1.ID,
		LowerVersion: "normal",
		IsInternal:   false,
	})
	require.NotNil(t, p1v)
	require.NoError(t, err)

	// Owner owner.ID should have two packages now
	has, err = packages_model.HasOwnerPackages(db.DefaultContext, owner.ID)
	require.True(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountOwnerPackages(db.DefaultContext, owner.ID)
	require.EqualValues(t, 2, count)
	require.NoError(t, err)

	// For repo ID 0 there should be now one package, because p1 is not assigned to a repo
	has, err = packages_model.HasRepositoryPackages(db.DefaultContext, 0)
	require.True(t, has)
	require.NoError(t, err)
	count, err = packages_model.CountRepositoryPackages(db.DefaultContext, 0)
	require.EqualValues(t, 1, count)
	require.NoError(t, err)
}
