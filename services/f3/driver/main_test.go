// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"testing"

	"forgejo.org/models/unittest"
	driver_options "forgejo.org/services/f3/driver/options"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/perm/access"
	_ "forgejo.org/services/f3/driver/tests"

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
