// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"path/filepath"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
)

func AddFixtures(dirs ...string) func() {
	return unittest.OverrideFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: dirs,
		},
	)
}

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
