// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"

	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"
)

// CmdDocs represents the available docs sub-command.
var CmdDocs = &cli.Command{
	Name:        "docs",
	Usage:       "Output CLI documentation",
	Description: "A command to output Forgejo's CLI documentation, optionally to a file.",
	Action:      runDocs,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "man",
			Usage: "Output man pages instead",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Path to output to instead of stdout (will overwrite if exists)",
		},
	},
}

func runDocs(ctx context.Context, c *cli.Command) error {
	docOutput, err := docs.ToMarkdown(c)
	if c.Bool("man") {
		docOutput, err = docs.ToMan(c)
	}
	if err != nil {
		return err
	}

	out := os.Stdout
	if c.String("output") != "" {
		fi, err := os.Create(c.String("output"))
		if err != nil {
			return err
		}
		defer fi.Close()
		out = fi
	}

	_, err = fmt.Fprintln(out, docOutput)
	return err
}
