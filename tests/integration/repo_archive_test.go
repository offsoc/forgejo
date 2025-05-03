// Copyright 2024 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/routers/web"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoDownloadArchive(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.EnableGzip, true)()
	defer test.MockVariableValue(&web.GzipMinSize, 10)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	req := NewRequest(t, "GET", "/user2/repo1/archive/master.zip")
	req.Header.Set("Accept-Encoding", "gzip")
	resp := MakeRequest(t, req, http.StatusOK)
	bs, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Empty(t, resp.Header().Get("Content-Encoding"))
	assert.Len(t, bs, 320)

	// Verify that unrecognized archive type returns 404
	req = NewRequest(t, "GET", "/user2/repo1/archive/master.invalid")
	MakeRequest(t, req, http.StatusNotFound)
}

func TestRepoDownloadArchiveSubdir(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		defer test.MockVariableValue(&setting.EnableGzip, true)()
		defer test.MockVariableValue(&web.GzipMinSize, 10)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

		// Create a subdirectory
		err := createOrReplaceFileInBranch(user, repo, "subdir/test.txt", "master", "Test")
		require.NoError(t, err)

		t.Run("Frontend", func(t *testing.T) {
			resp := MakeRequest(t, NewRequestf(t, "GET", "/%s/src/branch/master/subdir", repo.FullName()), http.StatusOK)
			page := NewHTMLParser(t, resp.Body)

			page.AssertElement(t, fmt.Sprintf(".folder-actions a.archive-link[href='/%s/archive/master:subdir.zip'][type='application/zip']", repo.FullName()), true)
			page.AssertElement(t, fmt.Sprintf(".folder-actions a.archive-link[href='/%s/archive/master:subdir.tar.gz'][type='application/gzip']", repo.FullName()), true)
		})

		t.Run("Backend", func(t *testing.T) {
			resp := MakeRequest(t, NewRequestf(t, "GET", "/%s/archive/master:subdir.tar.gz", repo.FullName()), http.StatusOK)

			uncompressedStream, err := gzip.NewReader(resp.Body)
			require.NoError(t, err)

			tarReader := tar.NewReader(uncompressedStream)

			header, err := tarReader.Next()
			require.NoError(t, err)
			assert.Equal(t, tar.TypeDir, int32(header.Typeflag))
			assert.Equal(t, fmt.Sprintf("%s/", repo.Name), header.Name)

			header, err = tarReader.Next()
			require.NoError(t, err)
			assert.Equal(t, tar.TypeReg, int32(header.Typeflag))
			assert.Equal(t, fmt.Sprintf("%s/test.txt", repo.Name), header.Name)

			_, err = tarReader.Next()
			assert.Equal(t, io.EOF, err)
		})
	})
}
