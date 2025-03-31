// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalMarkupRenderer(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	if !setting.Database.Type.IsSQLite3() {
		t.Skip()
		return
	}

	const repoURL = "user30/renderer"
	req := NewRequest(t, "GET", repoURL+"/src/branch/master/README.html")
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header()["Content-Type"][0])

	bs, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	doc := NewHTMLParser(t, bytes.NewBuffer(bs))
	div := doc.Find("div.file-view")
	data, err := div.Html()
	require.NoError(t, err)
	assert.Equal(t, "<div>\n\ttest external renderer\n</div>", strings.TrimSpace(data))
}
