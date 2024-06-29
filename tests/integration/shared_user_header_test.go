// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/optional"
	shared_user "code.gitea.io/gitea/routers/web/shared/user"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/contexttest"
	user_service "code.gitea.io/gitea/services/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func setupUser(t *testing.T, pronouns string, privatePronouns, signedIn bool) *context.Context {
	t.Helper()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	assert.NoError(t, user_service.UpdateUser(db.DefaultContext, user, &user_service.UpdateOptions{
		Pronouns:            optional.Some(pronouns),
		KeepPronounsPrivate: optional.Some(privatePronouns),
	}))

	ctx, _ := contexttest.MockContext(t, "/user1")
	contexttest.LoadUser(t, ctx, 1)
	ctx.ContextUser = ctx.Doer
	ctx.IsSigned = signedIn
	shared_user.PrepareContextForProfileBigAvatar(ctx)

	return ctx
}

func TestSharedUserHeader_PronounsPrivacy(t *testing.T) {
	t.Run("HidePronounsIfNoneSet", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()
		ctx := setupUser(t, "", false, false)

		assert.Equal(t, false, ctx.Data["ShowPronouns"])

		ctx = setupUser(t, "", false, true)

		assert.Equal(t, false, ctx.Data["ShowPronouns"])
	})
	t.Run("HidePronounsIfSetButPrivateAndNotLoggedIn", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()
		ctx := setupUser(t, "any", true, false)

		assert.Equal(t, false, ctx.Data["ShowPronouns"])
	})
	t.Run("ShowPronounsIfSetAndNotPrivateAndNotLoggedIn", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()
		ctx := setupUser(t, "any", false, false)

		assert.Equal(t, true, ctx.Data["ShowPronouns"])
	})
	t.Run("ShowPronounsIfSetAndPrivateAndLoggedIn", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()
		ctx := setupUser(t, "any", true, true)

		assert.Equal(t, true, ctx.Data["ShowPronouns"])
	})
	t.Run("ShowPronounsIfSetAndNotPrivateAndLoggedIn", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()
		ctx := setupUser(t, "any", false, true)

		assert.Equal(t, true, ctx.Data["ShowPronouns"])
	})
}
