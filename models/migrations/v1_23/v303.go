// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package v1_23 //nolint

import (
	"forgejo.org/models/migrations/base"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func GiteaLastDrop(x *xorm.Engine) error {
	tables, err := x.DBMetas()
	if err != nil {
		return err
	}

	sess := x.NewSession()
	defer sess.Close()

	for _, drop := range []struct {
		table  string
		column string
	}{
		{"badge", "slug"},
		{"oauth2_application", "skip_secondary_authorization"},
		{"repository", "default_wiki_branch"},
		{"repo_unit", "everyone_access_mode"},
		{"protected_branch", "can_force_push"},
		{"protected_branch", "enable_force_push_allowlist"},
		{"protected_branch", "force_push_allowlist_user_i_ds"},
		{"protected_branch", "force_push_allowlist_team_i_ds"},
		{"protected_branch", "force_push_allowlist_deploy_keys"},
	} {
		var table *schemas.Table
		found := false

		for _, table = range tables {
			if table.Name == drop.table {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		if table.GetColumn(drop.column) == nil {
			continue
		}

		if err := base.DropTableColumns(sess, drop.table, drop.column); err != nil {
			return err
		}
	}

	return sess.Commit()
}
