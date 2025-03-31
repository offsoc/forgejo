// SPDX-License-Identifier: MIT

package semver

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoSemVerSetGet(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext

	newVersion, err := version.NewVersion("v1.2.3")
	require.NoError(t, err)
	require.NoError(t, SetVersionString(ctx, newVersion.String()))
	databaseVersion, err := GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, newVersion.String(), databaseVersion.String())
	assert.True(t, newVersion.Equal(databaseVersion))
}

func TestForgejoSemVerMissing(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := db.DefaultContext
	e := db.GetEngine(ctx)

	_, err := e.Exec("delete from forgejo_sem_ver")
	require.NoError(t, err)

	v, err := GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", v.String())

	_, err = e.Exec("drop table forgejo_sem_ver")
	require.NoError(t, err)

	v, err = GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", v.String())
}
