// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"net/http"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/services/auth"
	"forgejo.org/services/packages"
)

var _ auth.Method = &Auth{}

type Auth struct{}

func (a *Auth) Name() string {
	return "container"
}

// Verify extracts the user from the Bearer token
// If it's an anonymous session a ghost user is returned
func (a *Auth) Verify(req *http.Request, w http.ResponseWriter, store auth.DataStore, sess auth.SessionStore) (*user_model.User, error) {
	uid, scope, err := packages.ParseAuthorizationToken(req)
	if err != nil {
		log.Trace("ParseAuthorizationToken: %v", err)
		return nil, err
	}

	if uid == 0 {
		return nil, nil
	}

	// Propagate scope of the authorization token.
	if scope != "" {
		store.GetData()["IsApiToken"] = true
		store.GetData()["ApiTokenScope"] = scope
	}

	u, err := user_model.GetPossibleUserByID(req.Context(), uid)
	if err != nil {
		log.Error("GetPossibleUserByID:  %v", err)
		return nil, err
	}

	return u, nil
}
