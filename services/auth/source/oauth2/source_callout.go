// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

// Callout redirects request/response pair to authenticate against the provider
func (source *Source) Callout(request *http.Request, response http.ResponseWriter, codeChallengeS256 string, acrValues []string) error {
	// not sure if goth is thread safe (?) when using multiple providers
	request.Header.Set(ProviderHeaderKey, source.authSource.Name)

	var querySuffix string
	if codeChallengeS256 != "" {
		querySuffix = "&" + url.Values{
			"code_challenge_method": []string{"S256"},
			"code_challenge":        []string{codeChallengeS256},
		}.Encode()
	}

	var acrQuerySuffix string
	if len(acrValues) != 0 {
		acrQuerySuffix = "&" + url.Values{
			"acr_values": []string{strings.Join(acrValues, " ")},
		}.Encode()
	}

	// don't use the default gothic begin handler to prevent issues when some error occurs
	// normally the gothic library will write some custom stuff to the response instead of our own nice error page
	// gothic.BeginAuthHandler(response, request)

	gothRWMutex.RLock()
	defer gothRWMutex.RUnlock()

	url, err := gothic.GetAuthURL(response, request)
	if err == nil {
		// hacky way to set the code_challenge, but no better way until
		// https://github.com/markbates/goth/issues/516 is resolved
		http.Redirect(response, request, url+querySuffix+acrQuerySuffix, http.StatusTemporaryRedirect)
	}
	return err
}

// Callback handles OAuth callback, resolve to a goth user and send back to original url
// this will trigger a new authentication request, but because we save it in the session we can use that
func (source *Source) Callback(request *http.Request, response http.ResponseWriter, codeVerifier string) (goth.User, error) {
	// not sure if goth is thread safe (?) when using multiple providers
	request.Header.Set(ProviderHeaderKey, source.authSource.Name)

	if codeVerifier != "" {
		// hacky way to set the code_verifier...
		// Will be picked up inside CompleteUserAuth: params := req.URL.Query()
		// https://github.com/markbates/goth/pull/474/files
		request = request.Clone(request.Context())
		q := request.URL.Query()
		q.Add("code_verifier", codeVerifier)
		request.URL.RawQuery = q.Encode()
	}

	gothRWMutex.RLock()
	defer gothRWMutex.RUnlock()

	user, err := gothic.CompleteUserAuth(response, request)
	if err != nil {
		return user, err
	}

	return user, nil
}
