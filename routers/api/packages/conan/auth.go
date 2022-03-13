// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package conan

import (
	"net/http"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/packages"
)

type Auth struct{}

func (a *Auth) Name() string {
	return "conan"
}

func (a *Auth) Verify(req *http.Request, w http.ResponseWriter, store auth.DataStore, sess auth.SessionStore) *user_model.User {
	uid, err := packages.ParseAuthorizationToken(req)
	if err != nil {
		log.Trace("ParseAuthorizationToken: %v", err)
		return nil
	}

	if uid == 0 {
		return nil
	}

	u, err := user_model.GetUserByID(uid)
	if err != nil {
		log.Error("GetUserByID:  %v", err)
		return nil
	}

	return u
}
