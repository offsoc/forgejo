// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"
	driver_options "code.gitea.io/gitea/services/f3/driver/options"

	_ "code.gitea.io/gitea/models"
	_ "code.gitea.io/gitea/models/actions"
	_ "code.gitea.io/gitea/models/activities"
	_ "code.gitea.io/gitea/models/perm/access"
	_ "code.gitea.io/gitea/services/f3/driver/tests"

	tests_f3 "code.forgejo.org/f3/gof3/v3/tree/tests/f3"
	"github.com/stretchr/testify/require"
)

func TestF3(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	tests_f3.ForgeCompliance(t, driver_options.Name)
}

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
