// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"path"
	"path/filepath"
)

var Gist = struct {
	Enabled  bool
	RootPath string
}{
	Enabled:  true,
	RootPath: "",
}

func loadGistFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("gist")
	Gist.Enabled = sec.Key("ENABLED").MustBool(true)
	Gist.RootPath = sec.Key("ROOT").MustString(path.Join(AppDataPath, "gists"))
	if !filepath.IsAbs(Gist.RootPath) {
		Gist.RootPath = filepath.Join(AppWorkPath, Gist.RootPath)
	} else {
		Gist.RootPath = filepath.Clean(Gist.RootPath)
	}
}
