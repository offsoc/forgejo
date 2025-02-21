// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migrations

import (
	"net"
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/require"
)

func TestMigrateWhiteBlocklist(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
	nonAdminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})

	setting.Migrations.AllowedDomains = "github.com"
	setting.Migrations.AllowLocalNetworks = false
	require.NoError(t, Init())

	err := IsMigrateURLAllowed("https://gitlab.com/gitlab/gitlab.git", nonAdminUser)
	require.Error(t, err)

	err = IsMigrateURLAllowed("https://github.com/go-gitea/gitea.git", nonAdminUser)
	require.NoError(t, err)

	err = IsMigrateURLAllowed("https://gITHUb.com/go-gitea/gitea.git", nonAdminUser)
	require.NoError(t, err)

	setting.Migrations.AllowedDomains = ""
	setting.Migrations.BlockedDomains = "github.com"
	require.NoError(t, Init())

	err = IsMigrateURLAllowed("https://gitlab.com/gitlab/gitlab.git", nonAdminUser)
	require.NoError(t, err)

	err = IsMigrateURLAllowed("https://github.com/go-gitea/gitea.git", nonAdminUser)
	require.Error(t, err)

	err = IsMigrateURLAllowed("https://10.0.0.1/go-gitea/gitea.git", nonAdminUser)
	require.Error(t, err)

	setting.Migrations.AllowLocalNetworks = true
	require.NoError(t, Init())
	err = IsMigrateURLAllowed("https://10.0.0.1/go-gitea/gitea.git", nonAdminUser)
	require.NoError(t, err)

	old := setting.ImportLocalPaths
	setting.ImportLocalPaths = false

	err = IsMigrateURLAllowed("/home/foo/bar/goo", adminUser)
	require.Error(t, err)

	setting.ImportLocalPaths = true
	abs, err := filepath.Abs(".")
	require.NoError(t, err)

	err = IsMigrateURLAllowed(abs, adminUser)
	require.NoError(t, err)

	err = IsMigrateURLAllowed(abs, nonAdminUser)
	require.Error(t, err)

	nonAdminUser.AllowImportLocal = true
	err = IsMigrateURLAllowed(abs, nonAdminUser)
	require.NoError(t, err)

	setting.ImportLocalPaths = old
}

func TestAllowBlockList(t *testing.T) {
	init := func(allow, block string, local bool) {
		setting.Migrations.AllowedDomains = allow
		setting.Migrations.BlockedDomains = block
		setting.Migrations.AllowLocalNetworks = local
		require.NoError(t, Init())
	}

	// default, allow all external, block none, no local networks
	init("", "", false)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.Error(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// allow all including local networks (it could lead to SSRF in production)
	init("", "", true)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// allow wildcard, block some subdomains. if the domain name is allowed, then the local network check is skipped
	init("*.domain.com", "blocked.domain.com", false)
	require.NoError(t, checkByAllowBlockList("sub.domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.NoError(t, checkByAllowBlockList("sub.domain.com", []net.IP{net.ParseIP("127.0.0.1")}))
	require.Error(t, checkByAllowBlockList("blocked.domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.Error(t, checkByAllowBlockList("sub.other.com", []net.IP{net.ParseIP("1.2.3.4")}))

	// allow wildcard (it could lead to SSRF in production)
	init("*", "", false)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// local network can still be blocked
	init("*", "127.0.0.*", false)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.Error(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// reset
	init("", "", false)
}

func TestURLAllowedSSH(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	sshURL := "ssh://git@git.gay/gitgay/forgejo"

	t.Run("Migrate URL", func(t *testing.T) {
		require.Error(t, IsMigrateURLAllowed(sshURL, user))
	})

	t.Run("Pushmirror URL", func(t *testing.T) {
		require.NoError(t, IsPushMirrorURLAllowed(sshURL, user))
	})
}
