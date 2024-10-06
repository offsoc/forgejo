// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package options

import (
	"net/http"

	"code.forgejo.org/f3/gof3/v3/options"
	"code.forgejo.org/f3/gof3/v3/options/cli"
	"code.forgejo.org/f3/gof3/v3/options/logger"
)

type NewMigrationHTTPClientFun func() *http.Client

type Options struct {
	options.Options
	logger.OptionsLogger
	cli.OptionsCLI

	NewMigrationHTTPClient NewMigrationHTTPClientFun
}

func (o *Options) GetNewMigrationHTTPClient() NewMigrationHTTPClientFun {
	return o.NewMigrationHTTPClient
}

func (o *Options) SetNewMigrationHTTPClient(fun NewMigrationHTTPClientFun) {
	o.NewMigrationHTTPClient = fun
}
