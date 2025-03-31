// SPDX-License-Identifier: MIT

//
// Tests verifying the Forgejo documentation on storage settings is correct
//
// https://forgejo.org/docs/v1.20/admin/storage/
//

package setting

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoDocs_StorageTypes(t *testing.T) {
	iniStr := `
[server]
APP_DATA_PATH = /
`
	testStorageTypesDefaultAndSpecificStorage(t, iniStr)
}

func testStorageGetPath(storage *Storage) string {
	if storage.Type == MinioStorageType {
		return storage.MinioConfig.BasePath
	}
	return storage.Path
}

var testSectionToBasePath = map[string]string{
	"attachment":          "attachments",
	"lfs":                 "lfs",
	"avatar":              "avatars",
	"repo-avatar":         "repo-avatars",
	"repo-archive":        "repo-archive",
	"packages":            "packages",
	"storage.actions_log": "actions_log",
	"actions.artifacts":   "actions_artifacts",
}

type testSectionToPathFun func(StorageType, string) string

func testBuildPath(t StorageType, path string) string {
	if t == LocalStorageType {
		return "/" + path
	}
	return path + "/"
}

func testSectionToPath(t StorageType, section string) string {
	return testBuildPath(t, testSectionToBasePath[section])
}

func testSpecificPath(t StorageType, section string) string {
	if t == LocalStorageType {
		return "/specific_local_path"
	}
	return "specific_s3_base_path/"
}

func testDefaultDir(t StorageType) string {
	if t == LocalStorageType {
		return "default_local_path"
	}
	return "default_s3_base_path"
}

func testDefaultPath(t StorageType) string {
	return testBuildPath(t, testDefaultDir(t))
}

func testSectionToDefaultPath(t StorageType, section string) string {
	return testBuildPath(t, filepath.Join(testDefaultDir(t), testSectionToPath(t, section)))
}

func testLegacyPath(t StorageType, section string) string {
	return testBuildPath(t, fmt.Sprintf("legacy_%s_path", section))
}

func testStorageTypeToSetting(t StorageType) string {
	if t == LocalStorageType {
		return "PATH"
	}
	return "MINIO_BASE_PATH"
}

var testSectionToLegacy = map[string]string{
	"lfs": fmt.Sprintf(`
[server]
APP_DATA_PATH = /
LFS_CONTENT_PATH = %s
`, testLegacyPath(LocalStorageType, "lfs")),
	"avatar": fmt.Sprintf(`
[picture]
AVATAR_UPLOAD_PATH = %s
`, testLegacyPath(LocalStorageType, "avatar")),
	"repo-avatar": fmt.Sprintf(`
[picture]
REPOSITORY_AVATAR_UPLOAD_PATH = %s
`, testLegacyPath(LocalStorageType, "repo-avatar")),
}

func testStorageTypesDefaultAndSpecificStorage(t *testing.T, iniStr string) {
	storageType := MinioStorageType
	t.Run(string(storageType), func(t *testing.T) {
		t.Run("override type minio", func(t *testing.T) {
			storageSection := `
[storage]
STORAGE_TYPE = minio
`
			testStorageTypesSpecificStorages(t, iniStr+storageSection, storageType, testSectionToPath, testSectionToPath)
		})
	})

	storageType = LocalStorageType

	t.Run(string(storageType), func(t *testing.T) {
		storageSection := ""
		testStorageTypesSpecificStorages(t, iniStr+storageSection, storageType, testSectionToPath, testSectionToPath)

		t.Run("override type local", func(t *testing.T) {
			storageSection := `
[storage]
STORAGE_TYPE = local
`
			testStorageTypesSpecificStorages(t, iniStr+storageSection, storageType, testSectionToPath, testSectionToPath)

			storageSection = fmt.Sprintf(`
[storage]
STORAGE_TYPE = local
PATH = %s
`, testDefaultPath(LocalStorageType))
			testStorageTypesSpecificStorageSections(t, iniStr+storageSection, storageType, testSectionToDefaultPath, testSectionToPath)
		})
	})
}

func testStorageTypesSpecificStorageSections(t *testing.T, iniStr string, defaultStorageType StorageType, defaultStorageTypePath, testSectionToPath testSectionToPathFun) {
	testSectionsMap := map[string]**Storage{
		"attachment":   &Attachment.Storage,
		"lfs":          &LFS.Storage,
		"avatar":       &Avatar.Storage,
		"repo-avatar":  &RepoAvatar.Storage,
		"repo-archive": &RepoArchive.Storage,
		"packages":     &Packages.Storage,
		// there are inconsistencies in how actions storage is determined in v1.20
		// it is still alpha and undocumented and is ignored for now
		//"storage.actions_log": &Actions.LogStorage,
		//"actions.artifacts":   &Actions.ArtifactStorage,
	}

	for sectionName, storage := range testSectionsMap {
		t.Run(sectionName, func(t *testing.T) {
			testStorageTypesSpecificStorage(t, iniStr, defaultStorageType, defaultStorageTypePath, testSectionToPath, sectionName, storage)
		})
	}
}

func testStorageTypesSpecificStorages(t *testing.T, iniStr string, defaultStorageType StorageType, defaultStorageTypePath, testSectionToPath testSectionToPathFun) {
	testSectionsMap := map[string]**Storage{
		"attachment":          &Attachment.Storage,
		"lfs":                 &LFS.Storage,
		"avatar":              &Avatar.Storage,
		"repo-avatar":         &RepoAvatar.Storage,
		"repo-archive":        &RepoArchive.Storage,
		"packages":            &Packages.Storage,
		"storage.actions_log": &Actions.LogStorage,
		"actions.artifacts":   &Actions.ArtifactStorage,
	}

	for sectionName, storage := range testSectionsMap {
		t.Run(sectionName, func(t *testing.T) {
			if legacy, ok := testSectionToLegacy[sectionName]; ok {
				if defaultStorageType == LocalStorageType {
					t.Run("legacy local", func(t *testing.T) {
						testStorageTypesSpecificStorage(t, iniStr+legacy, LocalStorageType, testLegacyPath, testSectionToPath, sectionName, storage)
						testStorageTypesSpecificStorageTypeOverride(t, iniStr+legacy, LocalStorageType, testLegacyPath, testSectionToPath, sectionName, storage)
					})
				} else {
					t.Run("legacy minio", func(t *testing.T) {
						testStorageTypesSpecificStorage(t, iniStr+legacy, MinioStorageType, defaultStorageTypePath, testSectionToPath, sectionName, storage)
						testStorageTypesSpecificStorageTypeOverride(t, iniStr+legacy, LocalStorageType, testLegacyPath, testSectionToPath, sectionName, storage)
					})
				}
			}
			for _, specificStorageType := range storageTypes {
				testStorageTypesSpecificStorageTypeOverride(t, iniStr, specificStorageType, defaultStorageTypePath, testSectionToPath, sectionName, storage)
			}
		})
	}
}

func testStorageTypesSpecificStorage(t *testing.T, iniStr string, defaultStorageType StorageType, defaultStorageTypePath, testSectionToPath testSectionToPathFun, sectionName string, storage **Storage) {
	var section string

	//
	// Specific section is absent
	//
	testStoragePathMatch(t, iniStr, defaultStorageType, defaultStorageTypePath, sectionName, storage)

	//
	// Specific section is empty
	//
	section = fmt.Sprintf(`
[%s]
`,
		sectionName)
	testStoragePathMatch(t, iniStr+section, defaultStorageType, defaultStorageTypePath, sectionName, storage)

	//
	// Specific section with a path override
	//
	section = fmt.Sprintf(`
[%s]
%s = %s
`,
		sectionName,
		testStorageTypeToSetting(defaultStorageType),
		testSpecificPath(defaultStorageType, ""))
	testStoragePathMatch(t, iniStr+section, defaultStorageType, testSpecificPath, sectionName, storage)
}

func testStorageTypesSpecificStorageTypeOverride(t *testing.T, iniStr string, overrideStorageType StorageType, defaultStorageTypePath, testSectionToPath testSectionToPathFun, sectionName string, storage **Storage) {
	var section string
	t.Run("specific-"+string(overrideStorageType), func(t *testing.T) {
		//
		// Specific section with a path and storage type override
		//
		section = fmt.Sprintf(`
[%s]
STORAGE_TYPE = %s
%s = %s
`,
			sectionName,
			overrideStorageType,
			testStorageTypeToSetting(overrideStorageType),
			testSpecificPath(overrideStorageType, ""))
		testStoragePathMatch(t, iniStr+section, overrideStorageType, testSpecificPath, sectionName, storage)

		//
		// Specific section with type override
		//
		section = fmt.Sprintf(`
[%s]
STORAGE_TYPE = %s
`,
			sectionName,
			overrideStorageType)
		testStoragePathMatch(t, iniStr+section, overrideStorageType, defaultStorageTypePath, sectionName, storage)
	})
}

func testStoragePathMatch(t *testing.T, iniStr string, storageType StorageType, testSectionToPath testSectionToPathFun, section string, storage **Storage) {
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err, iniStr)
	require.NoError(t, loadCommonSettingsFrom(cfg), iniStr)
	assert.Equal(t, testSectionToPath(storageType, section), testStorageGetPath(*storage), iniStr)
	assert.Equal(t, storageType, (*storage).Type, iniStr)
}
