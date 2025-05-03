// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgejo.org/modules/structs"
)

// CronList
// swagger:response CronList
type swaggerResponseCronList struct {
	// in:body
	Body []api.Cron `json:"body"`
}
