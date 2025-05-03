// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"net/url"

	"forgejo.org/modules/setting"
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
