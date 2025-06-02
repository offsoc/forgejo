// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestWebfinger(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	appURL, _ := url.Parse(setting.AppURL)

	type webfingerLink struct {
		Rel        string            `json:"rel,omitempty"`
		Type       string            `json:"type,omitempty"`
		Href       string            `json:"href,omitempty"`
		Titles     map[string]string `json:"titles,omitempty"`
		Properties map[string]any    `json:"properties,omitempty"`
	}

	type webfingerJRD struct {
		Subject    string           `json:"subject,omitempty"`
		Aliases    []string         `json:"aliases,omitempty"`
		Properties map[string]any   `json:"properties,omitempty"`
		Links      []*webfingerLink `json:"links,omitempty"`
	}

	session := loginUser(t, "user1")

	req := NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", user.LowerName, appURL.Host))
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, "application/jrd+json", resp.Header().Get("Content-Type"))

	var jrd webfingerJRD
	DecodeJSON(t, resp, &jrd)
	assert.Equal(t, "acct:user2@"+appURL.Host, jrd.Subject)
	assert.ElementsMatch(t, []string{user.HTMLURL(), appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(user.ID)}, jrd.Aliases)

	instanceReq := NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:ghost@%s", appURL.Host))
	instanceResp := MakeRequest(t, instanceReq, http.StatusOK)
	assert.Equal(t, "application/jrd+json", instanceResp.Header().Get("Content-Type"))

	var instanceActor webfingerJRD
	DecodeJSON(t, instanceResp, &instanceActor)
	assert.Equal(t, "acct:ghost@"+appURL.Host, instanceActor.Subject)
	assert.ElementsMatch(t, []string{appURL.String() + "api/v1/activitypub/actor"}, instanceActor.Aliases)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", user.LowerName, "unknown.host"))
	MakeRequest(t, req, http.StatusBadRequest)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", "user31", appURL.Host))
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", "user31", appURL.Host))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=mailto:%s", user.Email))
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=https://%s/%s/", appURL.Host, user.Name))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=https://%s/%s", appURL.Host, user.Name))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=http://%s/%s/foo", appURL.Host, user.Name))
	session.MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=http://%s", appURL.Host))
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=http://%s/%s/foo", "example.com", user.Name))
	MakeRequest(t, req, http.StatusBadRequest)
}
