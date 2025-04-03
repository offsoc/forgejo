// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"context"

	"forgejo.org/models/moderation"
	"forgejo.org/modules/json"
	"forgejo.org/modules/timeutil"
)

// RepositoryData represents a trimmed down repository that is used for preserving
// only the fields needed for abusive content reports (mainly string fields).
type RepositoryData struct {
	OwnerID     int64
	OwnerName   string
	Name        string
	Description string
	Website     string
	Topics      []string
	Avatar      string
	CreatedUnix timeutil.TimeStamp
	UpdatedUnix timeutil.TimeStamp
}

// newRepositoryData creates a trimmed down repository to be used just to create a JSON structure
// (keeping only the fields relevant for moderation purposes)
func newRepositoryData(repo *Repository) RepositoryData {
	return RepositoryData{
		OwnerID:     repo.OwnerID,
		OwnerName:   repo.OwnerName,
		Name:        repo.Name,
		Description: repo.Description,
		Website:     repo.Website,
		Topics:      repo.Topics,
		Avatar:      repo.Avatar,
		CreatedUnix: repo.CreatedUnix,
		UpdatedUnix: repo.UpdatedUnix,
	}
}

// IfNeededCreateShadowCopyForRepository checks if for the given repository there are any reports of abusive content submitted
// and if found a shadow copy of relevant repository fields will be stored into DB and linked to the above report(s).
// This function should be called when a repository is deleted or updated.
func IfNeededCreateShadowCopyForRepository(ctx context.Context, repo *Repository, forUpdates bool) error {
	shadowCopyNeeded, err := moderation.IsShadowCopyNeeded(ctx, moderation.ReportedContentTypeRepository, repo.ID)
	if err != nil {
		return err
	}

	if shadowCopyNeeded {
		if forUpdates {
			// get the unmodified repository fields
			repo, err = GetRepositoryByID(ctx, repo.ID)
			if err != nil {
				return err
			}
		}
		repoData := newRepositoryData(repo)
		content, err := json.Marshal(repoData)
		if err != nil {
			return err
		}
		return moderation.CreateShadowCopyForRepository(ctx, repo.ID, string(content))
	}

	return nil
}
