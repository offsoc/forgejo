// SPDX-License-Identifier: MIT

package semver

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"

	_ "code.gitea.io/gitea/models"
	_ "code.gitea.io/gitea/models/actions"
	_ "code.gitea.io/gitea/models/activities"
	_ "code.gitea.io/gitea/models/forgefed"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
