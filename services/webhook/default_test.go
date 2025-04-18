// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"

	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/json"
	webhook_module "forgejo.org/modules/webhook"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGiteaPayload(t *testing.T) {
	dh := defaultHandler{
		forgejo: false,
	}
	hook := &webhook_model.Webhook{
		RepoID:      3,
		IsActive:    true,
		Type:        webhook_module.GITEA,
		URL:         "https://gitea.example.com/",
		Meta:        ``,
		HTTPMethod:  "POST",
		ContentType: webhook_model.ContentTypeJSON,
	}

	// Woodpecker expects the ref to be short on tag creation only
	// https://github.com/woodpecker-ci/woodpecker/blob/00ccec078cdced80cf309cd4da460a5041d7991a/server/forge/gitea/helper.go#L134
	// see https://codeberg.org/codeberg/community/issues/1556
	t.Run("Create", func(t *testing.T) {
		p := createTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventCreate,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(t.Context(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitea.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "test", body.Ref) // short ref
	})

	t.Run("Push", func(t *testing.T) {
		p := pushTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventPush,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(t.Context(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitea.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})

	t.Run("Delete", func(t *testing.T) {
		p := deleteTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventDelete,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(t.Context(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitea.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "test", body.Ref) // short ref
	})
}

func TestForgejoPayload(t *testing.T) {
	dh := defaultHandler{
		forgejo: true,
	}
	hook := &webhook_model.Webhook{
		RepoID:      3,
		IsActive:    true,
		Type:        webhook_module.FORGEJO,
		URL:         "https://forgejo.example.com/",
		Meta:        ``,
		HTTPMethod:  "POST",
		ContentType: webhook_model.ContentTypeJSON,
	}

	// always return the full ref for consistency
	t.Run("Create", func(t *testing.T) {
		p := createTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventCreate,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(t.Context(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://forgejo.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})

	t.Run("Push", func(t *testing.T) {
		p := pushTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventPush,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(t.Context(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://forgejo.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})

	t.Run("Delete", func(t *testing.T) {
		p := deleteTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventDelete,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(t.Context(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://forgejo.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})
}

func TestOpenProjectPayload(t *testing.T) {
	t.Run("PullRequest", func(t *testing.T) {
		p := pullRequestTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		// adapted from https://github.com/opf/openproject/blob/4c5c45fe995da0060902bc8dd5f1bf704d0b8737/modules/github_integration/lib/open_project/github_integration/services/upsert_pull_request.rb#L56
		j := jsoniter.Get(data, "pull_request")

		assert.Equal(t, 12, j.Get("id").MustBeValid().ToInt())
		assert.Equal(t, "user1", j.Get("user", "login").MustBeValid().ToString())
		assert.Equal(t, 12, j.Get("number").MustBeValid().ToInt())
		assert.Equal(t, "http://localhost:3000/test/repo/pulls/12", j.Get("html_url").MustBeValid().ToString())
		assert.Equal(t, jsoniter.NilValue, j.Get("updated_at").ValueType())
		assert.Empty(t, j.Get("state").MustBeValid().ToString())
		assert.Equal(t, "Fix bug", j.Get("title").MustBeValid().ToString())
		assert.Equal(t, "fixes bug #2", j.Get("body").MustBeValid().ToString())

		assert.Equal(t, "test/repo", j.Get("base", "repo", "full_name").MustBeValid().ToString())
		assert.Equal(t, "http://localhost:3000/test/repo", j.Get("base", "repo", "html_url").MustBeValid().ToString())

		assert.False(t, j.Get("draft").MustBeValid().ToBool())
		assert.Equal(t, jsoniter.NilValue, j.Get("merge_commit_sha").ValueType())
		assert.False(t, j.Get("merged").MustBeValid().ToBool())
		assert.Equal(t, jsoniter.NilValue, j.Get("merged_by").ValueType())
		assert.Equal(t, jsoniter.NilValue, j.Get("merged_at").ValueType())
		assert.Equal(t, 0, j.Get("comments").MustBeValid().ToInt())
		assert.Equal(t, 0, j.Get("review_comments").MustBeValid().ToInt())
		assert.Equal(t, 0, j.Get("additions").MustBeValid().ToInt())
		assert.Equal(t, 0, j.Get("deletions").MustBeValid().ToInt())
		assert.Equal(t, 0, j.Get("changed_files").MustBeValid().ToInt())
		// assert.Equal(t,"labels:", j.Get("labels").map { |values| extract_label_values(values) )
	})
}
