// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package nuget

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	id                = "System.Forgejo"
	title             = "Package Title"
	language          = "Package Language"
	semver            = "1.0.1"
	authors           = "Forgejo Authors"
	owners            = "Package Owners"
	copyright         = "Package Copyright"
	projectURL        = "https://forgejo.org"
	licenseURL        = "https://forgejo.org/docs/latest/license/"
	iconURL           = "https://codeberg.org/forgejo/governance/raw/branch/main/branding/logo/forgejo.png"
	description       = "Package Description"
	releaseNotes      = "Package Release Notes"
	readme            = "Readme"
	tags              = "tag_1 tag_2 tag_3"
	minClientVersion  = "1.0.0.0"
	repositoryURL     = "https://codeberg.org/forgejo"
	targetFramework   = ".NETStandard2.1"
	dependencyID      = "System.Text.Json"
	dependencyVersion = "5.0.0"
)

const nuspecContent = `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata minClientVersion="` + minClientVersion + `">
    <id>` + id + `</id>
	<title>` + title + `</title>
	<language>` + language + `</language>
    <version>` + semver + `</version>
    <authors>` + authors + `</authors>
    <owners>` + owners + `</owners>
    <copyright>` + copyright + `</copyright>
    <developmentDependency>true</developmentDependency>
    <requireLicenseAcceptance>true</requireLicenseAcceptance>
    <projectUrl>` + projectURL + `</projectUrl>
    <licenseUrl>` + licenseURL + `</licenseUrl>
    <iconUrl>` + iconURL + `</iconUrl>
    <description>` + description + `</description>
    <releaseNotes>` + releaseNotes + `</releaseNotes>
    <repository url="` + repositoryURL + `" />
    <readme>README.md</readme>
    <tags>` + tags + `</tags>
    <dependencies>
      <group targetFramework="` + targetFramework + `">
        <dependency id="` + dependencyID + `" version="` + dependencyVersion + `" exclude="Build,Analyzers" />
      </group>
    </dependencies>
  </metadata>
</package>`

const symbolsNuspecContent = `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>` + id + `</id>
    <version>` + semver + `</version>
    <description>` + description + `</description>
    <packageTypes>
      <packageType name="SymbolsPackage" />
    </packageTypes>
    <dependencies>
      <group targetFramework="` + targetFramework + `" />
    </dependencies>
  </metadata>
</package>`

func TestParsePackageMetaData(t *testing.T) {
	createArchive := func(files map[string]string) []byte {
		var buf bytes.Buffer
		archive := zip.NewWriter(&buf)
		for name, content := range files {
			w, _ := archive.Create(name)
			w.Write([]byte(content))
		}
		archive.Close()
		return buf.Bytes()
	}

	t.Run("MissingNuspecFile", func(t *testing.T) {
		data := createArchive(map[string]string{"dummy.txt": ""})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		assert.Nil(t, np)
		require.ErrorIs(t, err, ErrMissingNuspecFile)
	})

	t.Run("MissingNuspecFileInRoot", func(t *testing.T) {
		data := createArchive(map[string]string{"sub/package.nuspec": ""})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		assert.Nil(t, np)
		require.ErrorIs(t, err, ErrMissingNuspecFile)
	})

	t.Run("InvalidNuspecFile", func(t *testing.T) {
		data := createArchive(map[string]string{"package.nuspec": ""})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		assert.Nil(t, np)
		require.Error(t, err)
	})

	t.Run("InvalidPackageId", func(t *testing.T) {
		data := createArchive(map[string]string{"package.nuspec": `<?xml version="1.0" encoding="utf-8"?>
		<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
		  <metadata></metadata>
		</package>`})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		assert.Nil(t, np)
		require.ErrorIs(t, err, ErrNuspecInvalidID)
	})

	t.Run("InvalidPackageVersion", func(t *testing.T) {
		data := createArchive(map[string]string{"package.nuspec": `<?xml version="1.0" encoding="utf-8"?>
		<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
		  <metadata>
			<id>` + id + `</id>
		  </metadata>
		</package>`})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		assert.Nil(t, np)
		require.ErrorIs(t, err, ErrNuspecInvalidVersion)
	})

	t.Run("MissingReadme", func(t *testing.T) {
		data := createArchive(map[string]string{"package.nuspec": nuspecContent})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		require.NoError(t, err)
		assert.NotNil(t, np)
		assert.Empty(t, np.Metadata.Readme)
	})

	t.Run("Dependency Package", func(t *testing.T) {
		data := createArchive(map[string]string{
			"package.nuspec": nuspecContent,
			"README.md":      readme,
		})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		require.NoError(t, err)
		assert.NotNil(t, np)
		assert.Equal(t, DependencyPackage, np.PackageType)

		assert.Equal(t, id, np.ID)
		assert.Equal(t, title, np.Metadata.Title)
		assert.Equal(t, language, np.Metadata.Language)
		assert.Equal(t, semver, np.Version)
		assert.Equal(t, authors, np.Metadata.Authors)
		assert.Equal(t, owners, np.Metadata.Owners)
		assert.Equal(t, copyright, np.Metadata.Copyright)
		assert.True(t, np.Metadata.DevelopmentDependency)
		assert.True(t, np.Metadata.RequireLicenseAcceptance)
		assert.Equal(t, projectURL, np.Metadata.ProjectURL)
		assert.Equal(t, licenseURL, np.Metadata.LicenseURL)
		assert.Equal(t, iconURL, np.Metadata.IconURL)
		assert.Equal(t, description, np.Metadata.Description)
		assert.Equal(t, releaseNotes, np.Metadata.ReleaseNotes)
		assert.Equal(t, readme, np.Metadata.Readme)
		assert.Equal(t, tags, np.Metadata.Tags)
		assert.Equal(t, minClientVersion, np.Metadata.MinClientVersion)
		assert.Equal(t, repositoryURL, np.Metadata.RepositoryURL)
		assert.Len(t, np.Metadata.Dependencies, 1)
		assert.Contains(t, np.Metadata.Dependencies, targetFramework)
		deps := np.Metadata.Dependencies[targetFramework]
		assert.Len(t, deps, 1)
		assert.Equal(t, dependencyID, deps[0].ID)
		assert.Equal(t, dependencyVersion, deps[0].Version)

		t.Run("NormalizedVersion", func(t *testing.T) {
			data := createArchive(map[string]string{"package.nuspec": `<?xml version="1.0" encoding="utf-8"?>
				<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
				  <metadata>
					<id>test</id>
					<version>1.04.5.2.5-rc.1+metadata</version>
				  </metadata>
				</package>`})

			np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
			require.NoError(t, err)
			assert.NotNil(t, np)
			assert.Equal(t, "1.4.5.2-rc.1", np.Version)
		})
	})

	t.Run("Symbols Package", func(t *testing.T) {
		data := createArchive(map[string]string{"package.nuspec": symbolsNuspecContent})

		np, err := ParsePackageMetaData(bytes.NewReader(data), int64(len(data)))
		require.NoError(t, err)
		assert.NotNil(t, np)
		assert.Equal(t, SymbolsPackage, np.PackageType)

		assert.Equal(t, id, np.ID)
		assert.Equal(t, semver, np.Version)
		assert.Equal(t, description, np.Metadata.Description)
		assert.Empty(t, np.Metadata.Dependencies)
	})
}
