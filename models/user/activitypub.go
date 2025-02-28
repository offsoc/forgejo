// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"
	"net/url"

	"forgejo.org/models/db"
	"forgejo.org/modules/setting"
)

// APActorID returns the IRI to the api endpoint of the user
func (u *User) APActorID() string {
	if u.ID == APActorUserID {
		return fmt.Sprintf("%vapi/v1/activitypub/actor", setting.AppURL)
	}

	return fmt.Sprintf("%vapi/v1/activitypub/user-id/%v", setting.AppURL, url.PathEscape(fmt.Sprintf("%v", u.ID)))
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

	return user, nil
}
