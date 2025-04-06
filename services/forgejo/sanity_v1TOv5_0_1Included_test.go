// SPDX-License-Identifier: MIT

package forgejo

import (
	"fmt"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/forgejo/semver"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/log"

	"github.com/stretchr/testify/require"
)

func TestForgejo_v1TOv5_0_1Included(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	logFatal = func(string, ...any) {}
	defer func() {
		logFatal = log.Fatal
	}()

	configWithSoragePath := `
[storage]
PATH = /something
`
	verifyForgejoV1TOv5_0_1Included(t, configWithSoragePath, "[storage].PATH is set")

	for _, c := range v1TOv5_0_1IncludedStorageSections {
		config := fmt.Sprintf("[%s]\n[%s]\n", c.section, c.storageSection)
		verifyForgejoV1TOv5_0_1Included(t, config, fmt.Sprintf("[%s] and [%s]", c.section, c.storageSection))
	}
}

func verifyForgejoV1TOv5_0_1Included(t *testing.T, config, message string) {
	ctx := db.DefaultContext
	e := db.GetEngine(ctx)

	for _, testCase := range []struct {
		name      string
		dbVersion int64
		semver    string
		config    string
	}{
		{
			name:      "5.0.0 with no " + message,
			dbVersion: ForgejoV5DatabaseVersion,
			semver:    "5.0.0+0-gitea-1.20.1",
			config:    "",
		},
		{
			name:      "5.0.1 with no " + message,
			dbVersion: ForgejoV5DatabaseVersion,
			semver:    "5.0.1+0-gitea-1.20.2",
			config:    "",
		},
		{
			name:      "5.0.2 with " + message,
			dbVersion: ForgejoV5DatabaseVersion,
			semver:    "5.0.2+0-gitea-1.20.3",
			config:    config,
		},
		{
			name:      "6.0.0 with " + message,
			dbVersion: ForgejoV6DatabaseVersion,
			semver:    "6.0.0+0-gitea-1.21.0",
			config:    config,
		},
	} {
		cfg := configFixture(t, testCase.config)
		semver.SetVersionString(ctx, testCase.semver)
		require.NoError(t, v1TOv5_0_1Included(e, testCase.dbVersion, cfg))
	}

	for _, testCase := range []struct {
		name      string
		dbVersion int64
		semver    string
		config    string
	}{
		{
			name:      "5.0.0 with  " + message,
			dbVersion: ForgejoV5DatabaseVersion,
			semver:    "5.0.0+0-gitea-1.20.1",
			config:    config,
		},
		{
			name:      "5.0.1 with " + message,
			dbVersion: ForgejoV5DatabaseVersion,
			semver:    "5.0.1+0-gitea-1.20.2",
			config:    config,
		},
		{
			//
			// When upgrading from
			//
			// Forgejo >= 5.0.1+0-gitea-1.20.2
			// Gitea > v1.21
			//
			// The version that the server was running prior to the upgrade
			// is not available.
			//
			name:      semver.DefaultVersionString + " with " + message,
			dbVersion: ForgejoV4DatabaseVersion,
			semver:    semver.DefaultVersionString,
			config:    config,
		},
	} {
		cfg := configFixture(t, testCase.config)
		semver.SetVersionString(ctx, testCase.semver)
		require.ErrorContains(t, v1TOv5_0_1Included(e, testCase.dbVersion, cfg), message)
	}
}
