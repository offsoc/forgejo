// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"

	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/translation"
)

const (
	tplTeamInviteMail base.TplName = "team_invite"
)

// MailTeamInvite sends team invites
func MailTeamInvite(ctx context.Context, inviter *user_model.User, team *org_model.Team, invite *org_model.TeamInvite) error {
	if setting.MailService == nil {
		return nil
	}

	org, err := user_model.GetUserByID(ctx, team.OrgID)
	if err != nil {
		return err
	}

	locale := translation.NewLocale(inviter.Language)

	// check if a user with this email already exists
	user, err := user_model.GetUserByEmail(ctx, invite.Email)
	if err != nil && !user_model.IsErrUserNotExist(err) {
		return err
	} else if user != nil && user.ProhibitLogin {
		return errors.New("login is prohibited for the invited user")
	}

	inviteRedirect := url.QueryEscape(fmt.Sprintf("/org/invite/%s", invite.Token))
	inviteURL := fmt.Sprintf("%suser/sign_up?redirect_to=%s", setting.AppURL, inviteRedirect)

	if (err == nil && user != nil) || setting.Service.DisableRegistration || setting.Service.AllowOnlyExternalRegistration {
		// user account exists or registration disabled
		inviteURL = fmt.Sprintf("%suser/login?redirect_to=%s", setting.AppURL, inviteRedirect)
	}

	subject := locale.TrString("mail.team_invite.subject", inviter.DisplayName(), org.DisplayName())
	mailMeta := map[string]any{
		"locale":       locale,
		"Inviter":      inviter,
		"Organization": org,
		"Team":         team,
		"Invite":       invite,
		"Subject":      subject,
		"InviteURL":    inviteURL,
	}

	var mailBody bytes.Buffer
	if err := bodyTemplates.ExecuteTemplate(&mailBody, string(tplTeamInviteMail), mailMeta); err != nil {
		log.Error("ExecuteTemplate [%s]: %v", string(tplTeamInviteMail)+"/body", err)
		return err
	}

	msg := NewMessage(invite.Email, subject, mailBody.String())
	msg.Info = subject

	SendAsync(msg)

	return nil
}
