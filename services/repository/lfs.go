// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"
	"time"

	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/lfs"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
)

// GarbageCollectLFSMetaObjectsOptions provides options for GarbageCollectLFSMetaObjects function
type GarbageCollectLFSMetaObjectsOptions struct {
	LogDetail               func(format string, v ...any)
	AutoFix                 bool
	OlderThan               time.Time
	UpdatedLessRecentlyThan time.Time
}

// GarbageCollectLFSMetaObjects garbage collects LFS objects for all repositories
func GarbageCollectLFSMetaObjects(ctx context.Context, opts GarbageCollectLFSMetaObjectsOptions) error {
	log.Trace("Doing: GarbageCollectLFSMetaObjects")
	defer log.Trace("Finished: GarbageCollectLFSMetaObjects")

	if opts.LogDetail == nil {
		opts.LogDetail = log.Debug
	}

	if !setting.LFS.StartServer {
		opts.LogDetail("LFS support is disabled")
		return nil
	}

	return git_model.IterateRepositoryIDsWithLFSMetaObjects(ctx, func(ctx context.Context, repoID, count int64) error {
		repo, err := repo_model.GetRepositoryByID(ctx, repoID)
		if err != nil {
			return err
		}

		return GarbageCollectLFSMetaObjectsForRepo(ctx, repo, opts)
	})
}

// GarbageCollectLFSMetaObjectsForRepo garbage collects LFS objects for a specific repository
func GarbageCollectLFSMetaObjectsForRepo(ctx context.Context, repo *repo_model.Repository, opts GarbageCollectLFSMetaObjectsOptions) error {
	opts.LogDetail("Checking %s", repo.FullName())
	total, orphaned, collected, deleted := int64(0), 0, 0, 0
	defer func() {
		if orphaned == 0 {
			opts.LogDetail("Found %d total LFSMetaObjects in %s", total, repo.FullName())
		} else if !opts.AutoFix {
			opts.LogDetail("Found %d/%d orphaned LFSMetaObjects in %s", orphaned, total, repo.FullName())
		} else {
			opts.LogDetail("Collected %d/%d orphaned/%d total LFSMetaObjects in %s. %d removed from storage.", collected, orphaned, total, repo.FullName(), deleted)
		}
	}()

	gitRepo, err := gitrepo.OpenRepository(ctx, repo)
	if err != nil {
		log.Error("Unable to open git repository %s: %v", repo.FullName(), err)
		return err
	}
	defer gitRepo.Close()

	store := lfs.NewContentStore()
	objectFormat := git.ObjectFormatFromName(repo.ObjectFormatName)

	err = git_model.IterateLFSMetaObjectsForRepo(ctx, repo.ID, func(ctx context.Context, metaObject *git_model.LFSMetaObject) error {
		total++
		pointerSha := git.ComputeBlobHash(objectFormat, []byte(metaObject.StringContent()))

		if gitRepo.IsObjectExist(pointerSha.String()) {
			return git_model.MarkLFSMetaObject(ctx, metaObject.ID)
		}
		orphaned++

		if !opts.AutoFix {
			return nil
		}
		// Non-existent pointer file
		_, err = git_model.RemoveLFSMetaObjectByOidFn(ctx, repo.ID, metaObject.Oid, func(count int64) error {
			if count > 0 {
				return nil
			}

			if err := store.Delete(metaObject.RelativePath()); err != nil {
				log.Error("Unable to remove lfs metaobject %s from store: %v", metaObject.Oid, err)
			}
			deleted++
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to remove meta-object %s in %s: %w", metaObject.Oid, repo.FullName(), err)
		}
		collected++

		return nil
	}, &git_model.IterateLFSMetaObjectsForRepoOptions{
		// Only attempt to garbage collect lfs meta objects older than a week as the order of git lfs upload
		// and git object upload is not necessarily guaranteed. It's possible to imagine a situation whereby
		// an LFS object is uploaded but the git branch is not uploaded immediately, or there are some rapid
		// changes in new branches that might lead to lfs objects becoming temporarily unassociated with git
		// objects.
		//
		// It is likely that a week is potentially excessive but it should definitely be enough that any
		// unassociated LFS object is genuinely unassociated.
		OlderThan:               timeutil.TimeStamp(opts.OlderThan.Unix()),
		UpdatedLessRecentlyThan: timeutil.TimeStamp(opts.UpdatedLessRecentlyThan.Unix()),
	})
	if err != nil {
		return err
	}
	return nil
}
