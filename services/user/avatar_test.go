// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package user

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alreadyDeletedStorage struct {
	storage.DiscardStorage
}

func (s alreadyDeletedStorage) Delete(_ string) error {
	return os.ErrNotExist
}

func TestUserDeleteAvatar(t *testing.T) {
	myImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)

	t.Run("AtomicStorageFailure", func(t *testing.T) {
		defer test.MockProtect[storage.ObjectStorage](&storage.Avatars)()

		require.NoError(t, unittest.PrepareTestDatabase())
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		err := UploadAvatar(db.DefaultContext, user, buff.Bytes())
		require.NoError(t, err)
		verification := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		assert.NotEqual(t, "", verification.Avatar)

		// fail to delete ...
		storage.Avatars = storage.UninitializedStorage
		err = DeleteAvatar(db.DefaultContext, user)
		require.Error(t, err)

		// ... the avatar is not removed from the database
		verification = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		assert.True(t, verification.UseCustomAvatar)

		// already deleted ...
		storage.Avatars = alreadyDeletedStorage{}
		err = DeleteAvatar(db.DefaultContext, user)
		require.NoError(t, err)

		// ... the avatar is removed from the database
		verification = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		assert.Equal(t, "", verification.Avatar)
	})

	t.Run("Success", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		err := UploadAvatar(db.DefaultContext, user, buff.Bytes())
		require.NoError(t, err)
		verification := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		assert.NotEqual(t, "", verification.Avatar)

		err = DeleteAvatar(db.DefaultContext, user)
		require.NoError(t, err)

		verification = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		assert.Equal(t, "", verification.Avatar)
	})
}
