// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"forgejo.org/modules/base"
	"forgejo.org/routers/web/shared"
	"forgejo.org/services/context"
)

const (
	tplSettingsStorageOverview base.TplName = "org/settings/storage_overview"
)

// StorageOverview render a size overview of the organization, as well as relevant
// quota limits of the instance.
func StorageOverview(ctx *context.Context) {
	shared.StorageOverview(ctx, ctx.Org.Organization.ID, tplSettingsStorageOverview)
}
