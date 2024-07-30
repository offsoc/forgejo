// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixtureGeneration(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(ctx context.Context, gen func(ctx context.Context) (string, error), name string) {
		expected, err := gen(ctx)
		require.NoError(t, err)

		p := filepath.Join(unittest.FixturesDir(), name+".yml")
		bytes, err := os.ReadFile(p)
		require.NoError(t, err)

		data := string(util.NormalizeEOL(bytes))
		assert.EqualValues(t, expected, data, "Differences detected for %s", p)
	}

	test(db.DefaultContext, GetYamlFixturesAccess, "access")
}
