// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mirror

import (
	"context"
	"errors"

	quota_model "forgejo.org/models/quota"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/setting"
)

// doMirrorSync causes this request to mirror itself
func doMirrorSync(ctx context.Context, req *SyncRequest) {
	if req.ReferenceID == 0 {
		log.Warn("Skipping mirror sync request, no mirror ID was specified")
		return
	}
	switch req.Type {
	case PushMirrorType:
		_ = SyncPushMirror(ctx, req.ReferenceID)
	case PullMirrorType:
		_ = SyncPullMirror(ctx, req.ReferenceID)
	default:
		log.Error("Unknown Request type in queue: %v for MirrorID[%d]", req.Type, req.ReferenceID)
	}
}

var errLimit = errors.New("reached limit")

// Update checks and updates mirror repositories.
func Update(ctx context.Context, pullLimit, pushLimit int) error {
	if !setting.Mirror.Enabled {
		log.Warn("Mirror feature disabled, but cron job enabled: skip update")
		return nil
	}
	log.Trace("Doing: Update")

	handler := func(bean any) error {
		var repo *repo_model.Repository
		var mirrorType SyncType
		var referenceID int64

		if m, ok := bean.(*repo_model.Mirror); ok {
			if m.GetRepository(ctx) == nil {
				log.Error("Disconnected mirror found: %d", m.ID)
				return nil
			}
			repo = m.Repo
			mirrorType = PullMirrorType
			referenceID = m.RepoID
		} else if m, ok := bean.(*repo_model.PushMirror); ok {
			if m.GetRepository(ctx) == nil {
				log.Error("Disconnected push-mirror found: %d", m.ID)
				return nil
			}
			repo = m.Repo
			mirrorType = PushMirrorType
			referenceID = m.ID
		} else {
			log.Error("Unknown bean: %v", bean)
			return nil
		}

		// Check we've not been cancelled
		select {
		case <-ctx.Done():
			return errors.New("aborted")
		default:
		}

		// Check if the repo's owner is over quota, for pull mirrors
		if mirrorType == PullMirrorType {
			ok, err := quota_model.EvaluateForUser(ctx, repo.OwnerID, quota_model.LimitSubjectSizeReposAll)
			if err != nil {
				log.Error("quota_model.EvaluateForUser: %v", err)
				return err
			}
			if !ok {
				log.Trace("Owner quota exceeded for %-v, not syncing", repo)
				return nil
			}
		}

		// Push to the Queue
		if err := PushToQueue(mirrorType, referenceID); err != nil {
			if err == queue.ErrAlreadyInQueue {
				if mirrorType == PushMirrorType {
					log.Trace("PushMirrors for %-v already queued for sync", repo)
				} else {
					log.Trace("PullMirrors for %-v already queued for sync", repo)
				}
				return nil
			}
			return err
		}
		return nil
	}

	pullMirrorsRequested := 0
	if pullLimit != 0 {
		if err := repo_model.MirrorsIterate(ctx, pullLimit, func(_ int, bean any) error {
			if err := handler(bean); err != nil {
				return err
			}
			pullMirrorsRequested++
			return nil
		}); err != nil && err != errLimit {
			log.Error("MirrorsIterate: %v", err)
			return err
		}
	}

	pushMirrorsRequested := 0
	if pushLimit != 0 {
		if err := repo_model.PushMirrorsIterate(ctx, pushLimit, func(idx int, bean any) error {
			if err := handler(bean); err != nil {
				return err
			}
			pushMirrorsRequested++
			return nil
		}); err != nil && err != errLimit {
			log.Error("PushMirrorsIterate: %v", err)
			return err
		}
	}
	log.Trace("Finished: Update: %d pull mirrors and %d push mirrors queued", pullMirrorsRequested, pushMirrorsRequested)
	return nil
}

func queueHandler(items ...*SyncRequest) []*SyncRequest {
	for _, req := range items {
		doMirrorSync(graceful.GetManager().ShutdownContext(), req)
	}
	return nil
}

// InitSyncMirrors initializes a go routine to sync the mirrors
func InitSyncMirrors() {
	StartSyncMirrors(queueHandler)
}
