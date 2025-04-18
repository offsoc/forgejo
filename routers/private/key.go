// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"net/http"

	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/modules/private"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/context"
)

// UpdatePublicKeyInRepo update public key and deploy key updates
func UpdatePublicKeyInRepo(ctx *context.PrivateContext) {
	keyID := ctx.ParamsInt64(":id")
	repoID := ctx.ParamsInt64(":repoid")
	if err := asymkey_model.UpdatePublicKeyUpdated(ctx, keyID); err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: err.Error(),
		})
		return
	}

	deployKey, err := asymkey_model.GetDeployKeyByRepo(ctx, keyID, repoID)
	if err != nil {
		if asymkey_model.IsErrDeployKeyNotExist(err) {
			ctx.PlainText(http.StatusOK, "success")
			return
		}
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: err.Error(),
		})
		return
	}
	deployKey.UpdatedUnix = timeutil.TimeStampNow()
	if err = asymkey_model.UpdateDeployKeyCols(ctx, deployKey, "updated_unix"); err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: err.Error(),
		})
		return
	}

	ctx.PlainText(http.StatusOK, "success")
}

// AuthorizedPublicKeyByContent searches content as prefix (leak e-mail part)
// and returns public key found.
func AuthorizedPublicKeyByContent(ctx *context.PrivateContext) {
	content := ctx.FormString("content")

	publicKey, err := asymkey_model.SearchPublicKeyByContent(ctx, content)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: err.Error(),
		})
		return
	}
	ctx.PlainText(http.StatusOK, publicKey.AuthorizedString())
}
