// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package avatars_test

import (
	"testing"

	avatars_model "forgejo.org/models/avatars"
	"forgejo.org/models/db"
	system_model "forgejo.org/models/system"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/setting/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gravatarSource = "https://secure.gravatar.com/avatar/"

func disableGravatar(t *testing.T) {
	err := system_model.SetSettings(db.DefaultContext, map[string]string{setting.Config().Picture.EnableFederatedAvatar.DynKey(): "false"})
	require.NoError(t, err)
	err = system_model.SetSettings(db.DefaultContext, map[string]string{setting.Config().Picture.DisableGravatar.DynKey(): "true"})
	require.NoError(t, err)
}

func enableGravatar(t *testing.T) {
	err := system_model.SetSettings(db.DefaultContext, map[string]string{setting.Config().Picture.DisableGravatar.DynKey(): "false"})
	require.NoError(t, err)
	setting.GravatarSource = gravatarSource
}

func TestHashEmail(t *testing.T) {
	assert.Equal(t,
		"d41d8cd98f00b204e9800998ecf8427e",
		avatars_model.HashEmail(""),
	)
	assert.Equal(t,
		"353cbad9b58e69c96154ad99f92bedc7",
		avatars_model.HashEmail("gitea@example.com"),
	)
}

func TestSizedAvatarLink(t *testing.T) {
	setting.AppSubURL = "/testsuburl"

	disableGravatar(t)
	config.GetDynGetter().InvalidateCache()
	assert.Equal(t, "/testsuburl/assets/img/avatar_default.png",
		avatars_model.GenerateEmailAvatarFastLink(db.DefaultContext, "gitea@example.com", 100))

	enableGravatar(t)
	config.GetDynGetter().InvalidateCache()
	assert.Equal(t,
		"https://secure.gravatar.com/avatar/353cbad9b58e69c96154ad99f92bedc7?d=identicon&s=100",
		avatars_model.GenerateEmailAvatarFastLink(db.DefaultContext, "gitea@example.com", 100),
	)
}
