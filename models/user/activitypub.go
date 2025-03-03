// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"
	"net/url"

	"forgejo.org/models/db"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/validation"
)

// APActorID returns the IRI to the api endpoint of the user
func (u *User) APActorID() string {
	if u.IsAPServerActor() {
		return fmt.Sprintf("%sapi/v1/activitypub/actor", setting.AppURL)
	}

	return fmt.Sprintf("%sapi/v1/activitypub/user-id/%s", setting.AppURL, url.PathEscape(fmt.Sprintf("%d", u.ID)))
}

// APActorKeyID returns the ID of the user's public key
func (u *User) APActorKeyID() string {
	return u.APActorID() + "#main-key"
}

func GetUserByFederatedURI(ctx context.Context, federatedURI string) (*User, error) {
	user := new(User)
	has, err := db.GetEngine(ctx).Where("normalized_federated_uri=?", federatedURI).Get(user)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	if res, err := validation.IsValid(*user); !res {
		return nil, err
	}

	return user, nil
}
