// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackAndUnpackData(t *testing.T) {
	s := "string"
	i := int64(4)
	f := float32(4.1)

	var s2 string
	var i2 int64
	var f2 float32

	data, err := PackData(s, i, f)
	require.NoError(t, err)

	require.NoError(t, UnpackData(data, &s2, &i2, &f2))
	require.NoError(t, UnpackData(data, &s2))
	require.Error(t, UnpackData(data, &i2))
	require.Error(t, UnpackData(data, &s2, &f2))
}
