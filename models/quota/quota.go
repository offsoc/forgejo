// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/setting"
)

func init() {
	db.RegisterModel(new(Rule))
	db.RegisterModel(new(Group))
	db.RegisterModel(new(GroupRuleMapping))
	db.RegisterModel(new(GroupMapping))
}

func EvaluateForUser(ctx context.Context, userID int64, subject LimitSubject) (bool, error) {
	if !setting.Quota.Enabled {
		return true, nil
	}

	groups, err := GetGroupsForUser(ctx, userID)
	if err != nil {
		return false, err
	}

	used, err := GetUsedForUser(ctx, userID)
	if err != nil {
		return false, err
	}

	acceptable, _ := groups.Evaluate(*used, subject)
	return acceptable, nil
}
