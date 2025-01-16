// Copyright 2024 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package v1_23 //nolint

import (
	"fmt"

	"code.gitea.io/gitea/models/migrations/base"

	"xorm.io/xorm"
)

func GiteaLastDrop(x *xorm.Engine) error {
	sess := x.NewSession()
	defer sess.Close()

	for _, drop := range []struct {
		table string
		field string
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
		if _, err := sess.Exec(fmt.Sprintf("SELECT `%s` FROM `%s` WHERE 0 = 1", drop.field, drop.table)); err != nil {
			continue
		}
		if err := base.DropTableColumns(sess, drop.table, drop.field); err != nil {
			return err
		}
	}

	return sess.Commit()
}
