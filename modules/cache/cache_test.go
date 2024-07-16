// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cache

import (
	"fmt"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCache() {
	conn, _ = newCache(setting.Cache{
		Adapter: "memory",
		TTL:     time.Minute,
	})
	setting.CacheService.TTL = 24 * time.Hour
}

func TestNewContext(t *testing.T) {
	require.NoError(t, Init())

	setting.CacheService.Cache = setting.Cache{Adapter: "redis", Conn: "some random string"}
	con, err := newCache(setting.Cache{
		Adapter:  "rand",
		Conn:     "false conf",
		Interval: 100,
	})
	require.Error(t, err)
	assert.Nil(t, con)
}

func TestTest(t *testing.T) {
	defer test.MockVariableValue(&conn, nil)()
	_, err := Test()
	require.Error(t, err)

	createTestCache()
	elapsed, err := Test()
	require.NoError(t, err)
	// mem cache should take from 300ns up to 1ms on modern hardware ...
	assert.Less(t, elapsed, SlowCacheThreshold)
}

func TestGetCache(t *testing.T) {
	createTestCache()

	assert.NotNil(t, GetCache())
}

func TestGetString(t *testing.T) {
	createTestCache()

	data, err := GetString("key", func() (string, error) {
		return "", fmt.Errorf("some error")
	})
	require.Error(t, err)
	assert.Equal(t, "", data)

	data, err = GetString("key", func() (string, error) {
		return "", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "", data)

	data, err = GetString("key", func() (string, error) {
		return "some data", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "", data)
	Remove("key")

	data, err = GetString("key", func() (string, error) {
		return "some data", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "some data", data)

	data, err = GetString("key", func() (string, error) {
		return "", fmt.Errorf("some error")
	})
	require.NoError(t, err)
	assert.Equal(t, "some data", data)
	Remove("key")
}

func TestGetInt(t *testing.T) {
	createTestCache()

	data, err := GetInt("key", func() (int, error) {
		return 0, fmt.Errorf("some error")
	})
	require.Error(t, err)
	assert.Equal(t, 0, data)

	data, err = GetInt("key", func() (int, error) {
		return 0, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 0, data)

	data, err = GetInt("key", func() (int, error) {
		return 100, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 0, data)
	Remove("key")

	data, err = GetInt("key", func() (int, error) {
		return 100, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 100, data)

	data, err = GetInt("key", func() (int, error) {
		return 0, fmt.Errorf("some error")
	})
	require.NoError(t, err)
	assert.Equal(t, 100, data)
	Remove("key")
}

func TestGetInt64(t *testing.T) {
	createTestCache()

	data, err := GetInt64("key", func() (int64, error) {
		return 0, fmt.Errorf("some error")
	})
	require.Error(t, err)
	assert.EqualValues(t, 0, data)

	data, err = GetInt64("key", func() (int64, error) {
		return 0, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, data)

	data, err = GetInt64("key", func() (int64, error) {
		return 100, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, data)
	Remove("key")

	data, err = GetInt64("key", func() (int64, error) {
		return 100, nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, 100, data)

	data, err = GetInt64("key", func() (int64, error) {
		return 0, fmt.Errorf("some error")
	})
	require.NoError(t, err)
	assert.EqualValues(t, 100, data)
	Remove("key")
}
