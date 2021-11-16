// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"os"
	"testing"

	"code.gitea.io/gitea/models"
<<<<<<< HEAD
	"code.gitea.io/gitea/modules/migrations"
=======
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
>>>>>>> 376f01539 (Fix imports)
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/migrations"

	"github.com/stretchr/testify/assert"
)

func TestMigrateLocalPath(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	adminUser := unittest.AssertExistsAndLoadBean(t, &models.User{Name: "user1"}).(*models.User)

	old := setting.ImportLocalPaths
	setting.ImportLocalPaths = true

	lowercasePath, err := os.MkdirTemp("", "lowercase") // may not be lowercase because MkdirTemp creates a random directory name which may be mixedcase
	assert.NoError(t, err)
	defer os.RemoveAll(lowercasePath)

	err = migrations.IsMigrateURLAllowed(lowercasePath, adminUser)
	assert.NoError(t, err, "case lowercase path")

	mixedcasePath, err := os.MkdirTemp("", "mIxeDCaSe")
	assert.NoError(t, err)
	defer os.RemoveAll(mixedcasePath)

	err = migrations.IsMigrateURLAllowed(mixedcasePath, adminUser)
	assert.NoError(t, err, "case mixedcase path")

	setting.ImportLocalPaths = old
}
