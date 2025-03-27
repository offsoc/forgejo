// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package debian

import (
	"strings"
	"testing"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/packages"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	packages_service "forgejo.org/services/packages"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func preparePackage(t *testing.T, owner *user_model.User, name string) {
	t.Helper()

	data, err := packages.CreateHashedBufferFromReader(strings.NewReader("data"))
	require.NoError(t, err)

	_, _, err = packages_service.CreatePackageOrAddFileToExisting(
		db.DefaultContext,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       owner,
				PackageType: packages_model.TypeDebian,
				Name:        name,
			},
			Creator: owner,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: name,
			},
			Data:    data,
			Creator: owner,
			IsLead:  true,
		},
	)

	require.NoError(t, err)
}

func TestSearchPackages(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	preparePackage(t, user2, "debian-1")
	preparePackage(t, user2, "debian-2")
	preparePackage(t, user3, "debian-1")

	packageFiles := []string{}
	require.NoError(t, SearchPackages(db.DefaultContext, &PackageSearchOptions{
		OwnerID: user2.ID,
	}, func(pfd *packages_model.PackageFileDescriptor) {
		assert.NotNil(t, pfd)
		packageFiles = append(packageFiles, pfd.File.Name)
	}))

	assert.Len(t, packageFiles, 2)
	assert.Contains(t, packageFiles, "debian-1")
	assert.Contains(t, packageFiles, "debian-2")

	packageFiles = []string{}
	require.NoError(t, SearchPackages(db.DefaultContext, &PackageSearchOptions{
		OwnerID: user3.ID,
	}, func(pfd *packages_model.PackageFileDescriptor) {
		assert.NotNil(t, pfd)
		packageFiles = append(packageFiles, pfd.File.Name)
	}))

	assert.Len(t, packageFiles, 1)
	assert.Contains(t, packageFiles, "debian-1")
}
