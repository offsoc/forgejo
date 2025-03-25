// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"net/http"

	driver_options "forgejo.org/services/f3/driver/options"

	"code.forgejo.org/f3/gof3/v3/options"
)

func newOptions() options.Interface {
	o := &driver_options.Options{}
	o.SetName(driver_options.Name)
	o.SetNewMigrationHTTPClient(func() *http.Client { return &http.Client{} })
	return o
}
