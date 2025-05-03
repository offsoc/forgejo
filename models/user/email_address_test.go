// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"fmt"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEmailAddresses(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	emails, _ := user_model.GetEmailAddresses(db.DefaultContext, int64(1))
	if assert.Len(t, emails, 3) {
		assert.True(t, emails[0].IsPrimary)
		assert.True(t, emails[2].IsActivated)
		assert.False(t, emails[2].IsPrimary)
	}

	emails, _ = user_model.GetEmailAddresses(db.DefaultContext, int64(2))
	if assert.Len(t, emails, 2) {
		assert.True(t, emails[0].IsPrimary)
		assert.True(t, emails[0].IsActivated)
	}
}

func TestIsEmailUsed(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	isExist, _ := user_model.IsEmailUsed(db.DefaultContext, "")
	assert.True(t, isExist)
	isExist, _ = user_model.IsEmailUsed(db.DefaultContext, "user11@example.com")
	assert.True(t, isExist)
	isExist, _ = user_model.IsEmailUsed(db.DefaultContext, "user1234567890@example.com")
	assert.False(t, isExist)
}

func TestActivate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	email := &user_model.EmailAddress{
		ID:    int64(1),
		UID:   int64(1),
		Email: "user11@example.com",
	}
	require.NoError(t, user_model.ActivateEmail(db.DefaultContext, email))

	emails, _ := user_model.GetEmailAddresses(db.DefaultContext, int64(1))
	assert.Len(t, emails, 3)
	assert.True(t, emails[0].IsActivated)
	assert.True(t, emails[0].IsPrimary)
	assert.False(t, emails[1].IsPrimary)
	assert.True(t, emails[2].IsActivated)
	assert.False(t, emails[2].IsPrimary)
}

func TestListEmails(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Must find all users and their emails
	opts := &user_model.SearchEmailOptions{
		ListOptions: db.ListOptions{
			PageSize: 10000,
		},
	}
	emails, count, err := user_model.SearchEmails(db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Greater(t, count, int64(5))

	contains := func(match func(s *user_model.SearchEmailResult) bool) bool {
		for _, v := range emails {
			if match(v) {
				return true
			}
		}
		return false
	}

	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 18 }))
	// 'org3' is an organization
	assert.False(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 3 }))

	// Must find no records
	opts = &user_model.SearchEmailOptions{Keyword: "NOTFOUND"}
	emails, count, err = user_model.SearchEmails(db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Must find users 'user2', 'user28', etc.
	opts = &user_model.SearchEmailOptions{Keyword: "user2"}
	emails, count, err = user_model.SearchEmails(db.DefaultContext, opts)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), count)
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 2 }))
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 27 }))

	// Must find only primary addresses (i.e. from the `user` table)
	opts = &user_model.SearchEmailOptions{IsPrimary: optional.Some(true)}
	emails, _, err = user_model.SearchEmails(db.DefaultContext, opts)
	require.NoError(t, err)
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.IsPrimary }))
	assert.False(t, contains(func(s *user_model.SearchEmailResult) bool { return !s.IsPrimary }))

	// Must find only inactive addresses (i.e. not validated)
	opts = &user_model.SearchEmailOptions{IsActivated: optional.Some(false)}
	emails, _, err = user_model.SearchEmails(db.DefaultContext, opts)
	require.NoError(t, err)
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return !s.IsActivated }))
	assert.False(t, contains(func(s *user_model.SearchEmailResult) bool { return s.IsActivated }))

	// Must find more than one page, but retrieve only one
	opts = &user_model.SearchEmailOptions{
		ListOptions: db.ListOptions{
			PageSize: 5,
			Page:     1,
		},
	}
	emails, count, err = user_model.SearchEmails(db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Len(t, emails, 5)
	assert.Greater(t, count, int64(len(emails)))
}

func TestGetActivatedEmailAddresses(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testCases := []struct {
		UID      int64
		expected []*user_model.ActivatedEmailAddress
	}{
		{
			UID:      1,
			expected: []*user_model.ActivatedEmailAddress{{ID: 9, Email: "user1@example.com"}, {ID: 33, Email: "user1-2@example.com"}, {ID: 34, Email: "user1-3@example.com"}},
		},
		{
			UID:      2,
			expected: []*user_model.ActivatedEmailAddress{{ID: 3, Email: "user2@example.com"}},
		},
		{
			UID:      4,
			expected: []*user_model.ActivatedEmailAddress{{ID: 11, Email: "user4@example.com"}},
		},
		{
			UID:      11,
			expected: []*user_model.ActivatedEmailAddress{},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("User %d", testCase.UID), func(t *testing.T) {
			emails, err := user_model.GetActivatedEmailAddresses(db.DefaultContext, testCase.UID)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, emails)
		})
	}
}

func TestDeletePrimaryEmailAddressOfUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user, err := user_model.GetUserByName(db.DefaultContext, "org3")
	require.NoError(t, err)
	assert.Equal(t, "org3@example.com", user.Email)

	require.NoError(t, user_model.DeletePrimaryEmailAddressOfUser(db.DefaultContext, user.ID))

	user, err = user_model.GetUserByName(db.DefaultContext, "org3")
	require.NoError(t, err)
	assert.Empty(t, user.Email)

	email, err := user_model.GetPrimaryEmailAddressOfUser(db.DefaultContext, user.ID)
	assert.True(t, user_model.IsErrEmailAddressNotExist(err))
	assert.Nil(t, email)
}
