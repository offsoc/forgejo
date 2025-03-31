// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_20 //nolint

import (
	"sort"
	"strings"
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	Old OldAccessTokenScope
	New AccessTokenScope
}

func createOldTokenScope(scopes ...OldAccessTokenScope) OldAccessTokenScope {
	s := make([]string, 0, len(scopes))
	for _, os := range scopes {
		s = append(s, string(os))
	}
	return OldAccessTokenScope(strings.Join(s, ","))
}

func createNewTokenScope(scopes ...AccessTokenScope) AccessTokenScope {
	s := make([]string, 0, len(scopes))
	for _, os := range scopes {
		s = append(s, string(os))
	}
	return AccessTokenScope(strings.Join(s, ","))
}

func Test_ConvertScopedAccessTokens(t *testing.T) {
	tests := []testCase{
		{
			createOldTokenScope(OldAccessTokenScopeRepo, OldAccessTokenScopeUserFollow),
			createNewTokenScope(AccessTokenScopeWriteRepository, AccessTokenScopeWriteUser),
		},
		{
			createOldTokenScope(OldAccessTokenScopeUser, OldAccessTokenScopeWritePackage, OldAccessTokenScopeSudo),
			createNewTokenScope(AccessTokenScopeWriteAdmin, AccessTokenScopeWritePackage, AccessTokenScopeWriteUser),
		},
		{
			createOldTokenScope(),
			createNewTokenScope(),
		},
		{
			createOldTokenScope(OldAccessTokenScopeReadGPGKey, OldAccessTokenScopeReadOrg, OldAccessTokenScopeAll),
			createNewTokenScope(AccessTokenScopeAll),
		},
		{
			createOldTokenScope(OldAccessTokenScopeReadGPGKey, "invalid"),
			createNewTokenScope("invalid", AccessTokenScopeReadUser),
		},
	}

	// add a test for each individual mapping
	for oldScope, newScope := range accessTokenScopeMap {
		tests = append(tests, testCase{
			oldScope,
			createNewTokenScope(newScope...),
		})
	}

	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(AccessToken))
	defer deferable()
	if x == nil || t.Failed() {
		t.Skip()
		return
	}

	// verify that no fixtures were loaded
	count, err := x.Count(&AccessToken{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	for _, tc := range tests {
		_, err = x.Insert(&AccessToken{
			Scope: string(tc.Old),
		})
		require.NoError(t, err)
	}

	// migrate the scopes
	err = ConvertScopedAccessTokens(x)
	require.NoError(t, err)

	// migrate the scopes again (migration should be idempotent)
	err = ConvertScopedAccessTokens(x)
	require.NoError(t, err)

	tokens := make([]AccessToken, 0)
	err = x.Find(&tokens)
	require.NoError(t, err)
	assert.Len(t, tokens, len(tests))

	// sort the tokens (insertion order by auto-incrementing primary key)
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].ID < tokens[j].ID
	})

	// verify that the converted scopes are equal to the expected test result
	for idx, newToken := range tokens {
		assert.Equal(t, string(tests[idx].New), newToken.Scope)
	}
}
