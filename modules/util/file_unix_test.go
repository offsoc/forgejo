// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyUmask(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test-filemode-")
	require.NoError(t, err)

	err = os.Chmod(f.Name(), 0o777)
	require.NoError(t, err)
	st, err := os.Stat(f.Name())
	require.NoError(t, err)
	assert.EqualValues(t, 0o777, st.Mode().Perm()&0o777)

	oldDefaultUmask := defaultUmask
	defaultUmask = 0o037
	defer func() {
		defaultUmask = oldDefaultUmask
	}()
	err = ApplyUmask(f.Name(), os.ModePerm)
	require.NoError(t, err)
	st, err = os.Stat(f.Name())
	require.NoError(t, err)
	assert.EqualValues(t, 0o740, st.Mode().Perm()&0o777)
}
