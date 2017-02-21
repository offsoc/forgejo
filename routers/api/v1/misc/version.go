// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package misc

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/sdk/gitea"
)

// ServerVersion shows the version of the Gitea server
func Version(ctx *context.APIContext) {
	ctx.JSON(200, &gitea.ServerVersion{Version: setting.AppVer})
}
