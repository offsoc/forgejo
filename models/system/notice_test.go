// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package system_test

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/system"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotice_TrStr(t *testing.T) {
	notice := &system.Notice{
		Type:        system.NoticeRepository,
		Description: "test description",
	}
	assert.Equal(t, "admin.notices.type_1", notice.TrStr())
}

func TestCreateNotice(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	noticeBean := &system.Notice{
		Type:        system.NoticeRepository,
		Description: "test description",
	}
	unittest.AssertNotExistsBean(t, noticeBean)
	require.NoError(t, system.CreateNotice(db.DefaultContext, noticeBean.Type, noticeBean.Description))
	unittest.AssertExistsAndLoadBean(t, noticeBean)
}

func TestCreateRepositoryNotice(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	noticeBean := &system.Notice{
		Type:        system.NoticeRepository,
		Description: "test description",
	}
	unittest.AssertNotExistsBean(t, noticeBean)
	require.NoError(t, system.CreateRepositoryNotice(noticeBean.Description))
	unittest.AssertExistsAndLoadBean(t, noticeBean)
}

// TODO TestRemoveAllWithNotice

func TestCountNotices(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	assert.Equal(t, int64(3), system.CountNotices(db.DefaultContext))
}

func TestNotices(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	notices, err := system.Notices(db.DefaultContext, 1, 2)
	require.NoError(t, err)
	if assert.Len(t, notices, 2) {
		assert.Equal(t, int64(3), notices[0].ID)
		assert.Equal(t, int64(2), notices[1].ID)
	}

	notices, err = system.Notices(db.DefaultContext, 2, 2)
	require.NoError(t, err)
	if assert.Len(t, notices, 1) {
		assert.Equal(t, int64(1), notices[0].ID)
	}
}

func TestDeleteNotices(t *testing.T) {
	// delete a non-empty range
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 2})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 3})
	require.NoError(t, system.DeleteNotices(db.DefaultContext, 1, 2))
	unittest.AssertNotExistsBean(t, &system.Notice{ID: 1})
	unittest.AssertNotExistsBean(t, &system.Notice{ID: 2})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 3})
}

func TestDeleteNotices2(t *testing.T) {
	// delete an empty range
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 2})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 3})
	require.NoError(t, system.DeleteNotices(db.DefaultContext, 3, 2))
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 2})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 3})
}

func TestDeleteNoticesByIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 2})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 3})
	err := db.DeleteByIDs[system.Notice](db.DefaultContext, 1, 3)
	require.NoError(t, err)
	unittest.AssertNotExistsBean(t, &system.Notice{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &system.Notice{ID: 2})
	unittest.AssertNotExistsBean(t, &system.Notice{ID: 3})
}
