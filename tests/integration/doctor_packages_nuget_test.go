// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	doctor "forgejo.org/services/doctor"
	packages_service "forgejo.org/services/packages"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorPackagesNuget(t *testing.T) {
	defer tests.PrepareTestEnv(t, 1)()
	// use local storage for tests because minio is too flaky
	defer test.MockVariableValue(&setting.Packages.Storage.Type, setting.LocalStorageType)()

	logger := log.GetLogger("doctor")

	ctx := db.DefaultContext

	packageName := "test.package"
	packageVersion := "1.0.3"
	packageAuthors := "KN4CK3R"
	packageDescription := "Gitea Test Package"

	createPackage := func(id, version string) io.Reader {
		var buf bytes.Buffer
		archive := zip.NewWriter(&buf)
		w, _ := archive.Create("package.nuspec")
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
		<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
			<metadata>
				<id>` + id + `</id>
				<version>` + version + `</version>
				<authors>` + packageAuthors + `</authors>
				<description>` + packageDescription + `</description>
				<dependencies>
					<group targetFramework=".NETStandard2.0">
						<dependency id="Microsoft.CSharp" version="4.5.0" />
					</group>
				</dependencies>
			</metadata>
		</package>`))
		archive.Close()
		return &buf
	}

	pkg := createPackage(packageName, packageVersion)

	pkgBuf, err := packages_module.CreateHashedBufferFromReader(pkg)
	require.NoError(t, err, "Error creating hashed buffer from nupkg")
	defer pkgBuf.Close()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	require.NoError(t, err, "Error getting user by ID 2")

	t.Run("PackagesNugetNuspecCheck", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		pi := &packages_service.PackageInfo{
			Owner:       doer,
			PackageType: packages_model.TypeNuGet,
			Name:        packageName,
			Version:     packageVersion,
		}
		_, _, err := packages_service.CreatePackageAndAddFile(
			ctx,
			&packages_service.PackageCreationInfo{
				PackageInfo:      *pi,
				SemverCompatible: true,
				Creator:          doer,
				Metadata:         nil,
			},
			&packages_service.PackageFileCreationInfo{
				PackageFileInfo: packages_service.PackageFileInfo{
					Filename: strings.ToLower(fmt.Sprintf("%s.%s.nupkg", packageName, packageVersion)),
				},
				Creator: doer,
				Data:    pkgBuf,
				IsLead:  true,
			},
		)
		require.NoError(t, err, "Error creating package and adding file")

		require.NoError(t, doctor.PackagesNugetNuspecCheck(ctx, logger, true), "Doctor check failed")

		s, _, pf, err := packages_service.GetFileStreamByPackageNameAndVersion(
			ctx,
			&packages_service.PackageInfo{
				Owner:       doer,
				PackageType: packages_model.TypeNuGet,
				Name:        packageName,
				Version:     packageVersion,
			},
			&packages_service.PackageFileInfo{
				Filename: strings.ToLower(fmt.Sprintf("%s.nuspec", packageName)),
			},
		)

		require.NoError(t, err, "Error getting nuspec file stream by package name and version")
		defer s.Close()

		assert.Equal(t, fmt.Sprintf("%s.nuspec", packageName), pf.Name, "Not a nuspec")
	})
}
