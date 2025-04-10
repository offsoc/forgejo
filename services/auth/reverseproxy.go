// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-FileCopyrightText: 2025 Informatyka Boguslawski sp. z o.o. sp.k. <https://www.ib.pl>
// SPDX-License-Identifier: MIT AND GPL-3.0-or-later

package auth

import (
	"fmt"
	"net/http"
	"strings"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web/middleware"

	gouuid "github.com/google/uuid"
)

// Ensure the struct implements the interface.
var (
	_ Method = &ReverseProxy{}
)

// ReverseProxyMethodName is the constant name of the ReverseProxy authentication method
const ReverseProxyMethodName = "reverse_proxy"

// ReverseProxy implements the Auth interface, but actually relies on
// a reverse proxy for authentication of users.
// On successful authentication the proxy is expected to populate the username in the
// "setting.ReverseProxyAuthUser" header. Optionally it can also populate the email of the
// user in the "setting.ReverseProxyAuthEmail" header.
type ReverseProxy struct{}

// getUserName extracts the username from the "setting.ReverseProxyAuthUser" header
func (r *ReverseProxy) getUserName(req *http.Request) string {
	return strings.TrimSpace(req.Header.Get(setting.ReverseProxyAuthUser))
}

// Name represents the name of auth method
func (r *ReverseProxy) Name() string {
	return ReverseProxyMethodName
}

// getUserFromAuthUser extracts the username from the "setting.ReverseProxyAuthUser" header
// of the request and returns the corresponding user object for that name.
// Verification of header data is not performed as it should have already been done by
// the reverse proxy.
// If a username is available in the "setting.ReverseProxyAuthUser" header an existing
// user object is returned (populated with username or email found in header).
// Returns nil if header is empty.
func (r *ReverseProxy) getUserFromAuthUser(req *http.Request) (*user_model.User, error) {
	username := r.getUserName(req)
	if len(username) == 0 {
		return nil, nil
	}
	log.Trace("ReverseProxy Authorization: Found username: %s", username)

	user, err := user_model.GetUserByName(req.Context(), username)
	if err != nil {
		if !user_model.IsErrUserNotExist(err) || !r.isAutoRegisterAllowed() {
			log.Error("GetUserByName: %v", err)
			return nil, err
		}
		user = r.newUser(req)
	}
	return user, nil
}

func (r *ReverseProxy) getUsernameFromAuthUser(req *http.Request) string {
	username := r.getUserName(req)
	if len(username) == 0 {
		return ""
	}
	log.Trace("ReverseProxy Authorization: Found username: %s", username)
	return username
}

// getEmail extracts the email from the "setting.ReverseProxyAuthEmail" header
func (r *ReverseProxy) getEmail(req *http.Request) string {
	return strings.TrimSpace(req.Header.Get(setting.ReverseProxyAuthEmail))
}

// getUserFromAuthEmail extracts the username from the "setting.ReverseProxyAuthEmail" header
// of the request and returns the corresponding user object for that email.
// Verification of header data is not performed as it should have already been done by
// the reverse proxy.
// If an email is available in the "setting.ReverseProxyAuthEmail" header an existing
// user object is returned (populated with the email found in header).
// Returns nil if header is empty or if "setting.EnableReverseProxyEmail" is disabled.
func (r *ReverseProxy) getUserFromAuthEmail(req *http.Request) *user_model.User {
	if !setting.Service.EnableReverseProxyEmail {
		return nil
	}
	email := r.getEmail(req)
	if len(email) == 0 {
		return nil
	}
	log.Trace("ReverseProxy Authorization: Found email: %s", email)

	user, err := user_model.GetUserByEmail(req.Context(), email)
	if err != nil {
		// Do not allow auto-registration, we don't have a username here
		if !user_model.IsErrUserNotExist(err) {
			log.Error("GetUserByEmail: %v", err)
		}
		return nil
	}
	return user
}

func (r *ReverseProxy) getUsernameFromAuthEmail(req *http.Request) string {
	if !setting.Service.EnableReverseProxyEmail {
		return ""
	}
	email := r.getEmail(req)
	if len(email) == 0 {
		return ""
	}
	log.Trace("ReverseProxy Authorization: Found email: %s", email)

	user, err := user_model.GetUserByEmail(req.Context(), email)
	if err != nil {
		// Do not allow auto-registration, we don't have a username here
		if !user_model.IsErrUserNotExist(err) {
			log.Error("GetUserByEmail: %v", err)
		}
		return ""
	}
	return user.LoginName
}

// Verify attempts to load a user object based on headers sent by the reverse proxy.
// First it will attempt to load it based on the username (see docs for getUserFromAuthUser),
// and failing that it will attempt to load it based on the email (see docs for getUserFromAuthEmail).
// Returns nil if the headers are empty or the user is not found or internal API is being called.
func (r *ReverseProxy) Verify(req *http.Request, w http.ResponseWriter, store DataStore, sess SessionStore) (*user_model.User, error) {

	// Internal API should not use this auth method.
	if middleware.IsInternalPath(req) {
		return nil, nil
	}

	var user *user_model.User = nil

	if r.isAutoRegisterAllowed() {

		// Use auto registration from reverse proxy if ENABLE_REVERSE_PROXY_AUTO_REGISTRATION enabled.

		user, err := r.getUserFromAuthUser(req)
		if err != nil {
			return nil, err
		}
		if user == nil {
			user = r.getUserFromAuthEmail(req)
			if user == nil {
				return nil, nil
			}
		}

	} else {

		// Use auto registration from other backends if ENABLE_REVERSE_PROXY_AUTO_REGISTRATION not enabled.
		// UserSignIn is required for user auto registration and update on login from LDAP.

		username := r.getUsernameFromAuthUser(req)
		if username == "" {
			username = r.getUsernameFromAuthEmail(req)
		}

		// Finish if no user found for reverse proxy auth.
		if username == "" {
			return nil, nil
		}

		u, _, err := UserSignIn(req.Context(), username, "")
		if err != nil {
			if !user_model.IsErrUserNotExist(err) {
				log.Error("UserSignIn: %v", err)
			}
			return nil, err
		}
		user = u
	}

	if user == nil {
		return nil, nil
	}

	// Make sure requests to API paths, attachment downloads, git and LFS do not create a new session
	if !middleware.IsAPIPath(req) && !isAttachmentDownload(req) && !isGitRawOrAttachOrLFSPath(req) {
		if sess != nil && (sess.Get("uid") == nil || sess.Get("uid").(int64) != user.ID) {

			// Register last login.
			user.SetLastLogin()

			if err := user_model.UpdateUserCols(req.Context(), user, "last_login_unix"); err != nil {
				log.Error(fmt.Sprintf("ReverseProxy Authorization: error updating user last login time [user: %d]", user.ID))
			}

			// Initialize new session. Will set lang and CSRF cookies.
			handleSignIn(w, req, sess, user)

			log.Trace("ReverseProxy Authorization: Logged in user %-v", user)
		}

		// Unfortunatelly we cannot do redirect here (would break git HTTP requests) to
		// reload page with user locale so first page after login may be displayed in
		// wrong language. Language handling in SSO mode should be reconsidered
		// in future gitea versions.
	}

	store.GetData()["IsReverseProxy"] = true
	return user, nil
}

// isAutoRegisterAllowed checks if EnableReverseProxyAutoRegister setting is true
func (r *ReverseProxy) isAutoRegisterAllowed() bool {
	return setting.Service.EnableReverseProxyAutoRegister
}

// newUser creates a new user object for the purpose of automatic registration
// and populates its name and email with the information present in request headers.
func (r *ReverseProxy) newUser(req *http.Request) *user_model.User {
	username := r.getUserName(req)
	if len(username) == 0 {
		return nil
	}

	email := gouuid.New().String() + "@localhost"
	if setting.Service.EnableReverseProxyEmail {
		webAuthEmail := req.Header.Get(setting.ReverseProxyAuthEmail)
		if len(webAuthEmail) > 0 {
			email = webAuthEmail
		}
	}

	var fullname string
	if setting.Service.EnableReverseProxyFullName {
		fullname = req.Header.Get(setting.ReverseProxyAuthFullName)
	}

	user := &user_model.User{
		Name:     username,
		Email:    email,
		FullName: fullname,
	}

	overwriteDefault := user_model.CreateUserOverwriteOptions{
		IsActive: optional.Some(true),
	}

	// The first user created should be an admin.
	if user_model.CountUsers(req.Context(), nil) == 0 {
		user.IsAdmin = true
	}

	if err := user_model.CreateUser(req.Context(), user, &overwriteDefault); err != nil {
		// FIXME: should I create a system notice?
		log.Error("CreateUser: %v", err)
		return nil
	}

	return user
}
