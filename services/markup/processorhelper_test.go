// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markup

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	gitea_context "forgejo.org/services/context"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessorHelper(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	userPublic := "user1"
	userPrivate := "user31"
	userLimited := "user33"
	userNoSuch := "no-such-user"

	unittest.AssertCount(t, &user.User{Name: userPublic}, 1)
	unittest.AssertCount(t, &user.User{Name: userPrivate}, 1)
	unittest.AssertCount(t, &user.User{Name: userLimited}, 1)
	unittest.AssertCount(t, &user.User{Name: userNoSuch}, 0)

	// when using general context, use user's visibility to check
	assert.True(t, ProcessorHelper().IsUsernameMentionable(t.Context(), userPublic))
	assert.False(t, ProcessorHelper().IsUsernameMentionable(t.Context(), userLimited))
	assert.False(t, ProcessorHelper().IsUsernameMentionable(t.Context(), userPrivate))
	assert.False(t, ProcessorHelper().IsUsernameMentionable(t.Context(), userNoSuch))

	// when using web context, use user.IsUserVisibleToViewer to check
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	base, baseCleanUp := gitea_context.NewBaseContext(httptest.NewRecorder(), req)
	defer baseCleanUp()
	giteaCtx := gitea_context.NewWebContext(base, &contexttest.MockRender{}, nil)

	assert.True(t, ProcessorHelper().IsUsernameMentionable(giteaCtx, userPublic))
	assert.False(t, ProcessorHelper().IsUsernameMentionable(giteaCtx, userPrivate))

	giteaCtx.Doer, err = user.GetUserByName(db.DefaultContext, userPrivate)
	require.NoError(t, err)
	assert.True(t, ProcessorHelper().IsUsernameMentionable(giteaCtx, userPublic))
	assert.True(t, ProcessorHelper().IsUsernameMentionable(giteaCtx, userPrivate))
}
