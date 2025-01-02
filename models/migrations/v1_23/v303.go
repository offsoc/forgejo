// Copyright 2024 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package v1_23 //nolint

import (
	"code.gitea.io/gitea/models/migrations/base"

	"xorm.io/xorm"
)

func GiteaLastDrop(x *xorm.Engine) error {
	sess := x.NewSession()
	defer sess.Close()

	if err := base.DropTableColumns(sess, "badge", "slug"); err != nil {
		return err
	}
	if err := base.DropTableColumns(sess, "oauth2_application", "skip_secondary_authorization"); err != nil {
		return err
	}
	if err := base.DropTableColumns(sess, "repository", "default_wiki_branch"); err != nil {
		return err
	}
	// the migration v297.go that adds everyone_access_mode exists in Gitea >= v1.22 and the column must be dropped
	// but it does not exist in Forgejo and a failure to drop the column can be ignored
	base.DropTableColumns(sess, "repo_unit", "everyone_access_mode")
	if err := base.DropTableColumns(sess, "protected_branch", "can_force_push", "enable_force_push_allowlist", "force_push_allowlist_user_i_ds", "force_push_allowlist_team_i_ds", "force_push_allowlist_deploy_keys"); err != nil {
		return err
	}

	return sess.Commit()
}
