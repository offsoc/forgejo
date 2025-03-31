// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/test"
	remote_service "forgejo.org/services/remote"
	"forgejo.org/tests"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemote_MaybePromoteUserSuccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	//
	// OAuth2 authentication source GitLab
	//
	gitlabName := "gitlab"
	_ = addAuthSource(t, authSourcePayloadGitLabCustom(gitlabName))
	//
	// Remote authentication source matching the GitLab authentication source
	//
	remoteName := "remote"
	remote := createRemoteAuthSource(t, remoteName, "http://mygitlab.eu", gitlabName)

	//
	// Create a user as if it had previously been created by the remote
	// authentication source.
	//
	gitlabUserID := "5678"
	gitlabEmail := "gitlabuser@example.com"
	userBeforeSignIn := &user_model.User{
		Name:        "gitlabuser",
		Type:        user_model.UserTypeRemoteUser,
		LoginType:   auth_model.Remote,
		LoginSource: remote.ID,
		LoginName:   gitlabUserID,
	}
	defer createUser(t.Context(), t, userBeforeSignIn)()

	//
	// A request for user information sent to Goth will return a
	// goth.User exactly matching the user created above.
	//
	defer mockCompleteUserAuth(func(res http.ResponseWriter, req *http.Request) (goth.User, error) {
		return goth.User{
			Provider: gitlabName,
			UserID:   gitlabUserID,
			Email:    gitlabEmail,
		}, nil
	})()
	req := NewRequest(t, "GET", fmt.Sprintf("/user/oauth2/%s/callback?code=XYZ&state=XYZ", gitlabName))
	resp := MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/", test.RedirectURL(resp))
	userAfterSignIn := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userBeforeSignIn.ID})

	// both are about the same user
	assert.Equal(t, userBeforeSignIn.ID, userAfterSignIn.ID)
	// the login time was updated, proof the login succeeded
	assert.Greater(t, userAfterSignIn.LastLoginUnix, userBeforeSignIn.LastLoginUnix)
	// the login type was promoted from Remote to OAuth2
	assert.Equal(t, auth_model.Remote, userBeforeSignIn.LoginType)
	assert.Equal(t, auth_model.OAuth2, userAfterSignIn.LoginType)
	// the OAuth2 email was used to set the missing user email
	assert.Empty(t, userBeforeSignIn.Email)
	assert.Equal(t, gitlabEmail, userAfterSignIn.Email)
}

func TestRemote_MaybePromoteUserFail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	ctx := t.Context()
	//
	// OAuth2 authentication source GitLab
	//
	gitlabName := "gitlab"
	gitlabSource := addAuthSource(t, authSourcePayloadGitLabCustom(gitlabName))
	//
	// Remote authentication source matching the GitLab authentication source
	//
	remoteName := "remote"
	remoteSource := createRemoteAuthSource(t, remoteName, "http://mygitlab.eu", gitlabName)

	{
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, &auth_model.Source{}, "", "")
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonNotAuth2, reason)
	}

	{
		remoteSource.Type = auth_model.OAuth2
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, remoteSource, "", "")
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonBadAuth2, reason)
		remoteSource.Type = auth_model.Remote
	}

	{
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, gitlabSource, "unknownloginname", "")
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonLoginNameNotExists, reason)
	}

	{
		remoteUserID := "844"
		remoteUser := &user_model.User{
			Name:        "withmailuser",
			Type:        user_model.UserTypeRemoteUser,
			LoginType:   auth_model.Remote,
			LoginSource: remoteSource.ID,
			LoginName:   remoteUserID,
			Email:       "some@example.com",
		}
		defer createUser(t.Context(), t, remoteUser)()
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, gitlabSource, remoteUserID, "")
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonEmailIsSet, reason)
	}

	{
		remoteUserID := "7464"
		nonexistentloginsource := int64(4344)
		remoteUser := &user_model.User{
			Name:        "badsourceuser",
			Type:        user_model.UserTypeRemoteUser,
			LoginType:   auth_model.Remote,
			LoginSource: nonexistentloginsource,
			LoginName:   remoteUserID,
		}
		defer createUser(t.Context(), t, remoteUser)()
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, gitlabSource, remoteUserID, "")
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonNoSource, reason)
	}

	{
		remoteUserID := "33335678"
		remoteUser := &user_model.User{
			Name:        "badremoteuser",
			Type:        user_model.UserTypeRemoteUser,
			LoginType:   auth_model.Remote,
			LoginSource: gitlabSource.ID,
			LoginName:   remoteUserID,
		}
		defer createUser(t.Context(), t, remoteUser)()
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, gitlabSource, remoteUserID, "")
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonSourceWrongType, reason)
	}

	{
		unrelatedName := "unrelated"
		unrelatedSource := addAuthSource(t, authSourcePayloadGitHubCustom(unrelatedName))
		assert.NotNil(t, unrelatedSource)

		remoteUserID := "488484"
		remoteEmail := "4848484@example.com"
		remoteUser := &user_model.User{
			Name:        "unrelateduser",
			Type:        user_model.UserTypeRemoteUser,
			LoginType:   auth_model.Remote,
			LoginSource: remoteSource.ID,
			LoginName:   remoteUserID,
		}
		defer createUser(t.Context(), t, remoteUser)()
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, unrelatedSource, remoteUserID, remoteEmail)
		require.NoError(t, err)
		assert.False(t, promoted)
		assert.Equal(t, remote_service.ReasonNoMatch, reason)
	}

	{
		remoteUserID := "5678"
		remoteEmail := "gitlabuser@example.com"
		remoteUser := &user_model.User{
			Name:        "remoteuser",
			Type:        user_model.UserTypeRemoteUser,
			LoginType:   auth_model.Remote,
			LoginSource: remoteSource.ID,
			LoginName:   remoteUserID,
		}
		defer createUser(t.Context(), t, remoteUser)()
		promoted, reason, err := remote_service.MaybePromoteRemoteUser(ctx, gitlabSource, remoteUserID, remoteEmail)
		require.NoError(t, err)
		assert.True(t, promoted)
		assert.Equal(t, remote_service.ReasonPromoted, reason)
	}
}
