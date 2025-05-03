// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgejo.org/modules/structs"
)

// ActivityFeedsList
// swagger:response ActivityFeedsList
type swaggerActivityFeedsList struct {
	// in:body
	Body []api.Activity `json:"body"`
}
