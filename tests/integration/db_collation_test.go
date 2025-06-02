// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestCollationTbl struct {
	ID  int64
	Txt string `xorm:"VARCHAR(10) UNIQUE"`
}

func TestDatabaseCollationSelfCheckUI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	assertSelfCheckExists := func(exists bool) {
		expectedHTTPResponse := http.StatusOK
		if !exists {
			expectedHTTPResponse = http.StatusNotFound
		}
		session := loginUser(t, "user1")
		req := NewRequest(t, "GET", "/admin/self_check")
		resp := session.MakeRequest(t, req, expectedHTTPResponse)
		htmlDoc := NewHTMLParser(t, resp.Body)

		htmlDoc.AssertElement(t, "a.item[href*='/admin/self_check']", exists)
	}

	if setting.Database.Type.IsMySQL() {
		assertSelfCheckExists(true)
	} else {
		assertSelfCheckExists(false)
	}
}

func TestDatabaseCollation(t *testing.T) {
	engine, err := db.GetMasterEngine(db.GetEngine(db.DefaultContext))
	require.NoError(t, err)
	x := engine

	// all created tables should use case-sensitive collation by default
	_, _ = x.Exec("DROP TABLE IF EXISTS test_collation_tbl")
	err = x.Sync(&TestCollationTbl{})
	require.NoError(t, err)
	_, _ = x.Exec("INSERT INTO test_collation_tbl (txt) VALUES ('main')")
	_, _ = x.Exec("INSERT INTO test_collation_tbl (txt) VALUES ('Main')") // case-sensitive, so it inserts a new row
	_, _ = x.Exec("INSERT INTO test_collation_tbl (txt) VALUES ('main')") // duplicate, so it doesn't insert
	cnt, err := x.Count(&TestCollationTbl{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, cnt)
	_, _ = x.Exec("DROP TABLE IF EXISTS test_collation_tbl")

	// by default, SQLite3 and PostgreSQL are using case-sensitive collations, but MySQL is not.
	if !setting.Database.Type.IsMySQL() {
		t.Skip("only MySQL requires the case-sensitive collation check at the moment")
		return
	}

	t.Run("Default startup makes database collation case-sensitive", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		r, err := db.CheckCollations(x)
		require.NoError(t, err)
		assert.True(t, r.IsCollationCaseSensitive(r.DatabaseCollation))
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.NotEmpty(t, r.AvailableCollation)
		assert.Empty(t, r.InconsistentCollationColumns)

		// and by the way test the helper functions
		if setting.Database.Type.IsMySQL() {
			assert.True(t, r.IsCollationCaseSensitive("utf8mb4_bin"))
			assert.True(t, r.IsCollationCaseSensitive("utf8mb4_xxx_as_cs"))
			assert.False(t, r.IsCollationCaseSensitive("utf8mb4_general_ci"))
			assert.True(t, r.CollationEquals("abc", "abc"))
			assert.True(t, r.CollationEquals("abc", "utf8mb4_abc"))
			assert.False(t, r.CollationEquals("utf8mb4_general_ci", "utf8mb4_unicode_ci"))
		} else {
			assert.Fail(t, "unexpected database type")
		}
	})

	t.Run("Convert tables to utf8mb4_bin", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer test.MockVariableValue(&setting.Database.CharsetCollation, "utf8mb4_bin")()
		require.NoError(t, db.ConvertDatabaseTable())

		r, err := db.CheckCollations(x)
		require.NoError(t, err)
		assert.Equal(t, "utf8mb4_bin", r.DatabaseCollation)
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.Empty(t, r.InconsistentCollationColumns)

		_, _ = x.Exec("DROP TABLE IF EXISTS test_tbl")
		_, err = x.Exec("CREATE TABLE test_tbl (txt varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL)")
		require.NoError(t, err)
		r, err = db.CheckCollations(x)
		require.NoError(t, err)
		assert.Contains(t, r.InconsistentCollationColumns, "test_tbl.txt")
	})

	t.Run("Convert tables to utf8mb4_general_ci", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer test.MockVariableValue(&setting.Database.CharsetCollation, "utf8mb4_general_ci")()
		require.NoError(t, db.ConvertDatabaseTable())

		r, err := db.CheckCollations(x)
		require.NoError(t, err)
		assert.Equal(t, "utf8mb4_general_ci", r.DatabaseCollation)
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.Empty(t, r.InconsistentCollationColumns)

		_, _ = x.Exec("DROP TABLE IF EXISTS test_tbl")
		_, err = x.Exec("CREATE TABLE test_tbl (txt varchar(10) COLLATE utf8mb4_bin NOT NULL)")
		require.NoError(t, err)
		r, err = db.CheckCollations(x)
		require.NoError(t, err)
		assert.Contains(t, r.InconsistentCollationColumns, "test_tbl.txt")
	})

	t.Run("Convert tables to default case-sensitive collation", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer test.MockVariableValue(&setting.Database.CharsetCollation, "")()
		require.NoError(t, db.ConvertDatabaseTable())

		r, err := db.CheckCollations(x)
		require.NoError(t, err)
		assert.True(t, r.IsCollationCaseSensitive(r.DatabaseCollation))
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.Empty(t, r.InconsistentCollationColumns)
	})
}
