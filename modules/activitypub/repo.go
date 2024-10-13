// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package activitypub

import (
	"context"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/util"
)

// GetRepoKeyPair function returns a repo's private and public keys
func GetRepoKeyPair(ctx context.Context, repo *repo_model.Repository) (string, string, error) {
	pub, priv, err := repo.GetRepoActivityPubPrivPem(ctx)
	if err != nil {
		return "", "", err
	}

	if pub == "" || priv == "" {
		priv, pub, err := util.GenerateKeyPair(rsaBits)
		if err != nil {
			return "", "", err
		}

		err = repo.SetRepoActivityPubPrivPem(ctx, pub, priv)

		return pub, priv, err
	}

	return pub, priv, err
}

// GetRepoPublicKey function returns a repo's public key
func GetRepoPublicKey(ctx context.Context, repo *repo_model.Repository) (pub string, err error) {
	pub, _, err = GetRepoKeyPair(ctx, repo)
	return pub, err
}

// GetRepoPrivateKey function returns a repo's private key
func GetRepoPrivateKey(ctx context.Context, repo *repo_model.Repository) (priv string, err error) {
	_, priv, err = GetRepoKeyPair(ctx, repo)
	return priv, err
}
