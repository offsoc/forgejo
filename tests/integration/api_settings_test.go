// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIExposedSettings(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	ui := new(api.GeneralUISettings)
	req := NewRequest(t, "GET", "/api/v1/settings/ui")
	resp := MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &ui)
	assert.Len(t, ui.AllowedReactions, len(setting.UI.Reactions))
	assert.ElementsMatch(t, setting.UI.Reactions, ui.AllowedReactions)

	apiSettings := new(api.GeneralAPISettings)
	req = NewRequest(t, "GET", "/api/v1/settings/api")
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &apiSettings)
	assert.Equal(t, &api.GeneralAPISettings{
		MaxResponseItems:       setting.API.MaxResponseItems,
		DefaultPagingNum:       setting.API.DefaultPagingNum,
		DefaultGitTreesPerPage: setting.API.DefaultGitTreesPerPage,
		DefaultMaxBlobSize:     setting.API.DefaultMaxBlobSize,
	}, apiSettings)

	repo := new(api.GeneralRepoSettings)
	req = NewRequest(t, "GET", "/api/v1/settings/repository")
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &repo)
	assert.Equal(t, &api.GeneralRepoSettings{
		MirrorsDisabled:      !setting.Mirror.Enabled,
		HTTPGitDisabled:      setting.Repository.DisableHTTPGit,
		MigrationsDisabled:   setting.Repository.DisableMigrations,
		TimeTrackingDisabled: false,
		LFSDisabled:          !setting.LFS.StartServer,
	}, repo)

	attachment := new(api.GeneralAttachmentSettings)
	req = NewRequest(t, "GET", "/api/v1/settings/attachment")
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &attachment)
	assert.Equal(t, &api.GeneralAttachmentSettings{
		Enabled:      setting.Attachment.Enabled,
		AllowedTypes: setting.Attachment.AllowedTypes,
		MaxFiles:     setting.Attachment.MaxFiles,
		MaxSize:      setting.Attachment.MaxSize,
	}, attachment)
}
