// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	packages_cleanup "forgejo.org/services/packages/cleanup"
	packages_container "forgejo.org/services/packages/container"
	"forgejo.org/tests"

	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagesContainerCleanupSHA256(t *testing.T) {
	defer tests.PrepareTestEnv(t, 1)()
	defer test.MockVariableValue(&setting.Packages.Storage.Type, setting.LocalStorageType)()
	defer test.MockVariableValue(&packages_container.SHA256BatchSize, 1)()

	ctx := db.DefaultContext

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	cleanupAndCheckLogs := func(t *testing.T, expected ...string) {
		t.Helper()
		logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
		logChecker.Filter(expected...)
		logChecker.StopMark(packages_container.SHA256LogFinish)
		defer cleanup()

		require.NoError(t, packages_cleanup.CleanupExpiredData(ctx, -1*time.Hour))

		logFiltered, logStopped := logChecker.Check(5 * time.Second)
		assert.True(t, logStopped)
		filtered := make([]bool, 0, len(expected))
		for range expected {
			filtered = append(filtered, true)
		}
		assert.Equal(t, filtered, logFiltered, expected)
	}

	userToken := ""

	t.Run("Authenticate", func(t *testing.T) {
		type TokenResponse struct {
			Token string `json:"token"`
		}

		authenticate := []string{`Bearer realm="` + setting.AppURL + `v2/token",service="container_registry",scope="*"`}

		t.Run("User", func(t *testing.T) {
			req := NewRequest(t, "GET", fmt.Sprintf("%sv2", setting.AppURL))
			resp := MakeRequest(t, req, http.StatusUnauthorized)

			assert.ElementsMatch(t, authenticate, resp.Header().Values("WWW-Authenticate"))

			req = NewRequest(t, "GET", fmt.Sprintf("%sv2/token", setting.AppURL)).
				AddBasicAuth(user.Name)
			resp = MakeRequest(t, req, http.StatusOK)

			tokenResponse := &TokenResponse{}
			DecodeJSON(t, resp, &tokenResponse)

			assert.NotEmpty(t, tokenResponse.Token)

			userToken = fmt.Sprintf("Bearer %s", tokenResponse.Token)

			req = NewRequest(t, "GET", fmt.Sprintf("%sv2", setting.AppURL)).
				AddTokenAuth(userToken)
			MakeRequest(t, req, http.StatusOK)
		})
	})

	image := "test"
	multiTag := "multi"

	url := fmt.Sprintf("%sv2/%s/%s", setting.AppURL, user.Name, image)

	blobDigest := "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	sha256ManifestDigest := "sha256:4305f5f5572b9a426b88909b036e52ee3cf3d7b9c1b01fac840e90747f56623d"
	indexManifestDigest := "sha256:b992f98104ab25f60d78368a674ce6f6a49741f4e32729e8496067ed06174e9b"

	uploadSHA256Version := func(t *testing.T) {
		t.Helper()

		blobContent, _ := base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)

		req := NewRequestWithBody(t, "POST", fmt.Sprintf("%s/blobs/uploads?digest=%s", url, blobDigest), bytes.NewReader(blobContent)).
			AddTokenAuth(userToken)
		resp := MakeRequest(t, req, http.StatusCreated)

		assert.Equal(t, fmt.Sprintf("/v2/%s/%s/blobs/%s", user.Name, image, blobDigest), resp.Header().Get("Location"))
		assert.Equal(t, blobDigest, resp.Header().Get("Docker-Content-Digest"))

		configDigest := "sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d"
		configContent := `{"architecture":"amd64","config":{"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/true"],"ArgsEscaped":true,"Image":"sha256:9bd8b88dc68b80cffe126cc820e4b52c6e558eb3b37680bfee8e5f3ed7b8c257"},"container":"b89fe92a887d55c0961f02bdfbfd8ac3ddf66167db374770d2d9e9fab3311510","container_config":{"Hostname":"b89fe92a887d","Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/true\"]"],"ArgsEscaped":true,"Image":"sha256:9bd8b88dc68b80cffe126cc820e4b52c6e558eb3b37680bfee8e5f3ed7b8c257"},"created":"2022-01-01T00:00:00.000000000Z","docker_version":"20.10.12","history":[{"created":"2022-01-01T00:00:00.000000000Z","created_by":"/bin/sh -c #(nop) COPY file:0e7589b0c800daaf6fa460d2677101e4676dd9491980210cb345480e513f3602 in /true "},{"created":"2022-01-01T00:00:00.000000001Z","created_by":"/bin/sh -c #(nop)  CMD [\"/true\"]","empty_layer":true}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:0ff3b91bdf21ecdf2f2f3d4372c2098a14dbe06cd678e8f0a85fd4902d00e2e2"]}}`

		req = NewRequestWithBody(t, "POST", fmt.Sprintf("%s/blobs/uploads?digest=%s", url, configDigest), strings.NewReader(configContent)).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusCreated)

		sha256ManifestContent := `{"schemaVersion":2,"mediaType":"` + oci.MediaTypeImageManifest + `","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d","size":1069},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","digest":"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4","size":32}]}`
		req = NewRequestWithBody(t, "PUT", fmt.Sprintf("%s/manifests/%s", url, sha256ManifestDigest), strings.NewReader(sha256ManifestContent)).
			AddTokenAuth(userToken).
			SetHeader("Content-Type", oci.MediaTypeImageManifest)
		resp = MakeRequest(t, req, http.StatusCreated)

		assert.Equal(t, sha256ManifestDigest, resp.Header().Get("Docker-Content-Digest"))

		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, sha256ManifestDigest)).
			AddTokenAuth(userToken)
		resp = MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, fmt.Sprintf("%d", len(sha256ManifestContent)), resp.Header().Get("Content-Length"))
		assert.Equal(t, sha256ManifestDigest, resp.Header().Get("Docker-Content-Digest"))
	}

	uploadIndexManifest := func(t *testing.T) {
		indexManifestContent := `{"schemaVersion":2,"mediaType":"` + oci.MediaTypeImageIndex + `","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","digest":"` + sha256ManifestDigest + `","platform":{"os":"linux","architecture":"arm","variant":"v7"}}]}`
		req := NewRequestWithBody(t, "PUT", fmt.Sprintf("%s/manifests/%s", url, multiTag), strings.NewReader(indexManifestContent)).
			AddTokenAuth(userToken).
			SetHeader("Content-Type", oci.MediaTypeImageIndex)
		resp := MakeRequest(t, req, http.StatusCreated)

		assert.Equal(t, indexManifestDigest, resp.Header().Get("Docker-Content-Digest"))
	}

	assertImageExists := func(t *testing.T, manifestDigest, blobDigest string) {
		req := NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/blobs/%s", url, blobDigest)).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusOK)
	}

	assertImageNotExists := func(t *testing.T, manifestDigest, blobDigest string) {
		req := NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusNotFound)

		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/blobs/%s", url, blobDigest)).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusNotFound)
	}

	assertImageDeleted := func(t *testing.T, image, manifestDigest, blobDigest string, cleanup func()) {
		t.Helper()
		packageVersion := unittest.AssertExistsAndLoadBean(t, &packages_model.PackageVersion{Version: manifestDigest})
		packageFile := unittest.AssertExistsAndLoadBean(t, &packages_model.PackageFile{VersionID: packageVersion.ID})
		unittest.AssertExistsAndLoadBean(t, &packages_model.PackageProperty{RefID: packageFile.ID, RefType: packages_model.PropertyTypeVersion})
		packageBlob := unittest.AssertExistsAndLoadBean(t, &packages_model.PackageBlob{ID: packageFile.BlobID})
		contentStore := packages_module.NewContentStore()
		require.NoError(t, contentStore.Has(packages_module.BlobHash256Key(packageBlob.HashSHA256)))

		assertImageExists(t, manifestDigest, blobDigest)

		cleanup()

		assertImageNotExists(t, manifestDigest, blobDigest)

		unittest.AssertNotExistsBean(t, &packages_model.PackageVersion{Version: manifestDigest})
		unittest.AssertNotExistsBean(t, &packages_model.PackageFile{VersionID: packageVersion.ID})
		unittest.AssertNotExistsBean(t, &packages_model.PackageProperty{RefID: packageFile.ID, RefType: packages_model.PropertyTypeVersion})
		unittest.AssertNotExistsBean(t, &packages_model.PackageBlob{ID: packageFile.BlobID})
		assert.Error(t, contentStore.Has(packages_module.BlobHash256Key(packageBlob.HashSHA256)))
	}

	assertImageAndPackageDeleted := func(t *testing.T, image, manifestDigest, blobDigest string, cleanup func()) {
		t.Helper()
		unittest.AssertExistsAndLoadBean(t, &packages_model.Package{Name: image})
		assertImageDeleted(t, image, manifestDigest, blobDigest, cleanup)
		unittest.AssertNotExistsBean(t, &packages_model.Package{Name: image})
	}

	t.Run("Nothing to look at", func(t *testing.T) {
		cleanupAndCheckLogs(t, "There are no container images with a version matching sha256:*")
	})

	uploadSHA256Version(t)

	t.Run("Dangling image found", func(t *testing.T) {
		assertImageAndPackageDeleted(t, image, sha256ManifestDigest, blobDigest, func() {
			cleanupAndCheckLogs(t,
				"Removing 3 entries from `package_file` and `package_property`",
				"Removing 1 entries from `package_version` and `package_property`",
			)
		})
	})

	uploadSHA256Version(t)
	uploadIndexManifest(t)

	t.Run("Corrupted index manifest metadata is ignored", func(t *testing.T) {
		assertImageExists(t, sha256ManifestDigest, blobDigest)
		_, err := db.GetEngine(ctx).Table("package_version").Where("version = ?", multiTag).Update(&packages_model.PackageVersion{MetadataJSON: `corrupted "manifests":[{ bad`})
		require.NoError(t, err)

		// do not expect the package to be deleted because it contains
		// corrupted metadata that prevents that from happening
		assertImageDeleted(t, image, sha256ManifestDigest, blobDigest, func() {
			cleanupAndCheckLogs(t,
				"Removing 3 entries from `package_file` and `package_property`",
				"Removing 1 entries from `package_version` and `package_property`",
				"is not a JSON string containing valid metadata",
			)
		})
	})

	uploadSHA256Version(t)
	uploadIndexManifest(t)

	t.Run("Image found but referenced", func(t *testing.T) {
		assertImageExists(t, sha256ManifestDigest, blobDigest)
		cleanupAndCheckLogs(t,
			"All container images with a version matching sha256:* are referenced by an index manifest",
		)
		assertImageExists(t, sha256ManifestDigest, blobDigest)
	})
}
