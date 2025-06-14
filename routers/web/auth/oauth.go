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
	"html/template"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/models/auth"
	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	auth_module "forgejo.org/modules/auth"
	"forgejo.org/modules/base"
	"forgejo.org/modules/container"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/modules/web/middleware"
	auth_service "forgejo.org/services/auth"
	source_service "forgejo.org/services/auth/source"
	"forgejo.org/services/auth/source/oauth2"
	"forgejo.org/services/context"
	"forgejo.org/services/externalaccount"
	"forgejo.org/services/forms"
	remote_service "forgejo.org/services/remote"
	user_service "forgejo.org/services/user"

	"code.forgejo.org/go-chi/binding"
	"github.com/golang-jwt/jwt/v5"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/fitbit"
	"github.com/markbates/goth/providers/openidConnect"
	"github.com/markbates/goth/providers/zoom"
	go_oauth2 "golang.org/x/oauth2"
)

const (
	tplGrantAccess base.TplName = "user/auth/grant"
	tplGrantError  base.TplName = "user/auth/grant_error"
)

// TODO move error and responses to SDK or models

// AuthorizeErrorCode represents an error code specified in RFC 6749
// https://datatracker.ietf.org/doc/html/rfc6749#section-4.2.2.1
type AuthorizeErrorCode string

const (
	// ErrorCodeInvalidRequest represents the according error in RFC 6749
	ErrorCodeInvalidRequest AuthorizeErrorCode = "invalid_request"
	// ErrorCodeUnauthorizedClient represents the according error in RFC 6749
	ErrorCodeUnauthorizedClient AuthorizeErrorCode = "unauthorized_client"
	// ErrorCodeAccessDenied represents the according error in RFC 6749
	ErrorCodeAccessDenied AuthorizeErrorCode = "access_denied"
	// ErrorCodeUnsupportedResponseType represents the according error in RFC 6749
	ErrorCodeUnsupportedResponseType AuthorizeErrorCode = "unsupported_response_type"
	// ErrorCodeInvalidScope represents the according error in RFC 6749
	ErrorCodeInvalidScope AuthorizeErrorCode = "invalid_scope"
	// ErrorCodeServerError represents the according error in RFC 6749
	ErrorCodeServerError AuthorizeErrorCode = "server_error"
	// ErrorCodeTemporaryUnavailable represents the according error in RFC 6749
	ErrorCodeTemporaryUnavailable AuthorizeErrorCode = "temporarily_unavailable"
)

// AuthorizeError represents an error type specified in RFC 6749
// https://datatracker.ietf.org/doc/html/rfc6749#section-4.2.2.1
type AuthorizeError struct {
	ErrorCode        AuthorizeErrorCode `json:"error" form:"error"`
	ErrorDescription string
	State            string
}

// Error returns the error message
func (err AuthorizeError) Error() string {
	return fmt.Sprintf("%s: %s", err.ErrorCode, err.ErrorDescription)
}

// AccessTokenErrorCode represents an error code specified in RFC 6749
// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
type AccessTokenErrorCode string

const (
	// AccessTokenErrorCodeInvalidRequest represents an error code specified in RFC 6749
	AccessTokenErrorCodeInvalidRequest AccessTokenErrorCode = "invalid_request"
	// AccessTokenErrorCodeInvalidClient represents an error code specified in RFC 6749
	AccessTokenErrorCodeInvalidClient = "invalid_client"
	// AccessTokenErrorCodeInvalidGrant represents an error code specified in RFC 6749
	AccessTokenErrorCodeInvalidGrant = "invalid_grant"
	// AccessTokenErrorCodeUnauthorizedClient represents an error code specified in RFC 6749
	AccessTokenErrorCodeUnauthorizedClient = "unauthorized_client"
	// AccessTokenErrorCodeUnsupportedGrantType represents an error code specified in RFC 6749
	AccessTokenErrorCodeUnsupportedGrantType = "unsupported_grant_type"
	// AccessTokenErrorCodeInvalidScope represents an error code specified in RFC 6749
	AccessTokenErrorCodeInvalidScope = "invalid_scope"
)

// AccessTokenError represents an error response specified in RFC 6749
// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
type AccessTokenError struct {
	ErrorCode        AccessTokenErrorCode `json:"error" form:"error"`
	ErrorDescription string               `json:"error_description"`
}

// Error returns the error message
func (err AccessTokenError) Error() string {
	return fmt.Sprintf("%s: %s", err.ErrorCode, err.ErrorDescription)
}

// errCallback represents a oauth2 callback error
type errCallback struct {
	Code        string
	Description string
}

func (err errCallback) Error() string {
	return err.Description
}

// TokenType specifies the kind of token
type TokenType string

const (
	// TokenTypeBearer represents a token type specified in RFC 6749
	TokenTypeBearer TokenType = "bearer"
	// TokenTypeMAC represents a token type specified in RFC 6749
	TokenTypeMAC = "mac"
)

// AccessTokenResponse represents a successful access token response
// https://datatracker.ietf.org/doc/html/rfc6749#section-4.2.2
type AccessTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    TokenType `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token,omitempty"`
}

func newAccessTokenResponse(ctx go_context.Context, grant *auth.OAuth2Grant, serverKey, clientKey oauth2.JWTSigningKey) (*AccessTokenResponse, *AccessTokenError) {
	if setting.OAuth2.InvalidateRefreshTokens {
		if err := grant.IncreaseCounter(ctx); err != nil {
			return nil, &AccessTokenError{
				ErrorCode:        AccessTokenErrorCodeInvalidGrant,
				ErrorDescription: "cannot increase the grant counter",
			}
		}
	}
	// generate access token to access the API
	expirationDate := timeutil.TimeStampNow().Add(setting.OAuth2.AccessTokenExpirationTime)
	accessToken := &oauth2.Token{
		GrantID: grant.ID,
		Type:    oauth2.TypeAccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationDate.AsTime()),
		},
	}
	signedAccessToken, err := accessToken.SignToken(serverKey)
	if err != nil {
		return nil, &AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidRequest,
			ErrorDescription: "cannot sign token",
		}
	}

	// generate refresh token to request an access token after it expired later
	refreshExpirationDate := timeutil.TimeStampNow().Add(setting.OAuth2.RefreshTokenExpirationTime * 60 * 60).AsTime()
	refreshToken := &oauth2.Token{
		GrantID: grant.ID,
		Counter: grant.Counter,
		Type:    oauth2.TypeRefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirationDate),
		},
	}
	signedRefreshToken, err := refreshToken.SignToken(serverKey)
	if err != nil {
		return nil, &AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidRequest,
			ErrorDescription: "cannot sign token",
		}
	}

	// generate OpenID Connect id_token
	signedIDToken := ""
	if grant.ScopeContains("openid") {
		app, err := auth.GetOAuth2ApplicationByID(ctx, grant.ApplicationID)
		if err != nil {
			return nil, &AccessTokenError{
				ErrorCode:        AccessTokenErrorCodeInvalidRequest,
				ErrorDescription: "cannot find application",
			}
		}
		user, err := user_model.GetUserByID(ctx, grant.UserID)
		if err != nil {
			if user_model.IsErrUserNotExist(err) {
				return nil, &AccessTokenError{
					ErrorCode:        AccessTokenErrorCodeInvalidRequest,
					ErrorDescription: "cannot find user",
				}
			}
			log.Error("Error loading user: %v", err)
			return nil, &AccessTokenError{
				ErrorCode:        AccessTokenErrorCodeInvalidRequest,
				ErrorDescription: "server error",
			}
		}

		idToken := &oauth2.OIDCToken{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationDate.AsTime()),
				Issuer:    strings.TrimSuffix(setting.AppURL, "/"),
				Audience:  []string{app.ClientID},
				Subject:   fmt.Sprint(grant.UserID),
			},
			Nonce: grant.Nonce,
		}
		if grant.ScopeContains("profile") {
			idToken.Name = user.DisplayName()
			idToken.PreferredUsername = user.Name
			idToken.Profile = user.HTMLURL()
			idToken.Picture = user.AvatarLink(ctx)
			idToken.Website = user.Website
			idToken.Locale = user.Language
			idToken.UpdatedAt = user.UpdatedUnix
		}
		if grant.ScopeContains("email") {
			idToken.Email = user.Email
			idToken.EmailVerified = user.IsActive
		}
		if grant.ScopeContains("groups") {
			onlyPublicGroups := ifOnlyPublicGroups(grant.Scope)

			groups, err := getOAuthGroupsForUser(ctx, user, onlyPublicGroups)
			if err != nil {
				log.Error("Error getting groups: %v", err)
				return nil, &AccessTokenError{
					ErrorCode:        AccessTokenErrorCodeInvalidRequest,
					ErrorDescription: "server error",
				}
			}
			idToken.Groups = groups
		}

		signedIDToken, err = idToken.SignToken(clientKey)
		if err != nil {
			return nil, &AccessTokenError{
				ErrorCode:        AccessTokenErrorCodeInvalidRequest,
				ErrorDescription: "cannot sign token",
			}
		}
	}

	return &AccessTokenResponse{
		AccessToken:  signedAccessToken,
		TokenType:    TokenTypeBearer,
		ExpiresIn:    setting.OAuth2.AccessTokenExpirationTime,
		RefreshToken: signedRefreshToken,
		IDToken:      signedIDToken,
	}, nil
}

type userInfoResponse struct {
	Sub      string   `json:"sub"`
	Name     string   `json:"name"`
	Username string   `json:"preferred_username"`
	Email    string   `json:"email"`
	Picture  string   `json:"picture"`
	Groups   []string `json:"groups,omitempty"`
}

func ifOnlyPublicGroups(scopes string) bool {
	scopes = strings.ReplaceAll(scopes, ",", " ")
	scopesList := strings.Fields(scopes)
	for _, scope := range scopesList {
		if scope == "all" || scope == "read:organization" || scope == "read:admin" {
			return false
		}
	}
	return true
}

// InfoOAuth manages request for userinfo endpoint
func InfoOAuth(ctx *context.Context) {
	if ctx.Doer == nil || ctx.Data["AuthedMethod"] != (&auth_service.OAuth2{}).Name() {
		ctx.Resp.Header().Set("WWW-Authenticate", `Bearer realm=""`)
		ctx.PlainText(http.StatusUnauthorized, "no valid authorization")
		return
	}

	response := &userInfoResponse{
		Sub:      fmt.Sprint(ctx.Doer.ID),
		Name:     ctx.Doer.DisplayName(),
		Username: ctx.Doer.Name,
		Email:    ctx.Doer.Email,
		Picture:  ctx.Doer.AvatarLink(ctx),
	}

	var token string
	if auHead := ctx.Req.Header.Get("Authorization"); auHead != "" {
		auths := strings.Fields(auHead)
		if len(auths) == 2 && (auths[0] == "token" || strings.ToLower(auths[0]) == "bearer") {
			token = auths[1]
		}
	}

	_, grantScopes := auth_service.CheckOAuthAccessToken(ctx, token)
	onlyPublicGroups := ifOnlyPublicGroups(grantScopes)

	groups, err := getOAuthGroupsForUser(ctx, ctx.Doer, onlyPublicGroups)
	if err != nil {
		ctx.ServerError("Oauth groups for user", err)
		return
	}
	response.Groups = groups

	ctx.JSON(http.StatusOK, response)
}

// returns a list of "org" and "org:team" strings,
// that the given user is a part of.
func getOAuthGroupsForUser(ctx go_context.Context, user *user_model.User, onlyPublicGroups bool) ([]string, error) {
	orgs, err := org_model.GetUserOrgsList(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("GetUserOrgList: %w", err)
	}

	var groups []string
	for _, org := range orgs {
		if setting.OAuth2.EnableAdditionalGrantScopes {
			if onlyPublicGroups {
				public, err := org_model.IsPublicMembership(ctx, org.ID, user.ID)
				if !public && err == nil {
					continue
				}
			}
		}

		groups = append(groups, org.Name)
		teams, err := org.LoadTeams(ctx)
		if err != nil {
			return nil, fmt.Errorf("LoadTeams: %w", err)
		}
		for _, team := range teams {
			if team.IsMember(ctx, user.ID) {
				groups = append(groups, org.Name+":"+team.LowerName)
			}
		}
	}
	return groups, nil
}

func parseBasicAuth(ctx *context.Context) (username, password string, err error) {
	authHeader := ctx.Req.Header.Get("Authorization")
	if authType, authData, ok := strings.Cut(authHeader, " "); ok && strings.EqualFold(authType, "Basic") {
		return base.BasicAuthDecode(authData)
	}
	return "", "", errors.New("invalid basic authentication")
}

// IntrospectOAuth introspects an oauth token
func IntrospectOAuth(ctx *context.Context) {
	clientIDValid := false
	if clientID, clientSecret, err := parseBasicAuth(ctx); err == nil {
		app, err := auth.GetOAuth2ApplicationByClientID(ctx, clientID)
		if err != nil && !auth.IsErrOauthClientIDInvalid(err) {
			// this is likely a database error; log it and respond without details
			log.Error("Error retrieving client_id: %v", err)
			ctx.Error(http.StatusInternalServerError)
			return
		}
		clientIDValid = err == nil && app.ValidateClientSecret([]byte(clientSecret))
	}
	if !clientIDValid {
		ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm=""`)
		ctx.PlainText(http.StatusUnauthorized, "no valid authorization")
		return
	}

	var response struct {
		Active   bool   `json:"active"`
		Scope    string `json:"scope,omitempty"`
		Username string `json:"username,omitempty"`
		jwt.RegisteredClaims
	}

	form := web.GetForm(ctx).(*forms.IntrospectTokenForm)
	token, err := oauth2.ParseToken(form.Token, oauth2.DefaultSigningKey)
	if err == nil {
		grant, err := auth.GetOAuth2GrantByID(ctx, token.GrantID)
		if err == nil && grant != nil {
			app, err := auth.GetOAuth2ApplicationByID(ctx, grant.ApplicationID)
			if err == nil && app != nil {
				response.Active = true
				response.Scope = grant.Scope
				response.Issuer = strings.TrimSuffix(setting.AppURL, "/")
				response.Audience = []string{app.ClientID}
				response.Subject = fmt.Sprint(grant.UserID)
			}
			if user, err := user_model.GetUserByID(ctx, grant.UserID); err == nil {
				response.Username = user.Name
			}
		}
	}

	ctx.JSON(http.StatusOK, response)
}

// AuthorizeOAuth manages authorize requests
func AuthorizeOAuth(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AuthorizationForm)
	errs := binding.Errors{}
	errs = form.Validate(ctx.Req, errs)
	if len(errs) > 0 {
		errstring := ""
		for _, e := range errs {
			errstring += e.Error() + "\n"
		}
		ctx.ServerError("AuthorizeOAuth: Validate: ", fmt.Errorf("errors occurred during validation: %s", errstring))
		return
	}

	app, err := auth.GetOAuth2ApplicationByClientID(ctx, form.ClientID)
	if err != nil {
		if auth.IsErrOauthClientIDInvalid(err) {
			handleAuthorizeError(ctx, AuthorizeError{
				ErrorCode:        ErrorCodeUnauthorizedClient,
				ErrorDescription: "Client ID not registered",
				State:            form.State,
			}, "")
			return
		}
		ctx.ServerError("GetOAuth2ApplicationByClientID", err)
		return
	}

	var user *user_model.User
	if app.UID != 0 {
		user, err = user_model.GetUserByID(ctx, app.UID)
		if err != nil {
			ctx.ServerError("GetUserByID", err)
			return
		}
	}

	if !app.ContainsRedirectURI(form.RedirectURI) {
		handleAuthorizeError(ctx, AuthorizeError{
			ErrorCode:        ErrorCodeInvalidRequest,
			ErrorDescription: "Unregistered Redirect URI",
			State:            form.State,
		}, "")
		return
	}

	if form.ResponseType != "code" {
		handleAuthorizeError(ctx, AuthorizeError{
			ErrorCode:        ErrorCodeUnsupportedResponseType,
			ErrorDescription: "Only code response type is supported.",
			State:            form.State,
		}, form.RedirectURI)
		return
	}

	// pkce support
	switch form.CodeChallengeMethod {
	case "S256":
	case "plain":
		if err := ctx.Session.Set("CodeChallengeMethod", form.CodeChallengeMethod); err != nil {
			handleAuthorizeError(ctx, AuthorizeError{
				ErrorCode:        ErrorCodeServerError,
				ErrorDescription: "cannot set code challenge method",
				State:            form.State,
			}, form.RedirectURI)
			return
		}
		if err := ctx.Session.Set("CodeChallengeMethod", form.CodeChallenge); err != nil {
			handleAuthorizeError(ctx, AuthorizeError{
				ErrorCode:        ErrorCodeServerError,
				ErrorDescription: "cannot set code challenge",
				State:            form.State,
			}, form.RedirectURI)
			return
		}
		// Here we're just going to try to release the session early
		if err := ctx.Session.Release(); err != nil {
			// we'll tolerate errors here as they *should* get saved elsewhere
			log.Error("Unable to save changes to the session: %v", err)
		}
	case "":
		// "Authorization servers SHOULD reject authorization requests from native apps that don't use PKCE by returning an error message"
		// https://datatracker.ietf.org/doc/html/rfc8252#section-8.1
		if !app.ConfidentialClient {
			// "the authorization endpoint MUST return the authorization error response with the "error" value set to "invalid_request""
			// https://datatracker.ietf.org/doc/html/rfc7636#section-4.4.1
			handleAuthorizeError(ctx, AuthorizeError{
				ErrorCode:        ErrorCodeInvalidRequest,
				ErrorDescription: "PKCE is required for public clients",
				State:            form.State,
			}, form.RedirectURI)
			return
		}
	default:
		// "If the server supporting PKCE does not support the requested transformation, the authorization endpoint MUST return the authorization error response with "error" value set to "invalid_request"."
		// https://www.rfc-editor.org/rfc/rfc7636#section-4.4.1
		handleAuthorizeError(ctx, AuthorizeError{
			ErrorCode:        ErrorCodeInvalidRequest,
			ErrorDescription: "unsupported code challenge method",
			State:            form.State,
		}, form.RedirectURI)
		return
	}

	grant, err := app.GetGrantByUserID(ctx, ctx.Doer.ID)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		return
	}

	// Redirect if user already granted access and the application is confidential.
	// I.e. always require authorization for public clients as recommended by RFC 6749 Section 10.2
	if app.ConfidentialClient && grant != nil {
		code, err := grant.GenerateNewAuthorizationCode(ctx, form.RedirectURI, form.CodeChallenge, form.CodeChallengeMethod)
		if err != nil {
			handleServerError(ctx, form.State, form.RedirectURI, err)
			return
		}
		redirect, err := code.GenerateRedirectURI(form.State)
		if err != nil {
			handleServerError(ctx, form.State, form.RedirectURI, err)
			return
		}
		// Update nonce to reflect the new session
		if len(form.Nonce) > 0 {
			err := grant.SetNonce(ctx, form.Nonce)
			if err != nil {
				log.Error("Unable to update nonce: %v", err)
			}
		}
		ctx.Redirect(redirect.String())
		return
	}

	// show authorize page to grant access
	ctx.Data["Application"] = app
	ctx.Data["RedirectURI"] = form.RedirectURI
	ctx.Data["State"] = form.State
	ctx.Data["Scope"] = form.Scope
	ctx.Data["Nonce"] = form.Nonce
	if user != nil {
		ctx.Data["ApplicationCreatorLinkHTML"] = template.HTML(fmt.Sprintf(`<a href="%s">@%s</a>`, html.EscapeString(user.HomeLink()), html.EscapeString(user.Name)))
	} else {
		ctx.Data["ApplicationCreatorLinkHTML"] = template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(setting.AppSubURL+"/"), html.EscapeString(setting.AppName)))
	}
	ctx.Data["ApplicationRedirectDomainHTML"] = template.HTML("<strong>" + html.EscapeString(form.RedirectURI) + "</strong>")
	// TODO document SESSION <=> FORM
	err = ctx.Session.Set("client_id", app.ClientID)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		log.Error(err.Error())
		return
	}
	err = ctx.Session.Set("redirect_uri", form.RedirectURI)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		log.Error(err.Error())
		return
	}
	err = ctx.Session.Set("state", form.State)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		log.Error(err.Error())
		return
	}
	// Here we're just going to try to release the session early
	if err := ctx.Session.Release(); err != nil {
		// we'll tolerate errors here as they *should* get saved elsewhere
		log.Error("Unable to save changes to the session: %v", err)
	}
	ctx.HTML(http.StatusOK, tplGrantAccess)
}

// GrantApplicationOAuth manages the post request submitted when a user grants access to an application
func GrantApplicationOAuth(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.GrantApplicationForm)
	if ctx.Session.Get("client_id") != form.ClientID || ctx.Session.Get("state") != form.State ||
		ctx.Session.Get("redirect_uri") != form.RedirectURI {
		ctx.Error(http.StatusBadRequest)
		return
	}

	if !form.Granted {
		handleAuthorizeError(ctx, AuthorizeError{
			State:            form.State,
			ErrorDescription: "the request is denied",
			ErrorCode:        ErrorCodeAccessDenied,
		}, form.RedirectURI)
		return
	}

	app, err := auth.GetOAuth2ApplicationByClientID(ctx, form.ClientID)
	if err != nil {
		ctx.ServerError("GetOAuth2ApplicationByClientID", err)
		return
	}
	grant, err := app.GetGrantByUserID(ctx, ctx.Doer.ID)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		return
	}
	if grant == nil {
		grant, err = app.CreateGrant(ctx, ctx.Doer.ID, form.Scope)
		if err != nil {
			handleAuthorizeError(ctx, AuthorizeError{
				State:            form.State,
				ErrorDescription: "cannot create grant for user",
				ErrorCode:        ErrorCodeServerError,
			}, form.RedirectURI)
			return
		}
	} else if grant.Scope != form.Scope {
		handleAuthorizeError(ctx, AuthorizeError{
			State:            form.State,
			ErrorDescription: "a grant exists with different scope",
			ErrorCode:        ErrorCodeServerError,
		}, form.RedirectURI)
		return
	}

	if len(form.Nonce) > 0 {
		err := grant.SetNonce(ctx, form.Nonce)
		if err != nil {
			log.Error("Unable to update nonce: %v", err)
		}
	}

	var codeChallenge, codeChallengeMethod string
	codeChallenge, _ = ctx.Session.Get("CodeChallenge").(string)
	codeChallengeMethod, _ = ctx.Session.Get("CodeChallengeMethod").(string)

	code, err := grant.GenerateNewAuthorizationCode(ctx, form.RedirectURI, codeChallenge, codeChallengeMethod)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		return
	}
	redirect, err := code.GenerateRedirectURI(form.State)
	if err != nil {
		handleServerError(ctx, form.State, form.RedirectURI, err)
		return
	}
	ctx.Redirect(redirect.String(), http.StatusSeeOther)
}

// OIDCWellKnown generates JSON so OIDC clients know Gitea's capabilities
func OIDCWellKnown(ctx *context.Context) {
	ctx.Data["SigningKey"] = oauth2.DefaultSigningKey
	ctx.Data["Issuer"] = strings.TrimSuffix(setting.AppURL, "/")
	ctx.JSONTemplate("user/auth/oidc_wellknown")
}

// OIDCKeys generates the JSON Web Key Set
func OIDCKeys(ctx *context.Context) {
	jwk, err := oauth2.DefaultSigningKey.ToJWK()
	if err != nil {
		log.Error("Error converting signing key to JWK: %v", err)
		ctx.Error(http.StatusInternalServerError)
		return
	}

	jwk["use"] = "sig"

	jwks := map[string][]map[string]string{
		"keys": {
			jwk,
		},
	}

	ctx.Resp.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(ctx.Resp)
	if err := enc.Encode(jwks); err != nil {
		log.Error("Failed to encode representation as json. Error: %v", err)
	}
}

// AccessTokenOAuth manages all access token requests by the client
func AccessTokenOAuth(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.AccessTokenForm)
	// if there is no ClientID or ClientSecret in the request body, fill these fields by the Authorization header and ensure the provided field matches the Authorization header
	if form.ClientID == "" || form.ClientSecret == "" {
		authHeader := ctx.Req.Header.Get("Authorization")
		if authType, authData, ok := strings.Cut(authHeader, " "); ok && strings.EqualFold(authType, "Basic") {
			clientID, clientSecret, err := base.BasicAuthDecode(authData)
			if err != nil {
				handleAccessTokenError(ctx, AccessTokenError{
					ErrorCode:        AccessTokenErrorCodeInvalidRequest,
					ErrorDescription: "cannot parse basic auth header",
				})
				return
			}
			// validate that any fields present in the form match the Basic auth header
			if form.ClientID != "" && form.ClientID != clientID {
				handleAccessTokenError(ctx, AccessTokenError{
					ErrorCode:        AccessTokenErrorCodeInvalidRequest,
					ErrorDescription: "client_id in request body inconsistent with Authorization header",
				})
				return
			}
			form.ClientID = clientID
			if form.ClientSecret != "" && form.ClientSecret != clientSecret {
				handleAccessTokenError(ctx, AccessTokenError{
					ErrorCode:        AccessTokenErrorCodeInvalidRequest,
					ErrorDescription: "client_secret in request body inconsistent with Authorization header",
				})
				return
			}
			form.ClientSecret = clientSecret
		}
	}

	serverKey := oauth2.DefaultSigningKey
	clientKey := serverKey
	if serverKey.IsSymmetric() {
		var err error
		clientKey, err = oauth2.CreateJWTSigningKey(serverKey.SigningMethod().Alg(), []byte(form.ClientSecret))
		if err != nil {
			handleAccessTokenError(ctx, AccessTokenError{
				ErrorCode:        AccessTokenErrorCodeInvalidRequest,
				ErrorDescription: "Error creating signing key",
			})
			return
		}
	}

	switch form.GrantType {
	case "refresh_token":
		handleRefreshToken(ctx, form, serverKey, clientKey)
	case "authorization_code":
		handleAuthorizationCode(ctx, form, serverKey, clientKey)
	default:
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnsupportedGrantType,
			ErrorDescription: "Only refresh_token or authorization_code grant type is supported",
		})
	}
}

func handleRefreshToken(ctx *context.Context, form forms.AccessTokenForm, serverKey, clientKey oauth2.JWTSigningKey) {
	app, err := auth.GetOAuth2ApplicationByClientID(ctx, form.ClientID)
	if err != nil {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidClient,
			ErrorDescription: fmt.Sprintf("cannot load client with client id: %q", form.ClientID),
		})
		return
	}
	// "The authorization server MUST ... require client authentication for confidential clients"
	// https://datatracker.ietf.org/doc/html/rfc6749#section-6
	if app.ConfidentialClient && !app.ValidateClientSecret([]byte(form.ClientSecret)) {
		errorDescription := "invalid client secret"
		if form.ClientSecret == "" {
			errorDescription = "invalid empty client secret"
		}
		// "invalid_client ... Client authentication failed"
		// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidClient,
			ErrorDescription: errorDescription,
		})
		return
	}

	token, err := oauth2.ParseToken(form.RefreshToken, serverKey)
	if err != nil {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnauthorizedClient,
			ErrorDescription: "unable to parse refresh token",
		})
		return
	}
	// get grant before increasing counter
	grant, err := auth.GetOAuth2GrantByID(ctx, token.GrantID)
	if err != nil || grant == nil {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidGrant,
			ErrorDescription: "grant does not exist",
		})
		return
	}

	// check if token got already used
	if setting.OAuth2.InvalidateRefreshTokens && (grant.Counter != token.Counter || token.Counter == 0) {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnauthorizedClient,
			ErrorDescription: "token was already used",
		})
		log.Warn("A client tried to use a refresh token for grant_id = %d was used twice!", grant.ID)
		return
	}
	accessToken, tokenErr := newAccessTokenResponse(ctx, grant, serverKey, clientKey)
	if tokenErr != nil {
		handleAccessTokenError(ctx, *tokenErr)
		return
	}
	ctx.JSON(http.StatusOK, accessToken)
}

func handleAuthorizationCode(ctx *context.Context, form forms.AccessTokenForm, serverKey, clientKey oauth2.JWTSigningKey) {
	app, err := auth.GetOAuth2ApplicationByClientID(ctx, form.ClientID)
	if err != nil {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidClient,
			ErrorDescription: fmt.Sprintf("cannot load client with client id: '%s'", form.ClientID),
		})
		return
	}
	if app.ConfidentialClient && !app.ValidateClientSecret([]byte(form.ClientSecret)) {
		errorDescription := "invalid client secret"
		if form.ClientSecret == "" {
			errorDescription = "invalid empty client secret"
		}
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnauthorizedClient,
			ErrorDescription: errorDescription,
		})
		return
	}
	if form.RedirectURI != "" && !app.ContainsRedirectURI(form.RedirectURI) {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnauthorizedClient,
			ErrorDescription: "unexpected redirect URI",
		})
		return
	}
	authorizationCode, err := auth.GetOAuth2AuthorizationByCode(ctx, form.Code)
	if err != nil || authorizationCode == nil {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnauthorizedClient,
			ErrorDescription: "client is not authorized",
		})
		return
	}
	// check if code verifier authorizes the client, PKCE support
	if !authorizationCode.ValidateCodeChallenge(form.CodeVerifier) {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeUnauthorizedClient,
			ErrorDescription: "failed PKCE code challenge",
		})
		return
	}
	// check if granted for this application
	if authorizationCode.Grant.ApplicationID != app.ID {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidGrant,
			ErrorDescription: "invalid grant",
		})
		return
	}
	// remove token from database to deny duplicate usage
	if err := authorizationCode.Invalidate(ctx); err != nil {
		handleAccessTokenError(ctx, AccessTokenError{
			ErrorCode:        AccessTokenErrorCodeInvalidRequest,
			ErrorDescription: "cannot proceed your request",
		})
	}
	resp, tokenErr := newAccessTokenResponse(ctx, authorizationCode.Grant, serverKey, clientKey)
	if tokenErr != nil {
		handleAccessTokenError(ctx, *tokenErr)
		return
	}
	// send successful response
	ctx.JSON(http.StatusOK, resp)
}

func handleAccessTokenError(ctx *context.Context, acErr AccessTokenError) {
	ctx.JSON(http.StatusBadRequest, acErr)
}

func handleServerError(ctx *context.Context, state, redirectURI string, err error) {
	log.Error("OAuth server error: %v", err)
	handleAuthorizeError(ctx, AuthorizeError{
		ErrorCode:        ErrorCodeServerError,
		ErrorDescription: "A server error occurred",
		State:            state,
	}, redirectURI)
}

func handleAuthorizeError(ctx *context.Context, authErr AuthorizeError, redirectURI string) {
	if redirectURI == "" {
		log.Warn("Authorization failed: %v", authErr.ErrorDescription)
		ctx.Data["Error"] = authErr
		ctx.HTML(http.StatusBadRequest, tplGrantError)
		return
	}
	redirect, err := url.Parse(redirectURI)
	if err != nil {
		ctx.ServerError("url.Parse", err)
		return
	}
	q := redirect.Query()
	q.Set("error", string(authErr.ErrorCode))
	q.Set("error_description", authErr.ErrorDescription)
	q.Set("state", authErr.State)
	redirect.RawQuery = q.Encode()
	ctx.Redirect(redirect.String(), http.StatusSeeOther)
}

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
			u.IsRestricted = isRestricted.ValueOrDefault(setting.Service.DefaultUserIsRestricted)

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

func getSSHKeys(source *oauth2.Source, gothUser *goth.User) ([]string, error) {
	key := source.AttributeSSHPublicKey
	value, exists := gothUser.RawData[key]
	if !exists {
		return []string{}, nil
	}

	rawSlice, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for SSH public key, expected []interface{} but got %T", value)
	}

	sshKeys := make([]string, 0, len(rawSlice))
	for i, v := range rawSlice {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected element type at index %d in SSH public key array, expected string but got %T", i, v)
		}
		sshKeys = append(sshKeys, str)
	}

	return sshKeys, nil
}

func updateSSHPubIfNeed(
	ctx *context.Context,
	authSource *auth.Source,
	fetchedUser *goth.User,
	user *user_model.User,
) error {
	oauth2Source := authSource.Cfg.(*oauth2.Source)

	if oauth2Source.ProvidesSSHKeys() {
		sshKeys, err := getSSHKeys(oauth2Source, fetchedUser)
		if err != nil {
			return err
		}

		if asymkey_model.SynchronizePublicKeys(ctx, user, authSource, sshKeys) {
			err = asymkey_model.RewriteAllPublicKeys(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func handleOAuth2SignIn(ctx *context.Context, source *auth.Source, u *user_model.User, gothUser goth.User) {
	updateAvatarIfNeed(ctx, gothUser.AvatarURL, u)
	err := updateSSHPubIfNeed(ctx, source, &gothUser, u)
	if err != nil {
		ctx.ServerError("updateSSHPubIfNeed", err)
		return
	}

	needs2FA := false
	if !source.Cfg.(*oauth2.Source).SkipLocalTwoFA {
		needs2FA, err = auth.HasTwoFactorByUID(ctx, u.ID)
		if err != nil {
			ctx.ServerError("UserSignIn", err)
			return
		}
	}

	oauth2Source := source.Cfg.(*oauth2.Source)
	groupTeamMapping, err := auth_module.UnmarshalGroupTeamMapping(oauth2Source.GroupTeamMap)
	if err != nil {
		ctx.ServerError("UnmarshalGroupTeamMapping", err)
		return
	}

	groups := getClaimedGroups(oauth2Source, &gothUser)

	// If this user is enrolled in 2FA and this source doesn't override it,
	// we can't sign the user in just yet. Instead, redirect them to the 2FA authentication page.
	if !needs2FA {
		if err := updateSession(ctx, nil, map[string]any{
			"uid": u.ID,
		}); err != nil {
			ctx.ServerError("updateSession", err)
			return
		}

		// Clear whatever CSRF cookie has right now, force to generate a new one
		ctx.Csrf.DeleteCookie(ctx)

		opts := &user_service.UpdateOptions{
			SetLastLogin: true,
		}
		opts.IsAdmin, opts.IsRestricted = getUserAdminAndRestrictedFromGroupClaims(oauth2Source, &gothUser)
		if err := user_service.UpdateUser(ctx, u, opts); err != nil {
			ctx.ServerError("UpdateUser", err)
			return
		}

		if oauth2Source.GroupTeamMap != "" || oauth2Source.GroupTeamMapRemoval {
			if err := source_service.SyncGroupsToTeams(ctx, u, groups, groupTeamMapping, oauth2Source.GroupTeamMapRemoval); err != nil {
				ctx.ServerError("SyncGroupsToTeams", err)
				return
			}
		}

		// update external user information
		if err := externalaccount.UpdateExternalUser(ctx, u, gothUser); err != nil {
			if !errors.Is(err, util.ErrNotExist) {
				log.Error("UpdateExternalUser failed: %v", err)
			}
		}

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

	opts := &user_service.UpdateOptions{}
	opts.IsAdmin, opts.IsRestricted = getUserAdminAndRestrictedFromGroupClaims(oauth2Source, &gothUser)
	if opts.IsAdmin.Has() || opts.IsRestricted.Has() {
		if err := user_service.UpdateUser(ctx, u, opts); err != nil {
			ctx.ServerError("UpdateUser", err)
			return
		}
	}

	if oauth2Source.GroupTeamMap != "" || oauth2Source.GroupTeamMapRemoval {
		if err := source_service.SyncGroupsToTeams(ctx, u, groups, groupTeamMapping, oauth2Source.GroupTeamMapRemoval); err != nil {
			ctx.ServerError("SyncGroupsToTeams", err)
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
