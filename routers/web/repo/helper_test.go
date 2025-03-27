// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
)

func TestMakeSelfOnTop(t *testing.T) {
	users := MakeSelfOnTop(nil, []*user.User{{ID: 2}, {ID: 1}})
	assert.Len(t, users, 2)
	assert.EqualValues(t, 2, users[0].ID)

	users = MakeSelfOnTop(&user.User{ID: 1}, []*user.User{{ID: 2}, {ID: 1}})
	assert.Len(t, users, 2)
	assert.EqualValues(t, 1, users[0].ID)

	users = MakeSelfOnTop(&user.User{ID: 2}, []*user.User{{ID: 2}, {ID: 1}})
	assert.Len(t, users, 2)
	assert.EqualValues(t, 2, users[0].ID)

	users = MakeSelfOnTop(&user.User{ID: 2}, []*user.User{{ID: 1}})
	assert.Len(t, users, 1)
	assert.EqualValues(t, 1, users[0].ID)

	users = MakeSelfOnTop(&user.User{ID: 2}, []*user.User{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}})
	assert.Len(t, users, 4)
	assert.EqualValues(t, 2, users[0].ID)
	assert.EqualValues(t, 1, users[1].ID)
	assert.EqualValues(t, 3, users[2].ID)
	assert.EqualValues(t, 4, users[3].ID)
}
