// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/models/perm"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/structs"

	"xorm.io/builder"
)

// SearchOrganizationsOptions options to filter organizations
type SearchOrganizationsOptions struct {
	db.ListOptions
	All bool
}

// FindOrgOptions finds orgs options
type FindOrgOptions struct {
	db.ListOptions
	UserID         int64
	IncludeLimited bool
	IncludePrivate bool
}

func queryUserOrgIDs(userID int64, includePrivate bool) *builder.Builder {
	cond := builder.Eq{"uid": userID}
	if !includePrivate {
		cond["is_public"] = true
	}
	return builder.Select("org_id").From("org_user").Where(cond)
}

func (opts FindOrgOptions) ToConds() builder.Cond {
	var cond builder.Cond = builder.Eq{"`user`.`type`": user_model.UserTypeOrganization}
	if opts.UserID > 0 {
		cond = cond.And(builder.In("`user`.`id`", queryUserOrgIDs(opts.UserID, opts.IncludePrivate)))
	}
	if !opts.IncludePrivate {
		if !opts.IncludeLimited {
			cond = cond.And(builder.Eq{"`user`.visibility": structs.VisibleTypePublic})
		} else {
			cond = cond.And(builder.In("`user`.visibility", structs.VisibleTypePublic, structs.VisibleTypeLimited))
		}
	}
	return cond
}

func (opts FindOrgOptions) ToOrders() string {
	return "`user`.lower_name ASC"
}

// GetOrgsCanCreateRepoByUserID returns a list of organizations where given user ID
// are allowed to create repos.
func GetOrgsCanCreateRepoByUserID(ctx context.Context, userID int64) ([]*Organization, error) {
	orgs := make([]*Organization, 0, 10)

	return orgs, db.GetEngine(ctx).Select("DISTINCT `user`.id, `user`.*").Table("`user`").
		Join("INNER", "`team_user`", "`team_user`.org_id = `user`.id").
		Join("INNER", "`team`", "`team`.id = `team_user`.team_id").
		Where(builder.Eq{"`team_user`.uid": userID}).
		And(builder.Eq{"`team`.authorize": perm.AccessModeOwner}.Or(builder.Eq{"`team`.can_create_org_repo": true})).
		Asc("`user`.name").
		Find(&orgs)
}

// MinimalOrg represents a simple organization with only the needed columns
type MinimalOrg = Organization

// GetUserOrgsList returns all organizations the given user has access to
func GetUserOrgsList(ctx context.Context, user *user_model.User) ([]*MinimalOrg, error) {
	schema, err := db.TableInfo(new(user_model.User))
	if err != nil {
		return nil, err
	}

	outputCols := []string{
		"id",
		"name",
		"full_name",
		"visibility",
		"avatar",
		"avatar_email",
		"use_custom_avatar",
	}

	selectColumns := &strings.Builder{}
	for i, col := range outputCols {
		fmt.Fprintf(selectColumns, "`%s`.%s", schema.Name, col)
		if i < len(outputCols)-1 {
			selectColumns.WriteString(", ")
		}
	}
	columnsStr := selectColumns.String()

	var orgs []*MinimalOrg
	if err := db.GetEngine(ctx).Select(columnsStr).
		Table("user").
		Where(builder.In("`user`.`id`", queryUserOrgIDs(user.ID, true))).
		OrderBy("`user`.lower_name ASC").
		Find(&orgs); err != nil {
		return nil, err
	}

	type orgCount struct {
		OrgID     int64
		RepoCount int
	}
	var orgCounts []orgCount
	if err := db.GetEngine(ctx).
		Select("owner_id AS org_id, COUNT(DISTINCT(repository.id)) as repo_count").
		Table("repository").
		Join("INNER", "org_user", "owner_id = org_user.org_id").
		Where("org_user.uid = ?", user.ID).
		And(builder.Or(
			builder.Eq{"repository.is_private": false},
			builder.In("repository.id", builder.Select("repo_id").From("team_repo").
				InnerJoin("team_user", "team_user.team_id = team_repo.team_id").
				Where(builder.Eq{"team_user.uid": user.ID})),
			builder.In("repository.id", builder.Select("repo_id").From("collaboration").
				Where(builder.Eq{"user_id": user.ID})),
		)).
		GroupBy("owner_id").Find(&orgCounts); err != nil {
		return nil, err
	}

	orgCountMap := make(map[int64]int, len(orgCounts))
	for _, orgCount := range orgCounts {
		orgCountMap[orgCount.OrgID] = orgCount.RepoCount
	}

	for _, org := range orgs {
		org.NumRepos = orgCountMap[org.ID]
	}

	return orgs, nil
}
