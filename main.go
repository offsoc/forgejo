// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"runtime"
	"strings"
	"time"

	"forgejo.org/cmd"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	// register supported doc types
	_ "forgejo.org/modules/markup/asciicast"
	_ "forgejo.org/modules/markup/console"
	_ "forgejo.org/modules/markup/csv"
	_ "forgejo.org/modules/markup/markdown"
	_ "forgejo.org/modules/markup/orgmode"

	"github.com/urfave/cli/v2"
)

// these flags will be set by the build flags
var (
	Version     = "development" // program version for this build
	Tags        = ""            // the Golang build tags
	MakeVersion = ""            // "make" program version if built with make

	ReleaseVersion = ""
)

var ForgejoVersion = "1.0.0"

func init() {
	setting.AppVer = Version
	setting.ForgejoVersion = ForgejoVersion
	setting.AppBuiltWith = formatBuiltWith()
	setting.AppStartTime = time.Now().UTC()
}

func forgejoEnv() {
	for _, k := range []string{"CUSTOM", "WORK_DIR"} {
		if v, ok := os.LookupEnv("FORGEJO_" + k); ok {
			os.Setenv("GITEA_"+k, v)
		}
	}
}

func main() {
	forgejoEnv()
	cli.OsExiter = func(code int) {
		log.GetManager().Close()
		os.Exit(code)
	}
	app := cmd.NewMainApp(Version, formatReleaseVersion()+formatBuiltWith())
	_ = cmd.RunMainApp(app, os.Args...) // all errors should have been handled by the RunMainApp
	log.GetManager().Close()
}

func formatReleaseVersion() string {
	if len(ReleaseVersion) > 0 {
		return " (release name " + ReleaseVersion + ")"
	}
	return ""
}

func formatBuiltWith() string {
	version := runtime.Version()
	if len(MakeVersion) > 0 {
		version = MakeVersion + ", " + runtime.Version()
	}
	if len(Tags) == 0 {
		return " built with " + version
	}

	return " built with " + version + " : " + strings.ReplaceAll(Tags, " ", ", ")
}
