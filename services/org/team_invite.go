// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"context"

	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/mailer"
)

// CreateTeamInvite make a persistent invite in db and mail it
func CreateTeamInvite(ctx context.Context, inviter *user_model.User, team *org_model.Team, uname string) error {
	invite, err := org_model.CreateTeamInvite(ctx, inviter, team, uname)
	if err != nil {
		return err
	}

	return mailer.MailTeamInvite(ctx, inviter, team, invite)
}
