// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"fmt"
	"testing"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/json"
	api "code.gitea.io/gitea/modules/structs"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagistPayload(t *testing.T) {
	payloads := []api.Payloader{
		createTestPayload(),
		deleteTestPayload(),
		forkTestPayload(),
		pushTestPayload(),
		issueTestPayload(),
		issueCommentTestPayload(),
		pullRequestCommentTestPayload(),
		pullRequestTestPayload(),
		repositoryTestPayload(),
		packageTestPayload(),
		wikiTestPayload(),
		pullReleaseTestPayload(),
	}

	for _, payloader := range payloads {
		t.Run(fmt.Sprintf("%T", payloader), func(t *testing.T) {
			data, err := payloader.JSONPayload()
			require.NoError(t, err)

			hook := &webhook_model.Webhook{
				RepoID:     3,
				IsActive:   true,
				Type:       webhook_module.PACKAGIST,
				URL:        "https://packagist.org/api/update-package?username=THEUSERNAME&apiToken=TOPSECRETAPITOKEN",
				Meta:       `{"package_url":"https://packagist.org/packages/example"}`,
				HTTPMethod: "POST",
			}
			task := &webhook_model.HookTask{
				HookID:         hook.ID,
				EventType:      webhook_module.HookEventPush,
				PayloadContent: string(data),
				PayloadVersion: 2,
			}

			req, reqBody, err := packagistHandler{}.NewRequest(t.Context(), hook, task)
			require.NotNil(t, req)
			require.NotNil(t, reqBody)
			require.NoError(t, err)

			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://packagist.org/api/update-package?username=THEUSERNAME&apiToken=TOPSECRETAPITOKEN", req.URL.String())
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
			var body PackagistPayload
			err = json.NewDecoder(req.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, "https://packagist.org/packages/example", body.PackagistRepository.URL)
		})
	}
}
