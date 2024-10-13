// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"code.gitea.io/gitea/models/db"
)

// SetRepoActivityPubPrivPem function sets a repo's private and public keys
func (repo *Repository) SetRepoActivityPubPrivPem(ctx context.Context, pub, priv string) (err error) {
	_, err = db.GetEngine(ctx).ID(repo.ID).Update(&Repository{
		RepoActivityPubPrivPem: priv,
		RepoActivityPubPubPem:  pub,
	})

	return err
}

// GetRepoActivityPubPrivPem function returns a repo's private and public keys
func (repo *Repository) GetRepoActivityPubPrivPem(ctx context.Context) (string, string, error) {
	type Keys struct {
		RepoActivityPubPrivPem string
		RepoActivityPubPubPem  string
	}

	var keys Keys
	_, err := db.GetEngine(ctx).Table("repository").Where("`repository`.id = ?", repo.ID).Get(&keys)
	if err != nil {
		return "", "", err
	}

	return keys.RepoActivityPubPubPem, keys.RepoActivityPubPrivPem, err
}
