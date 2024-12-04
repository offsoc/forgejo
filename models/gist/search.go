// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"context"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"

	"xorm.io/builder"
)

type SearchGistOptions struct {
	db.ListOptions
	Actor     *user_model.User
	OwnerID   int64
	Keyword   string
	SortOrder string
}

// SearchGistByCondition returns Conditions for the given Options
func SearchGistCondition(opts *SearchGistOptions) builder.Cond {
	cond := builder.NewCond()

	if opts.Actor == nil {
		cond = cond.And(builder.Eq{"gist.visibility": GistVisibilityPublic})
	} else if !opts.Actor.IsAdmin {
		ownCond := builder.NewCond()
		ownCond = ownCond.And(builder.Neq{"gist.visibility": GistVisibilityPublic})
		ownCond = ownCond.And(builder.Eq{"gist.owner_id": opts.Actor.ID})

		privateCond := builder.NewCond()
		privateCond = privateCond.Or(builder.Eq{"gist.visibility": GistVisibilityPublic})
		privateCond = privateCond.Or(ownCond)

		cond = cond.And(privateCond)
	}

	if opts.OwnerID != 0 {
		cond = cond.And(builder.Eq{"gist.owner_id": opts.OwnerID})
	}

	if opts.Keyword != "" {
		cond = cond.And(db.BuildCaseInsensitiveLike("gist.name", opts.Keyword))
	}

	return cond
}

// Search Gists find Gists by the given Options
func SearchGist(ctx context.Context, opts *SearchGistOptions) (GistList, int64, error) {
	cond := SearchGistCondition(opts)

	sess := db.GetEngine(ctx)

	count, err := sess.Where(cond).Count(new(Gist))
	if err != nil {
		return nil, 0, err
	}

	if opts.SortOrder != "" {
		var orderBy string

		switch opts.SortOrder {
		case "newest":
			orderBy = "gist.updated_unix DESC"
		case "oldest":
			orderBy = "gist.updated_unix ASC"
		case "alphabetically":
			orderBy = "gist.name ASC"
		case "reversealphabetically":
			orderBy = "gist.name DESC"
		}

		if orderBy != "" {
			sess.OrderBy(orderBy)
		} else {
			sess.OrderBy("gist.updated_unix DESC")
		}
	} else {
		sess.OrderBy("gist.updated_unix DESC")
	}

	sess = sess.Where(cond)

	if opts.PageSize > 0 {
		sess = sess.Limit(opts.PageSize, (opts.Page-1)*opts.PageSize)
	}

	gistList := make(GistList, 0)
	err = sess.Find(&gistList)
	if err != nil {
		return nil, 0, err
	}

	return gistList, count, nil
}

// CountGists return a number of all Gists that match the Options
func CountGist(ctx context.Context, opts *SearchGistOptions) (int64, error) {
	cond := SearchGistCondition(opts)
	return db.GetEngine(ctx).Where(cond).Count(new(Gist))
}
