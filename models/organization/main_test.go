// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package organization_test

import (
	"path/filepath"
	"testing"

	_ "code.gitea.io/gitea/models/organization"
	_ "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	_ "code.gitea.io/gitea/models/user"
	_ "code.gitea.io/gitea/models"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}
