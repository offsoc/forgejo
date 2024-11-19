// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package filebuffer

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBackedBuffer(t *testing.T) {
	cases := []struct {
		MaxMemorySize int
		Data          string
	}{
		{5, "test"},
		{5, "testtest"},
	}

	for _, c := range cases {
		buf, err := CreateFromReader(strings.NewReader(c.Data), c.MaxMemorySize)
		require.NoError(t, err)

		assert.EqualValues(t, len(c.Data), buf.Size())

		data, err := io.ReadAll(buf)
		require.NoError(t, err)
		assert.Equal(t, c.Data, string(data))

		require.NoError(t, buf.Close())
	}
}
