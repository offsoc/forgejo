// Copyright 2024 The Forgejo Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"errors"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	system_model "forgejo.org/models/system"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	notify_service "forgejo.org/services/notify"
	pull_service "forgejo.org/services/pull"
)

// WebSearchRepository represents a repository returned by web search
type WebSearchRepository struct {
	Repository               *structs.Repository `json:"repository"`
	LatestCommitStatus       *git.CommitStatus   `json:"latest_commit_status"`
	LocaleLatestCommitStatus string              `json:"locale_latest_commit_status"`
}

// WebSearchResults results of a successful web search
type WebSearchResults struct {
	OK   bool                   `json:"ok"`
	Data []*WebSearchRepository `json:"data"`
}

// CreateRepository creates a repository for the user/organization.
func CreateRepository(ctx context.Context, doer, owner *user_model.User, opts CreateRepoOptions) (*repo_model.Repository, error) {
	repo, err := CreateRepositoryDirectly(ctx, doer, owner, opts)
	if err != nil {
		// No need to rollback here we should do this in CreateRepository...
		return nil, err
	}

	notify_service.CreateRepository(ctx, doer, owner, repo)

	return repo, nil
}

// DeleteRepository deletes a repository for a user or organization.
func DeleteRepository(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, notify bool) error {
	if err := pull_service.CloseRepoBranchesPulls(ctx, doer, repo); err != nil {
		log.Error("CloseRepoBranchesPulls failed: %v", err)
	}

	if notify {
		// If the repo itself has webhooks, we need to trigger them before deleting it...
		notify_service.DeleteRepository(ctx, doer, repo)
	}

	return DeleteRepositoryDirectly(ctx, doer, repo.ID)
}

// PushCreateRepo creates a repository when a new repository is pushed to an appropriate namespace
func PushCreateRepo(ctx context.Context, authUser, owner *user_model.User, repoName string) (*repo_model.Repository, error) {
	if !authUser.IsAdmin {
		if owner.IsOrganization() {
			if ok, err := organization.CanCreateOrgRepo(ctx, owner.ID, authUser.ID); err != nil {
				return nil, err
			} else if !ok {
				return nil, errors.New("cannot push-create repository for org")
			}
		} else if authUser.ID != owner.ID {
			return nil, errors.New("cannot push-create repository for another user")
		}
	}

	repo, err := CreateRepository(ctx, authUser, owner, CreateRepoOptions{
		Name:      repoName,
		IsPrivate: setting.Repository.DefaultPushCreatePrivate || setting.Repository.ForcePrivate,
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// Init start repository service
func Init(ctx context.Context) error {
	if err := repo_module.LoadRepoConfig(); err != nil {
		return err
	}
	system_model.RemoveAllWithNotice(ctx, "Clean up temporary repository uploads", setting.Repository.Upload.TempPath)
	system_model.RemoveAllWithNotice(ctx, "Clean up temporary repositories", repo_module.LocalCopyPath())
	if err := initPushQueue(); err != nil {
		return err
	}
	return initBranchSyncQueue(graceful.GetManager().ShutdownContext())
}

// UpdateRepository updates a repository
func UpdateRepository(ctx context.Context, repo *repo_model.Repository, visibilityChanged bool) (err error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err = repo_module.UpdateRepository(ctx, repo, visibilityChanged); err != nil {
		return fmt.Errorf("updateRepository: %w", err)
	}

	return committer.Commit()
}

// LinkedRepository returns the linked repo if any
func LinkedRepository(ctx context.Context, a *repo_model.Attachment) (*repo_model.Repository, unit.Type, error) {
	if a.IssueID != 0 {
		iss, err := issues_model.GetIssueByID(ctx, a.IssueID)
		if err != nil {
			return nil, unit.TypeIssues, err
		}
		repo, err := repo_model.GetRepositoryByID(ctx, iss.RepoID)
		unitType := unit.TypeIssues
		if iss.IsPull {
			unitType = unit.TypePullRequests
		}
		return repo, unitType, err
	} else if a.ReleaseID != 0 {
		rel, err := repo_model.GetReleaseByID(ctx, a.ReleaseID)
		if err != nil {
			return nil, unit.TypeReleases, err
		}
		repo, err := repo_model.GetRepositoryByID(ctx, rel.RepoID)
		return repo, unit.TypeReleases, err
	}
	return nil, -1, nil
}
