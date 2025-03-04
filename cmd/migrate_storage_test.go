// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	packages_module "code.gitea.io/gitea/modules/packages"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/test"
	packages_service "code.gitea.io/gitea/services/packages"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createLocalStorage(t *testing.T) (storage.ObjectStorage, string) {
	t.Helper()

	p := t.TempDir()

	storage, err := storage.NewLocalStorage(
		t.Context(),
		&setting.Storage{
			Path: p,
		})
	require.NoError(t, err)

	return storage, p
}

func TestMigratePackages(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	creator := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	content := "package main\n\nfunc main() {\nfmt.Println(\"hi\")\n}\n"
	buf, err := packages_module.CreateHashedBufferFromReaderWithSize(strings.NewReader(content), 1024)
	require.NoError(t, err)
	defer buf.Close()

	v, f, err := packages_service.CreatePackageAndAddFile(db.DefaultContext, &packages_service.PackageCreationInfo{
		PackageInfo: packages_service.PackageInfo{
			Owner:       creator,
			PackageType: packages.TypeGeneric,
			Name:        "test",
			Version:     "1.0.0",
		},
		Creator:           creator,
		SemverCompatible:  true,
		VersionProperties: map[string]string{},
	}, &packages_service.PackageFileCreationInfo{
		PackageFileInfo: packages_service.PackageFileInfo{
			Filename: "a.go",
		},
		Creator: creator,
		Data:    buf,
		IsLead:  true,
	})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.NotNil(t, f)

	ctx := t.Context()

	dstStorage, p := createLocalStorage(t)

	err = migratePackages(ctx, dstStorage)
	require.NoError(t, err)

	entries, err := os.ReadDir(p)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.EqualValues(t, "01", entries[0].Name())
	assert.EqualValues(t, "tmp", entries[1].Name())
}

func TestMigrateActionsArtifacts(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	srcStorage, _ := createLocalStorage(t)
	defer test.MockVariableValue(&storage.ActionsArtifacts, srcStorage)()
	id := int64(42)

	addArtifact := func(storagePath string, status actions.ArtifactStatus) {
		id++
		artifact := &actions.ActionArtifact{
			ID:           id,
			ArtifactName: storagePath,
			StoragePath:  storagePath,
			Status:       int64(status),
		}
		_, err := db.GetEngine(db.DefaultContext).Insert(artifact)
		require.NoError(t, err)
		srcStorage.Save(storagePath, strings.NewReader(storagePath), -1)
	}

	exists := "/exists"
	addArtifact(exists, actions.ArtifactStatusUploadConfirmed)

	expired := "/expired"
	addArtifact(expired, actions.ArtifactStatusExpired)

	notFound := "/notfound"
	addArtifact(notFound, actions.ArtifactStatusUploadConfirmed)
	srcStorage.Delete(notFound)

	dstStorage, _ := createLocalStorage(t)

	require.NoError(t, migrateActionsArtifacts(db.DefaultContext, dstStorage))

	object, err := dstStorage.Open(exists)
	require.NoError(t, err)
	buf, err := io.ReadAll(object)
	require.NoError(t, err)
	assert.Equal(t, exists, string(buf))

	_, err = dstStorage.Stat(expired)
	require.Error(t, err)

	_, err = dstStorage.Stat(notFound)
	require.Error(t, err)
}
