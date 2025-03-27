// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package remote

import (
	"context"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/services/auth/source/oauth2"
	remote_source "forgejo.org/services/auth/source/remote"
)

type Reason int

const (
	ReasonNoMatch Reason = iota
	ReasonNotAuth2
	ReasonBadAuth2
	ReasonLoginNameNotExists
	ReasonNotRemote
	ReasonEmailIsSet
	ReasonNoSource
	ReasonSourceWrongType
	ReasonCanPromote
	ReasonPromoted
	ReasonUpdateFail
	ReasonErrorLoginName
	ReasonErrorGetSource
)

func NewReason(level log.Level, reason Reason, message string, args ...any) Reason {
	log.Log(1, level, message, args...)
	return reason
}

func getUsersByLoginName(ctx context.Context, name string) ([]*user_model.User, error) {
	if len(name) == 0 {
		return nil, user_model.ErrUserNotExist{Name: name}
	}

	users := make([]*user_model.User, 0, 5)

	return users, db.GetEngine(ctx).
		Table("user").
		Where("login_name = ? AND login_type = ? AND type = ?", name, auth_model.Remote, user_model.UserTypeRemoteUser).
		Find(&users)
}

// The remote user has:
//
//	Type        UserTypeRemoteUser
//	LogingType  Remote
//	LoginName   set to the unique identifier of the originating authentication source
//	LoginSource set to the Remote source that can be matched against an OAuth2 source
//
// If the source from which an authentication happens is OAuth2, an existing
// remote user will be promoted to an OAuth2 user provided:
//
//	user.LoginName is the same as goth.UserID (argument loginName)
//	user.LoginSource has a MatchingSource equals to the name of the OAuth2 provider
//
// Once promoted, the user will be logged in without further interaction from the
// user and will own all repositories, issues, etc. associated with it.
func MaybePromoteRemoteUser(ctx context.Context, source *auth_model.Source, loginName, email string) (promoted bool, reason Reason, err error) {
	user, reason, err := getRemoteUserToPromote(ctx, source, loginName, email)
	if err != nil || user == nil {
		return false, reason, err
	}
	promote := &user_model.User{
		ID:          user.ID,
		Type:        user_model.UserTypeIndividual,
		Email:       email,
		LoginSource: source.ID,
		LoginType:   source.Type,
	}
	reason = NewReason(log.DEBUG, ReasonPromoted, "promote user %v: LoginName %v => %v, LoginSource %v => %v, LoginType %v => %v, Email %v => %v", user.ID, user.LoginName, promote.LoginName, user.LoginSource, promote.LoginSource, user.LoginType, promote.LoginType, user.Email, promote.Email)
	if err := user_model.UpdateUserCols(ctx, promote, "type", "email", "login_source", "login_type"); err != nil {
		return false, ReasonUpdateFail, err
	}
	return true, reason, nil
}

func getRemoteUserToPromote(ctx context.Context, source *auth_model.Source, loginName, email string) (*user_model.User, Reason, error) { //nolint:unparam
	if !source.IsOAuth2() {
		return nil, NewReason(log.DEBUG, ReasonNotAuth2, "source %v is not OAuth2", source), nil
	}
	oauth2Source, ok := source.Cfg.(*oauth2.Source)
	if !ok {
		return nil, NewReason(log.ERROR, ReasonBadAuth2, "source claims to be OAuth2 but is not"), nil
	}

	users, err := getUsersByLoginName(ctx, loginName)
	if err != nil {
		return nil, NewReason(log.ERROR, ReasonErrorLoginName, "getUserByLoginName('%s') %v", loginName, err), err
	}
	if len(users) == 0 {
		return nil, NewReason(log.ERROR, ReasonLoginNameNotExists, "no user with LoginType UserTypeRemoteUser and LoginName '%s'", loginName), nil
	}

	reason := ReasonNoSource
	for _, u := range users {
		userSource, err := auth_model.GetSourceByID(ctx, u.LoginSource)
		if err != nil {
			if auth_model.IsErrSourceNotExist(err) {
				reason = NewReason(log.DEBUG, ReasonNoSource, "source id = %v for user %v not found %v", u.LoginSource, u.ID, err)
				continue
			}
			return nil, NewReason(log.ERROR, ReasonErrorGetSource, "GetSourceByID('%s') %v", u.LoginSource, err), err
		}
		if u.Email != "" {
			reason = NewReason(log.DEBUG, ReasonEmailIsSet, "the user email is already set to '%s'", u.Email)
			continue
		}
		remoteSource, ok := userSource.Cfg.(*remote_source.Source)
		if !ok {
			reason = NewReason(log.DEBUG, ReasonSourceWrongType, "expected a remote source but got %T %v", userSource, userSource)
			continue
		}

		if oauth2Source.Provider != remoteSource.MatchingSource {
			reason = NewReason(log.DEBUG, ReasonNoMatch, "skip OAuth2 source %s because it is different from %s which is the expected match for the remote source %s", oauth2Source.Provider, remoteSource.MatchingSource, remoteSource.URL)
			continue
		}

		return u, ReasonCanPromote, nil
	}

	return nil, reason, nil
}
