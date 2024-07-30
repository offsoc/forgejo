// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_discardStorage(t *testing.T) {
	tests := []DiscardStorage{
		UninitializedStorage,
		DiscardStorage("empty"),
	}
	for _, tt := range tests {
		t.Run(string(tt), func(t *testing.T) {
			{
				got, err := tt.Open("path")
				assert.Nil(t, got)
				require.Error(t, err, string(tt))
			}
			{
				got, err := tt.Save("path", bytes.NewReader([]byte{0}), 1)
				assert.Equal(t, int64(0), got)
				require.Error(t, err, string(tt))
			}
			{
				got, err := tt.Stat("path")
				assert.Nil(t, got)
				require.Error(t, err, string(tt))
			}
			{
				err := tt.Delete("path")
				require.Error(t, err, string(tt))
			}
			{
				got, err := tt.URL("path", "name")
				assert.Nil(t, got)
				require.Errorf(t, err, string(tt))
			}
			{
				err := tt.IterateObjects("", func(_ string, _ Object) error { return nil })
				require.Error(t, err, string(tt))
			}
		})
	}
}
