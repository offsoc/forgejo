// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package forgejo

import (
	"context"
	"errors"

	"forgejo.org/models"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"forgejo.org/services/f3/util"

	_ "forgejo.org/services/f3/driver" // register the driver

	f3_cmd "code.forgejo.org/f3/gof3/v3/cmd"
	f3_logger "code.forgejo.org/f3/gof3/v3/logger"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
	"github.com/urfave/cli/v3"
)

var CmdF3 = &cli.Command{
	Name:  "f3",
	Usage: "F3",
	Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
		return f3_logger.ContextSetLogger(ctx, util.NewF3Logger(nil, log.GetLogger(log.DEFAULT))), nil
	},
	Commands: []*cli.Command{
		SubcmdF3Mirror(),
	},
}

func SubcmdF3Mirror() *cli.Command {
	mirrorCmd := f3_cmd.CreateCmdMirror()
	mirrorCmd.Before = prepareWorkPathAndCustomConf
	f3Action := mirrorCmd.Action
	mirrorCmd.Action = func(ctx context.Context, c *cli.Command) error { return runMirror(ctx, c, f3Action) }
	return mirrorCmd
}

func runMirror(ctx context.Context, c *cli.Command, action cli.ActionFunc) error {
	setting.LoadF3Setting()
	if !setting.F3.Enabled {
		return errors.New("F3 is disabled, it is not ready to be used and is only present for development purposes")
	}

	var cancel context.CancelFunc
	if !ContextGetNoInit(ctx) {
		ctx, cancel = installSignals(ctx)
		defer cancel()

		if err := initDB(ctx); err != nil {
			return err
		}

		if err := storage.Init(); err != nil {
			return err
		}

		if err := git.InitSimple(ctx); err != nil {
			return err
		}
		if err := models.Init(ctx); err != nil {
			return err
		}
	}

	err := action(ctx, c)
	if panicError, ok := err.(f3_util.PanicError); ok {
		log.Debug("F3 Stack trace\n%s", panicError.Stack())
	}
	return err
}
