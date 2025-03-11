// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package user

import (
	"context"
	"reflect"
	"slices"

	"code.gitea.io/gitea/models/moderation"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/xorm/names"
)

// UserData represents a trimmed down user that is used for preserving
// only the fields needed for abusive content reports (mainly string fields).
type UserData struct {
	Name        string
	FullName    string
	Email       string
	LoginName   string
	Location    string
	Website     string
	Pronouns    string
	Description string
	CreatedUnix timeutil.TimeStamp
	UpdatedUnix timeutil.TimeStamp
	// This field was intentionally renamed so that is not the same with the one from User struct.
	// If we keep it the same as in User, during login it might trigger the creation of a shadow copy.
	// TODO: Should we decide that this field is not that relevant for abuse reporting purposes, better remove it.
	LastLogin   timeutil.TimeStamp `json:"LastLoginUnix"`
	Avatar      string
	AvatarEmail string
}

// newUserData creates a trimmed down user to be used just to create a JSON structure
// (keeping only the fields relevant for moderation purposes)
func newUserData(user *User) UserData {
	return UserData{
		Name:        user.Name,
		FullName:    user.FullName,
		Email:       user.Email,
		LoginName:   user.LoginName,
		Location:    user.Location,
		Website:     user.Website,
		Pronouns:    user.Pronouns,
		Description: user.Description,
		CreatedUnix: user.CreatedUnix,
		UpdatedUnix: user.UpdatedUnix,
		LastLogin:   user.LastLoginUnix,
		Avatar:      user.Avatar,
		AvatarEmail: user.AvatarEmail,
	}
}

// IfNeededCreateShadowCopyForUser checks if for the given user there are any reports of abusive content submitted
// and if found a shadow copy of relevant user fields will be stored into DB and linked to the above report(s).
// This function should be called when a user is deleted or updated.
//
// For deletions alteredCols argument must be omitted.
//
// In case of updates it will first checks whether any of the columns being updated (alteredCols argument)
// is relevant for moderation purposes (i.e. included in the UserData struct).
func IfNeededCreateShadowCopyForUser(ctx context.Context, user *User, alteredCols ...string) error {
	// TODO: this can be triggered quite often (e.g. by routers/web/repo/middlewares.go SetDiffViewStyle())

	shouldCheckIfReported := len(alteredCols) == 0 // no columns being updated, therefore a deletion
	if !shouldCheckIfReported {
		// for updates we need to go further only if certain column are being changed
		mapper := new(names.GonicMapper)
		ucType := reflect.TypeOf(UserData{})

		for i := 0; i < ucType.NumField(); i++ {
			field := ucType.Field(i)
			// get the corresponding column name for a field name (e.g. FieldName -> field_name).
			colName := mapper.Obj2Table(field.Name)
			if shouldCheckIfReported = slices.Contains(alteredCols, colName); shouldCheckIfReported {
				break
			}
		}
	}

	if shouldCheckIfReported && moderation.IsReported(ctx, moderation.ReportedContentTypeUser, user.ID) {
		userData := newUserData(user)
		content, err := json.Marshal(userData)
		if err != nil {
			return err
		}
		return moderation.CreateShadowCopyForUser(ctx, user.ID, string(content))
	}

	return nil
}
