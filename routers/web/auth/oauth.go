// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	go_context "context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strings"

	"code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	auth_module "code.gitea.io/gitea/modules/auth"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web/middleware"
	source_service "code.gitea.io/gitea/services/auth/source"
	"code.gitea.io/gitea/services/auth/source/oauth2"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/externalaccount"
	remote_service "code.gitea.io/gitea/services/remote"
	user_service "code.gitea.io/gitea/services/user"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/fitbit"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/markbates/goth/providers/zoom"
	go_oauth2 "golang.org/x/oauth2"
)

// SignInOAuth handles the OAuth2 login buttons
func SignInOAuth(ctx *context.Context) {
	provider := ctx.Params(":provider")

	authSource, err := auth.GetActiveOAuth2SourceByName(ctx, provider)
	if err != nil {
		ctx.ServerError("SignIn", err)
		return
	}

	redirectTo := ctx.FormString("redirect_to")
	if len(redirectTo) > 0 {
		middleware.SetRedirectToCookie(ctx.Resp, redirectTo)
	}

	// try to do a direct callback flow, so we don't authenticate the user again but use the valid accesstoken to get the user
	user, gothUser, err := oAuth2UserLoginCallback(ctx, authSource, ctx.Req, ctx.Resp)
	if err == nil && user != nil {
		// we got the user without going through the whole OAuth2 authentication flow again
		handleOAuth2SignIn(ctx, authSource, user, gothUser)
		return
	}

	codeChallenge, err := generateCodeChallenge(ctx, provider)
	if err != nil {
		ctx.ServerError("SignIn", fmt.Errorf("could not generate code_challenge: %w", err))
		return
	}

	if err = authSource.Cfg.(*oauth2.Source).Callout(ctx.Req, ctx.Resp, codeChallenge); err != nil {
		if strings.Contains(err.Error(), "no provider for ") {
			if err = oauth2.ResetOAuth2(ctx); err != nil {
				ctx.ServerError("SignIn", err)
				return
			}
			if err = authSource.Cfg.(*oauth2.Source).Callout(ctx.Req, ctx.Resp, codeChallenge); err != nil {
				ctx.ServerError("SignIn", err)
			}
			return
		}
		ctx.ServerError("SignIn", err)
	}
	// redirect is done in oauth2.Auth
}

// SignInOAuthCallback handles the callback from the given provider
func SignInOAuthCallback(ctx *context.Context) {
	provider := ctx.Params(":provider")

	if ctx.Req.FormValue("error") != "" {
		var errorKeyValues []string
		for k, vv := range ctx.Req.Form {
			for _, v := range vv {
				errorKeyValues = append(errorKeyValues, fmt.Sprintf("%s = %s", html.EscapeString(k), html.EscapeString(v)))
			}
		}
		sort.Strings(errorKeyValues)
		ctx.Flash.Error(strings.Join(errorKeyValues, "<br>"), true)
	}

	// first look if the provider is still active
	authSource, err := auth.GetActiveOAuth2SourceByName(ctx, provider)
	if err != nil {
		ctx.ServerError("SignIn", err)
		return
	}

	if authSource == nil {
		ctx.ServerError("SignIn", errors.New("no valid provider found, check configured callback url in provider"))
		return
	}

	u, gothUser, err := oAuth2UserLoginCallback(ctx, authSource, ctx.Req, ctx.Resp)
	if err != nil {
		if user_model.IsErrUserProhibitLogin(err) {
			uplerr := err.(user_model.ErrUserProhibitLogin)
			log.Info("Failed authentication attempt for %s from %s: %v", uplerr.Name, ctx.RemoteAddr(), err)
			ctx.Data["Title"] = ctx.Tr("auth.prohibit_login")
			ctx.HTML(http.StatusOK, "user/auth/prohibit_login")
			return
		}
		if callbackErr, ok := err.(errCallback); ok {
			log.Info("Failed OAuth callback: (%v) %v", callbackErr.Code, callbackErr.Description)
			switch callbackErr.Code {
			case "access_denied":
				ctx.Flash.Error(ctx.Tr("auth.oauth.signin.error.access_denied"))
			case "temporarily_unavailable":
				ctx.Flash.Error(ctx.Tr("auth.oauth.signin.error.temporarily_unavailable"))
			default:
				ctx.Flash.Error(ctx.Tr("auth.oauth.signin.error"))
			}
			ctx.Redirect(setting.AppSubURL + "/user/login")
			return
		}
		if err, ok := err.(*go_oauth2.RetrieveError); ok {
			ctx.Flash.Error("OAuth2 RetrieveError: "+err.Error(), true)
			ctx.Redirect(setting.AppSubURL + "/user/login")
			return
		}
		ctx.ServerError("UserSignIn", err)
		return
	}

	if u == nil {
		if ctx.Doer != nil {
			// attach user to already logged in user
			err = externalaccount.LinkAccountToUser(ctx, ctx.Doer, gothUser)
			if err != nil {
				ctx.ServerError("UserLinkAccount", err)
				return
			}

			ctx.Redirect(setting.AppSubURL + "/user/settings/security")
			return
		} else if !setting.Service.AllowOnlyInternalRegistration && setting.OAuth2Client.EnableAutoRegistration {
			// create new user with details from oauth2 provider
			if gothUser.UserID == "" {
				log.Error("OAuth2 Provider %s returned empty or missing field: UserID", authSource.Name)
				if authSource.IsOAuth2() && authSource.Cfg.(*oauth2.Source).Provider == "openidConnect" {
					log.Error("You may need to change the 'OPENID_CONNECT_SCOPES' setting to request all required fields")
				}
				err = fmt.Errorf("OAuth2 Provider %s returned empty or missing field: UserID", authSource.Name)
				ctx.ServerError("CreateUser", err)
				return
			}
			var missingFields []string
			if gothUser.Email == "" {
				missingFields = append(missingFields, "email")
			}
			if setting.OAuth2Client.Username == setting.OAuth2UsernameNickname && gothUser.NickName == "" {
				missingFields = append(missingFields, "nickname")
			}
			if len(missingFields) > 0 {
				// we don't have enough information to create an account automatically,
				// so we prompt the user for the remaining bits
				log.Trace("OAuth2 Provider %s returned empty or missing fields: %s, prompting the user for them", authSource.Name, missingFields)
				showLinkingLogin(ctx, gothUser)
				return
			}
			uname, err := getUserName(&gothUser)
			if err != nil {
				ctx.ServerError("UserSignIn", err)
				return
			}
			u = &user_model.User{
				Name:        uname,
				FullName:    gothUser.Name,
				Email:       gothUser.Email,
				LoginType:   auth.OAuth2,
				LoginSource: authSource.ID,
				LoginName:   gothUser.UserID,
			}

			overwriteDefault := &user_model.CreateUserOverwriteOptions{
				IsActive: optional.Some(!setting.OAuth2Client.RegisterEmailConfirm && !setting.Service.RegisterManualConfirm),
			}

			source := authSource.Cfg.(*oauth2.Source)

			isAdmin, isRestricted := getUserAdminAndRestrictedFromGroupClaims(source, &gothUser)
			u.IsAdmin = isAdmin.ValueOrDefault(false)
			u.IsRestricted = isRestricted.ValueOrDefault(false)

			if !createAndHandleCreatedUser(ctx, base.TplName(""), nil, u, overwriteDefault, &gothUser, setting.OAuth2Client.AccountLinking != setting.OAuth2AccountLinkingDisabled) {
				// error already handled
				return
			}

			if err := syncGroupsToTeams(ctx, source, &gothUser, u); err != nil {
				ctx.ServerError("SyncGroupsToTeams", err)
				return
			}
		} else {
			// no existing user is found, request attach or new account
			showLinkingLogin(ctx, gothUser)
			return
		}
	}

	handleOAuth2SignIn(ctx, authSource, u, gothUser)
}

func claimValueToStringSet(claimValue any) container.Set[string] {
	var groups []string

	switch rawGroup := claimValue.(type) {
	case []string:
		groups = rawGroup
	case []any:
		for _, group := range rawGroup {
			groups = append(groups, fmt.Sprintf("%s", group))
		}
	default:
		str := fmt.Sprintf("%s", rawGroup)
		groups = strings.Split(str, ",")
	}
	return container.SetOf(groups...)
}

func syncGroupsToTeams(ctx *context.Context, source *oauth2.Source, gothUser *goth.User, u *user_model.User) error {
	if source.GroupTeamMap != "" || source.GroupTeamMapRemoval {
		groupTeamMapping, err := auth_module.UnmarshalGroupTeamMapping(source.GroupTeamMap)
		if err != nil {
			return err
		}

		groups := getClaimedGroups(source, gothUser)

		if err := source_service.SyncGroupsToTeams(ctx, u, groups, groupTeamMapping, source.GroupTeamMapRemoval); err != nil {
			return err
		}
	}

	return nil
}

func getClaimedGroups(source *oauth2.Source, gothUser *goth.User) container.Set[string] {
	groupClaims, has := gothUser.RawData[source.GroupClaimName]
	if !has {
		return nil
	}

	return claimValueToStringSet(groupClaims)
}

func getUserAdminAndRestrictedFromGroupClaims(source *oauth2.Source, gothUser *goth.User) (isAdmin, isRestricted optional.Option[bool]) {
	groups := getClaimedGroups(source, gothUser)

	if source.AdminGroup != "" {
		isAdmin = optional.Some(groups.Contains(source.AdminGroup))
	}
	if source.RestrictedGroup != "" {
		isRestricted = optional.Some(groups.Contains(source.RestrictedGroup))
	}

	return isAdmin, isRestricted
}

func showLinkingLogin(ctx *context.Context, gothUser goth.User) {
	if err := updateSession(ctx, nil, map[string]any{
		"linkAccountGothUser": gothUser,
	}); err != nil {
		ctx.ServerError("updateSession", err)
		return
	}
	ctx.Redirect(setting.AppSubURL + "/user/link_account")
}

func updateAvatarIfNeed(ctx *context.Context, url string, u *user_model.User) {
	if setting.OAuth2Client.UpdateAvatar && len(url) > 0 {
		resp, err := http.Get(url)
		if err == nil {
			defer func() {
				_ = resp.Body.Close()
			}()
		}
		// ignore any error
		if err == nil && resp.StatusCode == http.StatusOK {
			data, err := io.ReadAll(io.LimitReader(resp.Body, setting.Avatar.MaxFileSize+1))
			if err == nil && int64(len(data)) <= setting.Avatar.MaxFileSize {
				_ = user_service.UploadAvatar(ctx, u, data)
			}
		}
	}
}

func handleOAuth2SignIn(ctx *context.Context, source *auth.Source, u *user_model.User, gothUser goth.User) {
	updateAvatarIfNeed(ctx, gothUser.AvatarURL, u)

	needs2FA := false
	if !source.Cfg.(*oauth2.Source).SkipLocalTwoFA {
		_, err := auth.GetTwoFactorByUID(ctx, u.ID)
		if err != nil && !auth.IsErrTwoFactorNotEnrolled(err) {
			ctx.ServerError("UserSignIn", err)
			return
		}
		needs2FA = err == nil
	}

	oauth2Source := source.Cfg.(*oauth2.Source)
	groupTeamMapping, err := auth_module.UnmarshalGroupTeamMapping(oauth2Source.GroupTeamMap)
	if err != nil {
		ctx.ServerError("UnmarshalGroupTeamMapping", err)
		return
	}

	groups := getClaimedGroups(oauth2Source, &gothUser)

	opts := &user_service.UpdateOptions{}

	// Reactivate user if they are deactivated
	if !u.IsActive {
		opts.IsActive = optional.Some(true)
	}

	// Update GroupClaims
	opts.IsAdmin, opts.IsRestricted = getUserAdminAndRestrictedFromGroupClaims(oauth2Source, &gothUser)

	if oauth2Source.GroupTeamMap != "" || oauth2Source.GroupTeamMapRemoval {
		if err := source_service.SyncGroupsToTeams(ctx, u, groups, groupTeamMapping, oauth2Source.GroupTeamMapRemoval); err != nil {
			ctx.ServerError("SyncGroupsToTeams", err)
			return
		}
	}

	if err := externalaccount.EnsureLinkExternalToUser(ctx, u, gothUser); err != nil {
		ctx.ServerError("EnsureLinkExternalToUser", err)
		return
	}

	// If this user is enrolled in 2FA and this source doesn't override it,
	// we can't sign the user in just yet. Instead, redirect them to the 2FA authentication page.
	if !needs2FA {
		// Register last login
		opts.SetLastLogin = true

		if err := user_service.UpdateUser(ctx, u, opts); err != nil {
			ctx.ServerError("UpdateUser", err)
			return
		}

		if err := updateSession(ctx, nil, map[string]any{
			"uid": u.ID,
		}); err != nil {
			ctx.ServerError("updateSession", err)
			return
		}

		// Clear whatever CSRF cookie has right now, force to generate a new one
		ctx.Csrf.DeleteCookie(ctx)

		if err := resetLocale(ctx, u); err != nil {
			ctx.ServerError("resetLocale", err)
			return
		}

		if redirectTo := ctx.GetSiteCookie("redirect_to"); len(redirectTo) > 0 {
			middleware.DeleteRedirectToCookie(ctx.Resp)
			ctx.RedirectToFirst(redirectTo)
			return
		}

		ctx.Redirect(setting.AppSubURL + "/")
		return
	}

	if opts.IsActive.Has() || opts.IsAdmin.Has() || opts.IsRestricted.Has() {
		if err := user_service.UpdateUser(ctx, u, opts); err != nil {
			ctx.ServerError("UpdateUser", err)
			return
		}
	}

	if err := updateSession(ctx, nil, map[string]any{
		// User needs to use 2FA, save data and redirect to 2FA page.
		"twofaUid":      u.ID,
		"twofaRemember": false,
	}); err != nil {
		ctx.ServerError("updateSession", err)
		return
	}

	// If WebAuthn is enrolled -> Redirect to WebAuthn instead
	regs, err := auth.GetWebAuthnCredentialsByUID(ctx, u.ID)
	if err == nil && len(regs) > 0 {
		ctx.Redirect(setting.AppSubURL + "/user/webauthn")
		return
	}

	ctx.Redirect(setting.AppSubURL + "/user/two_factor")
}

// generateCodeChallenge stores a code verifier in the session and returns a S256 code challenge for PKCE
func generateCodeChallenge(ctx *context.Context, provider string) (codeChallenge string, err error) {
	// the `code_verifier` is only forwarded by specific providers
	// https://codeberg.org/forgejo/forgejo/issues/4033
	p, ok := goth.GetProviders()[provider]
	if !ok {
		return "", nil
	}
	switch p.(type) {
	default:
		return "", nil
	case *openidConnect.Provider, *fitbit.Provider, *zoom.Provider:
		// those providers forward the `code_verifier`
		// a code_challenge can be generated
	}

	codeVerifier, err := util.CryptoRandomString(43) // 256/log2(62) = 256 bits of entropy (each char having log2(62) of randomness)
	if err != nil {
		return "", err
	}
	if err = ctx.Session.Set("CodeVerifier", codeVerifier); err != nil {
		return "", err
	}
	return encodeCodeChallenge(codeVerifier)
}

func encodeCodeChallenge(codeVerifier string) (string, error) {
	hasher := sha256.New()
	_, err := io.WriteString(hasher, codeVerifier)
	codeChallenge := base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
	return codeChallenge, err
}

// OAuth2UserLoginCallback attempts to handle the callback from the OAuth2 provider and if successful
// login the user
func oAuth2UserLoginCallback(ctx *context.Context, authSource *auth.Source, request *http.Request, response http.ResponseWriter) (*user_model.User, goth.User, error) {
	gothUser, err := oAuth2FetchUser(ctx, authSource, request, response)
	if err != nil {
		return nil, goth.User{}, err
	}

	if _, _, err := remote_service.MaybePromoteRemoteUser(ctx, authSource, gothUser.UserID, gothUser.Email); err != nil {
		return nil, goth.User{}, err
	}

	u, err := oAuth2GothUserToUser(request.Context(), authSource, gothUser)
	return u, gothUser, err
}

func oAuth2FetchUser(ctx *context.Context, authSource *auth.Source, request *http.Request, response http.ResponseWriter) (goth.User, error) {
	oauth2Source := authSource.Cfg.(*oauth2.Source)

	// Make sure that the response is not an error response.
	errorName := request.FormValue("error")

	if len(errorName) > 0 {
		errorDescription := request.FormValue("error_description")

		// Delete the goth session
		err := gothic.Logout(response, request)
		if err != nil {
			return goth.User{}, err
		}

		return goth.User{}, errCallback{
			Code:        errorName,
			Description: errorDescription,
		}
	}

	// Proceed to authenticate through goth.
	codeVerifier, _ := ctx.Session.Get("CodeVerifier").(string)
	_ = ctx.Session.Delete("CodeVerifier")
	gothUser, err := oauth2Source.Callback(request, response, codeVerifier)
	if err != nil {
		if err.Error() == "securecookie: the value is too long" || strings.Contains(err.Error(), "Data too long") {
			log.Error("OAuth2 Provider %s returned too long a token. Current max: %d. Either increase the [OAuth2] MAX_TOKEN_LENGTH or reduce the information returned from the OAuth2 provider", authSource.Name, setting.OAuth2.MaxTokenLength)
			err = fmt.Errorf("OAuth2 Provider %s returned too long a token. Current max: %d. Either increase the [OAuth2] MAX_TOKEN_LENGTH or reduce the information returned from the OAuth2 provider", authSource.Name, setting.OAuth2.MaxTokenLength)
		}
		return goth.User{}, err
	}

	if oauth2Source.RequiredClaimName != "" {
		claimInterface, has := gothUser.RawData[oauth2Source.RequiredClaimName]
		if !has {
			return goth.User{}, user_model.ErrUserProhibitLogin{Name: gothUser.UserID}
		}

		if oauth2Source.RequiredClaimValue != "" {
			groups := claimValueToStringSet(claimInterface)

			if !groups.Contains(oauth2Source.RequiredClaimValue) {
				return goth.User{}, user_model.ErrUserProhibitLogin{Name: gothUser.UserID}
			}
		}
	}

	return gothUser, nil
}

func oAuth2GothUserToUser(ctx go_context.Context, authSource *auth.Source, gothUser goth.User) (*user_model.User, error) {
	user := &user_model.User{
		LoginName:   gothUser.UserID,
		LoginType:   auth.OAuth2,
		LoginSource: authSource.ID,
	}

	hasUser, err := user_model.GetUser(ctx, user)
	if err != nil {
		return nil, err
	}

	if hasUser {
		return user, nil
	}
	log.Debug("no user found for LoginName %v, LoginSource %v, LoginType %v", user.LoginName, user.LoginSource, user.LoginType)

	// search in external linked users
	externalLoginUser := &user_model.ExternalLoginUser{
		ExternalID:    gothUser.UserID,
		LoginSourceID: authSource.ID,
	}
	hasUser, err = user_model.GetExternalLogin(ctx, externalLoginUser)
	if err != nil {
		return nil, err
	}
	if hasUser {
		user, err = user_model.GetUserByID(ctx, externalLoginUser.UserID)
		return user, err
	}

	// no user found to login
	return nil, nil
}
