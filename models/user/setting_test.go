// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettings(t *testing.T) {
	keyName := "test_user_setting"
	require.NoError(t, unittest.PrepareTestDatabase())

	newSetting := &user_model.Setting{UserID: 99, SettingKey: keyName, SettingValue: "Gitea User Setting Test"}

	// create setting
	err := user_model.SetUserSetting(db.DefaultContext, newSetting.UserID, newSetting.SettingKey, newSetting.SettingValue)
	require.NoError(t, err)
	// test about saving unchanged values
	err = user_model.SetUserSetting(db.DefaultContext, newSetting.UserID, newSetting.SettingKey, newSetting.SettingValue)
	require.NoError(t, err)

	// get specific setting
	settings, err := user_model.GetSettings(db.DefaultContext, 99, []string{keyName})
	require.NoError(t, err)
	assert.Len(t, settings, 1)
	assert.Equal(t, newSetting.SettingValue, settings[keyName].SettingValue)

	settingValue, err := user_model.GetUserSetting(db.DefaultContext, 99, keyName)
	require.NoError(t, err)
	assert.Equal(t, newSetting.SettingValue, settingValue)

	settingValue, err = user_model.GetUserSetting(db.DefaultContext, 99, "no_such")
	require.NoError(t, err)
	assert.Empty(t, settingValue)

	// updated setting
	updatedSetting := &user_model.Setting{UserID: 99, SettingKey: keyName, SettingValue: "Updated"}
	err = user_model.SetUserSetting(db.DefaultContext, updatedSetting.UserID, updatedSetting.SettingKey, updatedSetting.SettingValue)
	require.NoError(t, err)

	// get all settings
	settings, err = user_model.GetUserAllSettings(db.DefaultContext, 99)
	require.NoError(t, err)
	assert.Len(t, settings, 1)
	assert.Equal(t, updatedSetting.SettingValue, settings[updatedSetting.SettingKey].SettingValue)

	// delete setting
	err = user_model.DeleteUserSetting(db.DefaultContext, 99, keyName)
	require.NoError(t, err)
	settings, err = user_model.GetUserAllSettings(db.DefaultContext, 99)
	require.NoError(t, err)
	assert.Empty(t, settings)
}
