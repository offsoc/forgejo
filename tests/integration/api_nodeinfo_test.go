// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestNodeinfo(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/nodeinfo")
	resp := MakeRequest(t, req, http.StatusOK)
	VerifyJSONSchema(t, resp, "nodeinfo_2.1.json")

	var nodeinfo api.NodeInfo
	DecodeJSON(t, resp, &nodeinfo)
	assert.True(t, nodeinfo.OpenRegistrations)
	assert.Equal(t, "forgejo", nodeinfo.Software.Name)
	assert.Equal(t, 29, nodeinfo.Usage.Users.Total)
	assert.Equal(t, 22, nodeinfo.Usage.LocalPosts)
	assert.Equal(t, 4, nodeinfo.Usage.LocalComments)
}
