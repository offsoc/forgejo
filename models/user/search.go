// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/container"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/structs"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// SearchUserOptions contains the options for searching
type SearchUserOptions struct {
	db.ListOptions

	Keyword       string
	Type          UserType
	UID           int64
	LoginName     string                 // this option should be used only for admin user
	SourceID      optional.Option[int64] // this option should be used only for admin user
	OrderBy       db.SearchOrderBy
	Visible       []structs.VisibleType
	Actor         *User // The user doing the search
	SearchByEmail bool  // Search by email as well as username/full name

	SupportedSortOrders container.Set[string] // if not nil, only allow to use the sort orders in this set

	IsActive           optional.Option[bool]
	IsAdmin            optional.Option[bool]
	IsRestricted       optional.Option[bool]
	IsTwoFactorEnabled optional.Option[bool]
	IsProhibitLogin    optional.Option[bool]
	IncludeReserved    bool

	Load2FAStatus     bool
	ExtraParamStrings map[string]string
}

func (opts *SearchUserOptions) toSearchQueryBase(ctx context.Context) *xorm.Session {
	var cond builder.Cond
	if opts.Type == UserTypeIndividual {
		cond = builder.In("type", UserTypeIndividual, UserTypeRemoteUser)
	} else {
		cond = builder.Eq{"type": opts.Type}
	}
	if opts.IncludeReserved {
		switch opts.Type {
		case UserTypeIndividual:
			cond = cond.Or(builder.Eq{"type": UserTypeUserReserved}).Or(
				builder.Eq{"type": UserTypeBot},
			).Or(
				builder.Eq{"type": UserTypeRemoteUser},
			)
		case UserTypeOrganization:
			cond = cond.Or(builder.Eq{"type": UserTypeOrganizationReserved})
		}
	}

	if len(opts.Keyword) > 0 {
		lowerKeyword := strings.ToLower(opts.Keyword)
		keywordCond := builder.Or(
			builder.Like{"lower_name", lowerKeyword},
			builder.Like{"LOWER(full_name)", lowerKeyword},
		)
		if opts.SearchByEmail {
			var emailCond builder.Cond
			emailCond = builder.Like{"LOWER(email)", lowerKeyword}
			if opts.Actor == nil {
				emailCond = emailCond.And(builder.Eq{"keep_email_private": false})
			} else if !opts.Actor.IsAdmin {
				emailCond = emailCond.And(
					builder.Or(
						builder.Eq{"keep_email_private": false},
						builder.Eq{"id": opts.Actor.ID},
					),
				)
			}
			keywordCond = keywordCond.Or(emailCond)
		}

		cond = cond.And(keywordCond)
	}

	// If visibility filtered
	if len(opts.Visible) > 0 {
		cond = cond.And(builder.In("visibility", opts.Visible))
	}

	cond = cond.And(BuildCanSeeUserCondition(opts.Actor))

	if opts.UID > 0 {
		cond = cond.And(builder.Eq{"id": opts.UID})
	}

	if opts.SourceID.Has() {
		cond = cond.And(builder.Eq{"login_source": opts.SourceID.Value()})
	}
	if opts.LoginName != "" {
		cond = cond.And(builder.Eq{"login_name": opts.LoginName})
	}

	if opts.IsActive.Has() {
		cond = cond.And(builder.Eq{"is_active": opts.IsActive.Value()})
	}

	if opts.IsAdmin.Has() {
		cond = cond.And(builder.Eq{"is_admin": opts.IsAdmin.Value()})
	}

	if opts.IsRestricted.Has() {
		cond = cond.And(builder.Eq{"is_restricted": opts.IsRestricted.Value()})
	}

	if opts.IsProhibitLogin.Has() {
		cond = cond.And(builder.Eq{"prohibit_login": opts.IsProhibitLogin.Value()})
	}

	e := db.GetEngine(ctx)
	if !opts.IsTwoFactorEnabled.Has() {
		return e.Where(cond)
	}

	// Check if the user has two factor enabled, which is TOTP or Webauthn.
	if opts.IsTwoFactorEnabled.Value() {
		cond = cond.And(builder.Expr("two_factor.uid IS NOT NULL OR webauthn_credential.user_id IS NOT NULL"))
	} else {
		cond = cond.And(builder.Expr("two_factor.uid IS NULL AND webauthn_credential.user_id IS NULL"))
	}

	return e.Join("LEFT OUTER", "two_factor", "two_factor.uid = `user`.id").
		Join("LEFT OUTER", "webauthn_credential", "webauthn_credential.user_id = `user`.id").
		Where(cond)
}

// SearchUsers takes options i.e. keyword and part of user name to search,
// it returns results in given range and number of total results.
func SearchUsers(ctx context.Context, opts *SearchUserOptions) (users []*User, _ int64, _ error) {
	sessCount := opts.toSearchQueryBase(ctx)
	defer sessCount.Close()
	count, err := sessCount.Count(new(User))
	if err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	if len(opts.OrderBy) == 0 {
		opts.OrderBy = db.SearchOrderByAlphabetically
	}

	sessQuery := opts.toSearchQueryBase(ctx).OrderBy(opts.OrderBy.String())
	defer sessQuery.Close()
	if opts.PageSize > 0 {
		sessQuery = db.SetSessionPagination(sessQuery, opts)
	}

	// the sql may contain JOIN, so we must only select User related columns
	sessQuery = sessQuery.Select("`user`.*")
	users = make([]*User, 0, opts.PageSize)
	return users, count, sessQuery.Find(&users)
}

// BuildCanSeeUserCondition creates a condition which can be used to restrict results to users/orgs the actor can see
func BuildCanSeeUserCondition(actor *User) builder.Cond {
	if actor != nil {
		// If Admin - they see all users!
		if !actor.IsAdmin {
			// Users can see an organization they are a member of
			cond := builder.In("`user`.id", builder.Select("org_id").From("org_user").Where(builder.Eq{"uid": actor.ID}))
			if !actor.IsRestricted {
				// Not-Restricted users can see public and limited users/organizations
				cond = cond.Or(builder.In("`user`.visibility", structs.VisibleTypePublic, structs.VisibleTypeLimited))
			}
			// Don't forget about self
			return cond.Or(builder.Eq{"`user`.id": actor.ID})
		}

		return nil
	}

	// Force visibility for privacy
	// Not logged in - only public users
	return builder.In("`user`.visibility", structs.VisibleTypePublic)
}
