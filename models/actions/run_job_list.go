// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/container"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

type ActionJobList []*ActionRunJob

func (jobs ActionJobList) GetRunIDs() []int64 {
	return container.FilterSlice(jobs, func(j *ActionRunJob) (int64, bool) {
		return j.RunID, j.RunID != 0
	})
}

func (jobs ActionJobList) LoadRuns(ctx context.Context, withRepo bool) error {
	runIDs := jobs.GetRunIDs()
	runs := make(map[int64]*ActionRun, len(runIDs))
	if err := db.GetEngine(ctx).In("id", runIDs).Find(&runs); err != nil {
		return err
	}
	for _, j := range jobs {
		if j.RunID > 0 && j.Run == nil {
			j.Run = runs[j.RunID]
		}
	}
	if withRepo {
		var runsList RunList = make([]*ActionRun, 0, len(runs))
		for _, r := range runs {
			runsList = append(runsList, r)
		}
		return runsList.LoadRepos(ctx)
	}
	return nil
}

func (jobs ActionJobList) LoadAttributes(ctx context.Context, withRepo bool) error {
	return jobs.LoadRuns(ctx, withRepo)
}

type FindRunJobOptions struct {
	db.ListOptions
	RunID         int64
	RepoID        int64
	OwnerID       int64
	CommitSHA     string
	Statuses      []Status
	UpdatedBefore timeutil.TimeStamp
	Events        []string // []webhook_module.HookEventType
	RunNumber     int64
}

func (opts FindRunJobOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RunID > 0 {
		cond = cond.And(builder.Eq{"run_id": opts.RunID})
	}
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.OwnerID > 0 {
		cond = cond.And(builder.Eq{"owner_id": opts.OwnerID})
	}
	if opts.CommitSHA != "" {
		cond = cond.And(builder.Eq{"commit_sha": opts.CommitSHA})
	}
	if len(opts.Statuses) > 0 {
		cond = cond.And(builder.In("status", opts.Statuses))
	}
	if opts.UpdatedBefore > 0 {
		cond = cond.And(builder.Lt{"updated": opts.UpdatedBefore})
	}
	if len(opts.Events) > 0 {
		cond = cond.And(builder.In("event", opts.Events))
	}
	if opts.RunNumber > 0 {
		cond = cond.And(builder.Eq{"`index`": opts.RunNumber})
	}
	return cond
}
