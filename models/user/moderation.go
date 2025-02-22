// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package user

import (
	"context"

	"code.gitea.io/gitea/models/moderation"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/timeutil"
)

// UserData represents a trimmed down user that is used for preserving only the fields needed for abusive content reports.
type UserData struct {
	Name                   string
	FullName               string
	Email                  string
	LoginName              string
	Location               string
	Website                string
	Pronouns               string
	Description            string
	CreatedUnix            timeutil.TimeStamp
	UpdatedUnix            timeutil.TimeStamp
	LastLoginUnix          timeutil.TimeStamp
	Avatar                 string
	AvatarEmail            string
	NormalizedFederatedURI string
}

// newUserData creates a trimmed down user to be used just to create a JSON structure
// (keeping only the fields relevant for moderation purposes)
func newUserData(user *User) UserData {
	return UserData{
		Name:          user.Name,
		FullName:      user.FullName,
		Email:         user.Email,
		LoginName:     user.LoginName,
		Location:      user.Location,
		Website:       user.Website,
		Pronouns:      user.Pronouns,
		Description:   user.Description,
		CreatedUnix:   user.CreatedUnix,
		UpdatedUnix:   user.UpdatedUnix,
		LastLoginUnix: user.LastLoginUnix,
		Avatar:        user.Avatar,
		AvatarEmail:   user.AvatarEmail,
	}
}

// IfNeededCreateShadowCopyForUser checks if for the given user there are any reports of abusive content submitted
// and if found a shadow copy of relevant user fields will be stored into DB and linked to the above report(s).
// This function should be called when a user is deleted or updated.
func IfNeededCreateShadowCopyForUser(ctx context.Context, user *User) error {
	if moderation.IsReported(ctx, moderation.ReportedContentTypeUser, user.ID) {
		userContent := newUserData(user)
		content, _ := json.Marshal(userContent)
		return moderation.CreateShadowCopyForUser(ctx, user.ID, string(content))
	}

	return nil
}
