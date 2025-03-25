// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// ErrUserRedirectNotExist represents a "UserRedirectNotExist" kind of error.
type ErrUserRedirectNotExist struct {
	Name string
}

// IsErrUserRedirectNotExist check if an error is an ErrUserRedirectNotExist.
func IsErrUserRedirectNotExist(err error) bool {
	_, ok := err.(ErrUserRedirectNotExist)
	return ok
}

func (err ErrUserRedirectNotExist) Error() string {
	return fmt.Sprintf("user redirect does not exist [name: %s]", err.Name)
}

func (err ErrUserRedirectNotExist) Unwrap() error {
	return util.ErrNotExist
}

type ErrCooldownPeriod struct {
	ExpireTime time.Time
}

func IsErrCooldownPeriod(err error) bool {
	_, ok := err.(ErrCooldownPeriod)
	return ok
}

func (err ErrCooldownPeriod) Error() string {
	return fmt.Sprintf("cooldown period for claiming this username has not yet expired: the cooldown period ends at %s", err.ExpireTime)
}

// Redirect represents that a user name should be redirected to another
type Redirect struct {
	ID             int64              `xorm:"pk autoincr"`
	LowerName      string             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	RedirectUserID int64              // userID to redirect to
	CreatedUnix    timeutil.TimeStamp `xorm:"created NOT NULL DEFAULT 0"`
}

// TableName provides the real table name
func (Redirect) TableName() string {
	return "user_redirect"
}

func init() {
	db.RegisterModel(new(Redirect))
}

// GetUserRedirect returns the redirect for a given username, this is a
// case-insensitive operation.
func GetUserRedirect(ctx context.Context, userName string) (*Redirect, error) {
	userName = strings.ToLower(userName)
	redirect := &Redirect{LowerName: userName}
	if has, err := db.GetEngine(ctx).Get(redirect); err != nil {
		return nil, err
	} else if !has {
		return nil, ErrUserRedirectNotExist{Name: userName}
	}
	return redirect, nil
}

// LookupUserRedirect look up userID if a user has a redirect name
func LookupUserRedirect(ctx context.Context, userName string) (int64, error) {
	redirect, err := GetUserRedirect(ctx, userName)
	if err != nil {
		return 0, err
	}
	return redirect.RedirectUserID, nil
}

// NewUserRedirect create a new user redirect
func NewUserRedirect(ctx context.Context, ID int64, oldUserName, newUserName string) error {
	oldUserName = strings.ToLower(oldUserName)
	newUserName = strings.ToLower(newUserName)

	if err := DeleteUserRedirect(ctx, oldUserName); err != nil {
		return err
	}

	if err := DeleteUserRedirect(ctx, newUserName); err != nil {
		return err
	}

	return db.Insert(ctx, &Redirect{
		LowerName:      oldUserName,
		RedirectUserID: ID,
	})
}

// LimitUserRedirects deletes the oldest entries in user_redirect of the user,
// such that the amount of user_redirects is at most `n` amount of entries.
func LimitUserRedirects(ctx context.Context, userID, n int64) error {
	// NOTE: It's not possible to combine these two queries into one due to a limitation of MySQL.
	keepIDs := make([]int64, n)
	if err := db.GetEngine(ctx).SQL("SELECT id FROM user_redirect WHERE redirect_user_id = ? ORDER BY created_unix DESC LIMIT "+strconv.FormatInt(n, 10), userID).Find(&keepIDs); err != nil {
		return err
	}

	_, err := db.GetEngine(ctx).Exec(builder.Delete(builder.And(builder.Eq{"redirect_user_id": userID}, builder.NotIn("id", keepIDs))).From("user_redirect"))
	return err
}

// DeleteUserRedirect delete any redirect from the specified user name to
// anything else
func DeleteUserRedirect(ctx context.Context, userName string) error {
	userName = strings.ToLower(userName)
	_, err := db.GetEngine(ctx).Delete(&Redirect{LowerName: userName})
	return err
}

// CanClaimUsername returns if its possible to claim the given username,
// it checks if the cooldown period for claiming an existing username is over.
// If there's a cooldown period, the second argument returns the time when
// that cooldown period is over.
// In the scenario of renaming, the doerID can be specified to allow the original
// user of the username to reclaim it within the cooldown period.
func CanClaimUsername(ctx context.Context, username string, doerID int64) (bool, time.Time, error) {
	// Only check for a cooldown period if UsernameCooldownPeriod is a positive number.
	if setting.Service.UsernameCooldownPeriod <= 0 {
		return true, time.Time{}, nil
	}

	userRedirect, err := GetUserRedirect(ctx, username)
	if err != nil {
		if IsErrUserRedirectNotExist(err) {
			return true, time.Time{}, nil
		}
		return false, time.Time{}, err
	}

	// Allow reclaiming of user's own username.
	if userRedirect.RedirectUserID == doerID {
		return true, time.Time{}, nil
	}

	// We do not know if the redirect user id was for an organization, so
	// unconditionally execute the following query to retrieve all users that
	// are part of the "Owner" team. If the redirect user ID is not an organization
	// the returned list would be empty.
	ownerTeamUIDs := []int64{}
	if err := db.GetEngine(ctx).SQL("SELECT uid FROM team_user INNER JOIN team ON team_user.`team_id` = team.`id` WHERE team.`org_id` = ? AND team.`name` = 'Owners'", userRedirect.RedirectUserID).Find(&ownerTeamUIDs); err != nil {
		return false, time.Time{}, err
	}

	if slices.Contains(ownerTeamUIDs, doerID) {
		return true, time.Time{}, nil
	}

	// Multiply the value of UsernameCooldownPeriod by the amount of seconds in a day.
	expireTime := userRedirect.CreatedUnix.Add(86400 * setting.Service.UsernameCooldownPeriod).AsLocalTime()
	return time.Until(expireTime) <= 0, expireTime, nil
}
