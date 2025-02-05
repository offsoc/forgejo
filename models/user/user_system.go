// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
)

const (
	GhostUserID        = -1
	GhostUserName      = "Ghost"
	GhostUserLowerName = "ghost"
)

// NewGhostUser creates and returns a fake user for someone has deleted their account.
func NewGhostUser() *User {
	return &User{
		ID:        GhostUserID,
		Name:      GhostUserName,
		LowerName: GhostUserLowerName,
	}
}

// IsGhost check if user is fake user for a deleted account
func (u *User) IsGhost() bool {
	if u == nil {
		return false
	}
	return u.ID == GhostUserID && u.Name == GhostUserName
}

// NewReplaceUser creates and returns a fake user for external user
func NewReplaceUser(name string) *User {
	return &User{
		ID:        0,
		Name:      name,
		LowerName: strings.ToLower(name),
	}
}

const (
	ActionsUserID   = -2
	ActionsUserName = "forgejo-actions"
	ActionsFullName = "Forgejo Actions"
	ActionsEmail    = "noreply@forgejo.org"
)

// NewActionsUser creates and returns a fake user for running the actions.
func NewActionsUser() *User {
	return &User{
		ID:                      ActionsUserID,
		Name:                    ActionsUserName,
		LowerName:               ActionsUserName,
		IsActive:                true,
		FullName:                ActionsFullName,
		Email:                   ActionsEmail,
		KeepEmailPrivate:        true,
		LoginName:               ActionsUserName,
		Type:                    UserTypeIndividual,
		AllowCreateOrganization: true,
		Visibility:              structs.VisibleTypePublic,
	}
}

func (u *User) IsActions() bool {
	return u != nil && u.ID == ActionsUserID
}

const (
	APActorUserID   = -3
	APActorUserName = "actor"
	APActorEmail    = "noreply@forgejo.org"
)

func NewAPActorUser() *User {
	return &User{
		ID:               APActorUserID,
		Name:             APActorUserName,
		LowerName:        APActorUserName,
		IsActive:         true,
		Email:            APActorEmail,
		KeepEmailPrivate: true,
		LoginName:        APActorUserName,
		Type:             UserTypeIndividual,
		Visibility:       structs.VisibleTypePublic,
	}
}

func APActorUserAPActorID() string {
	path, _ := url.JoinPath(setting.AppURL, "/api/v1/activitypub/actor")
	return path
}

func (u *User) IsAPActor() bool {
	return u != nil && u.ID == APActorUserID
}
