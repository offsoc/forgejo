// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/web/middleware"
	"forgejo.org/services/actions"
	"forgejo.org/services/auth/source/oauth2"
)

// Ensure the struct implements the interface.
var (
	_ Method = &OAuth2{}
)

// grantAdditionalScopes returns valid scopes coming from grant
func grantAdditionalScopes(grantScopes string) string {
	// scopes_supported from templates/user/auth/oidc_wellknown.tmpl
	scopesSupported := []string{
		"openid",
		"profile",
		"email",
		"groups",
	}

	var apiTokenScopes []string
	for _, apiTokenScope := range strings.Split(grantScopes, " ") {
		if slices.Index(scopesSupported, apiTokenScope) == -1 {
			apiTokenScopes = append(apiTokenScopes, apiTokenScope)
		}
	}

	if len(apiTokenScopes) == 0 {
		return ""
	}

	var additionalGrantScopes []string
	allScopes := auth_model.AccessTokenScope("all")

	for _, apiTokenScope := range apiTokenScopes {
		grantScope := auth_model.AccessTokenScope(apiTokenScope)
		if ok, _ := allScopes.HasScope(grantScope); ok {
			additionalGrantScopes = append(additionalGrantScopes, apiTokenScope)
		} else if apiTokenScope == "public-only" {
			additionalGrantScopes = append(additionalGrantScopes, apiTokenScope)
		}
	}
	if len(additionalGrantScopes) > 0 {
		return strings.Join(additionalGrantScopes, ",")
	}

	return ""
}

// CheckOAuthAccessToken returns uid of user from oauth token
// + non default openid scopes requested
func CheckOAuthAccessToken(ctx context.Context, accessToken string) (int64, string) {
	if !setting.OAuth2.Enabled {
		return 0, ""
	}
	// JWT tokens require a "."
	if !strings.Contains(accessToken, ".") {
		return 0, ""
	}
	token, err := oauth2.ParseToken(accessToken, oauth2.DefaultSigningKey)
	if err != nil {
		log.Trace("oauth2.ParseToken: %v", err)
		return 0, ""
	}
	var grant *auth_model.OAuth2Grant
	if grant, err = auth_model.GetOAuth2GrantByID(ctx, token.GrantID); err != nil || grant == nil {
		return 0, ""
	}
	if token.Type != oauth2.TypeAccessToken {
		return 0, ""
	}
	if token.ExpiresAt.Before(time.Now()) || token.IssuedAt.After(time.Now()) {
		return 0, ""
	}
	grantScopes := grantAdditionalScopes(grant.Scope)
	return grant.UserID, grantScopes
}

// CheckTaskIsRunning verifies that the TaskID corresponds to a running task
func CheckTaskIsRunning(ctx context.Context, taskID int64) bool {
	// Verify the task exists
	task, err := actions_model.GetTaskByID(ctx, taskID)
	if err != nil {
		return false
	}

	// Verify that it's running
	return task.Status == actions_model.StatusRunning
}

// OAuth2 implements the Auth interface and authenticates requests
// (API requests only) by looking for an OAuth token in query parameters or the
// "Authorization" header.
type OAuth2 struct{}

// Name represents the name of auth method
func (o *OAuth2) Name() string {
	return "oauth2"
}

// parseToken returns the token from request, and a boolean value
// representing whether the token exists or not
func parseToken(req *http.Request) (string, bool) {
	_ = req.ParseForm()

	// check header token
	if auHead := req.Header.Get("Authorization"); auHead != "" {
		auths := strings.Fields(auHead)
		if len(auths) == 2 && (auths[0] == "token" || strings.ToLower(auths[0]) == "bearer") {
			return auths[1], true
		}
	}
	return "", false
}

// userIDFromToken returns the user id corresponding to the OAuth token.
// It will set 'IsApiToken' to true if the token is an API token and
// set 'ApiTokenScope' to the scope of the access token
func (o *OAuth2) userIDFromToken(ctx context.Context, tokenSHA string, store DataStore) int64 {
	// Let's see if token is valid.
	if strings.Contains(tokenSHA, ".") {
		// First attempt to decode an actions JWT, returning the actions user
		if taskID, err := actions.TokenToTaskID(tokenSHA); err == nil {
			if CheckTaskIsRunning(ctx, taskID) {
				store.GetData()["IsActionsToken"] = true
				store.GetData()["ActionsTaskID"] = taskID
				return user_model.ActionsUserID
			}
		}

		// Otherwise, check if this is an OAuth access token
		uid, grantScopes := CheckOAuthAccessToken(ctx, tokenSHA)
		if uid != 0 {
			store.GetData()["IsApiToken"] = true
			if grantScopes != "" {
				store.GetData()["ApiTokenScope"] = auth_model.AccessTokenScope(grantScopes)
			} else {
				store.GetData()["ApiTokenScope"] = auth_model.AccessTokenScopeAll // fallback to all
			}
		}
		return uid
	}
	t, err := auth_model.GetAccessTokenBySHA(ctx, tokenSHA)
	if err != nil {
		if auth_model.IsErrAccessTokenNotExist(err) {
			// check task token
			task, err := actions_model.GetRunningTaskByToken(ctx, tokenSHA)
			if err == nil && task != nil {
				log.Trace("Basic Authorization: Valid AccessToken for task[%d]", task.ID)

				store.GetData()["IsActionsToken"] = true
				store.GetData()["ActionsTaskID"] = task.ID

				return user_model.ActionsUserID
			}
		} else if !auth_model.IsErrAccessTokenNotExist(err) && !auth_model.IsErrAccessTokenEmpty(err) {
			log.Error("GetAccessTokenBySHA: %v", err)
		}
		return 0
	}
	t.UpdatedUnix = timeutil.TimeStampNow()
	if err = auth_model.UpdateAccessToken(ctx, t); err != nil {
		log.Error("UpdateAccessToken: %v", err)
	}
	store.GetData()["IsApiToken"] = true
	store.GetData()["ApiTokenScope"] = t.Scope
	return t.UID
}

// Verify extracts the user ID from the OAuth token in the query parameters
// or the "Authorization" header and returns the corresponding user object for that ID.
// If verification is successful returns an existing user object.
// Returns nil if verification fails.
func (o *OAuth2) Verify(req *http.Request, w http.ResponseWriter, store DataStore, sess SessionStore) (*user_model.User, error) {
	// These paths are not API paths, but we still want to check for tokens because they maybe in the API returned URLs
	if !middleware.IsAPIPath(req) && !isAttachmentDownload(req) && !isAuthenticatedTokenRequest(req) &&
		!isGitRawOrAttachPath(req) && !isArchivePath(req) {
		return nil, nil
	}

	token, ok := parseToken(req)
	if !ok {
		return nil, nil
	}

	id := o.userIDFromToken(req.Context(), token, store)

	if id <= 0 && id != -2 { // -2 means actions, so we need to allow it.
		return nil, user_model.ErrUserNotExist{}
	}
	log.Trace("OAuth2 Authorization: Found token for user[%d]", id)

	user, err := user_model.GetPossibleUserByID(req.Context(), id)
	if err != nil {
		if !user_model.IsErrUserNotExist(err) {
			log.Error("GetUserByName: %v", err)
		}
		return nil, err
	}

	log.Trace("OAuth2 Authorization: Logged in user %-v", user)
	return user, nil
}

func isAuthenticatedTokenRequest(req *http.Request) bool {
	switch req.URL.Path {
	case "/login/oauth/userinfo":
		fallthrough
	case "/login/oauth/introspect":
		return true
	}
	return false
}
