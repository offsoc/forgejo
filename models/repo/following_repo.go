// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"forgejo.org/modules/validation"
)

// FollowingRepo represents a federated Repository Actor connected with a local Repo
type FollowingRepo struct {
	ID               int64  `xorm:"pk autoincr"`
	RepoID           int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	ExternalID       string `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	URI              string
}

func NewFollowingRepo(repoID int64, externalID string, federationHostID int64, uri string) (FollowingRepo, error) {
	result := FollowingRepo{
		RepoID:           repoID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
		URI:              uri,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FollowingRepo{}, err
	}
	return result, nil
}

func (user FollowingRepo) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.RepoID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	result = append(result, validation.ValidateNotEmpty(user.URI, "Uri")...)
	return result
}
