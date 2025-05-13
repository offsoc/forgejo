// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package explore

import (
	"testing"

	"forgejo.org/models/db"

	"github.com/stretchr/testify/assert"
)

func TestMapSortOrder(t *testing.T) {
	assert.Equal(t, MapSortOrder("newest"), db.SearchOrderBy("`user`.created_unix DESC"))
	assert.Equal(t, MapSortOrder("oldest"), db.SearchOrderBy("`user`.created_unix ASC"))
	assert.Equal(t, MapSortOrder("leastupdate"), db.SearchOrderBy("`user`.updated_unix ASC"))
	assert.Equal(t, MapSortOrder("reversealphabetically"), db.SearchOrderBy("`user`.name DESC"))
	assert.Equal(t, MapSortOrder("lastlogin"), db.SearchOrderBy("`user`.last_login_unix ASC"))
	assert.Equal(t, MapSortOrder("reverselastlogin"), db.SearchOrderBy("`user`.last_login_unix DESC"))
	assert.Equal(t, MapSortOrder("alphabetically"), db.SearchOrderBy("`user`.name ASC"))
	assert.Equal(t, MapSortOrder("recentupdate"), db.SearchOrderBy("`user`.updated_unix DESC"))
}
