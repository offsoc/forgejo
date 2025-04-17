// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"fmt"
	"net/http"

	"forgejo.org/modules/setting"
	asymkey_service "forgejo.org/services/asymkey"
	"forgejo.org/services/context"

	"golang.org/x/crypto/ssh"
)

// SigningKey returns the public key of the default signing key if it exists
func SigningKey(ctx *context.APIContext) {
	// swagger:operation GET /signing-key.gpg miscellaneous getSigningKey
	// ---
	// summary: Get default signing-key.gpg
	// produces:
	//     - text/plain
	// responses:
	//   "200":
	//     description: "GPG armored public key"
	//     schema:
	//       type: string

	// swagger:operation GET /repos/{owner}/{repo}/signing-key.gpg repository repoSigningKey
	// ---
	// summary: Get signing-key.gpg for given repository
	// produces:
	//     - text/plain
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     description: "GPG armored public key"
	//     schema:
	//       type: string

	path := ""
	if ctx.Repo != nil && ctx.Repo.Repository != nil {
		path = ctx.Repo.Repository.RepoPath()
	}

	content, err := asymkey_service.PublicSigningKey(ctx, path)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "gpg export", err)
		return
	}
	_, err = ctx.Write([]byte(content))
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "gpg export", fmt.Errorf("Error writing key content %w", err))
	}
}

// SSHSigningKey returns the public SSH key of the default signing key if it exists
func SSHSigningKey(ctx *context.APIContext) {
	// swagger:operation GET /signing-key.ssh miscellaneous getSSHSigningKey
	// ---
	// summary: Get default signing-key.ssh
	// produces:
	//     - text/plain
	// responses:
	//   "200":
	//     description: "SSH public key in OpenSSH authorized key format"
	//     schema:
	//       type: string
	//   "404":
	//     "$ref": "#/responses/notFound"

	if setting.SSHInstanceKey == nil {
		ctx.NotFound()
		return
	}

	_, err := ctx.Write(ssh.MarshalAuthorizedKey(setting.SSHInstanceKey))
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ssh export", err)
	}
}
