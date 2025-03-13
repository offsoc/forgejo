// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migrations

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/migrations"
	migrate_base "code.gitea.io/gitea/models/migrations/base"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/testlogger"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

var currentEngine *xorm.Engine

func initMigrationTest(t *testing.T) func() {
	log.RegisterEventWriter("test", testlogger.NewTestLoggerWriter)

	deferFn := tests.PrintCurrentTest(t, 2)
	giteaRoot := base.SetupGiteaRoot()
	if giteaRoot == "" {
		tests.Printf("Environment variable $GITEA_ROOT not set\n")
		os.Exit(1)
	}
	setting.AppPath = path.Join(giteaRoot, "gitea")
	if _, err := os.Stat(setting.AppPath); err != nil {
		tests.Printf("Could not find gitea binary at %s\n", setting.AppPath)
		os.Exit(1)
	}

	giteaConf := os.Getenv("GITEA_CONF")
	if giteaConf == "" {
		tests.Printf("Environment variable $GITEA_CONF not set\n")
		os.Exit(1)
	} else if !path.IsAbs(giteaConf) {
		setting.CustomConf = path.Join(giteaRoot, giteaConf)
	} else {
		setting.CustomConf = giteaConf
	}

	unittest.InitSettings()

	assert.NotEmpty(t, setting.RepoRootPath)
	require.NoError(t, util.RemoveAll(setting.RepoRootPath))
	require.NoError(t, unittest.CopyDir(path.Join(filepath.Dir(setting.AppPath), "tests/gitea-repositories-meta"), setting.RepoRootPath))
	ownerDirs, err := os.ReadDir(setting.RepoRootPath)
	if err != nil {
		require.NoError(t, err, "unable to read the new repo root: %v\n", err)
	}
	for _, ownerDir := range ownerDirs {
		if !ownerDir.Type().IsDir() {
			continue
		}
		repoDirs, err := os.ReadDir(filepath.Join(setting.RepoRootPath, ownerDir.Name()))
		if err != nil {
			require.NoError(t, err, "unable to read the new repo root: %v\n", err)
		}
		for _, repoDir := range repoDirs {
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "objects", "pack"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "objects", "info"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "refs", "heads"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "refs", "tag"), 0o755)
		}
	}

	require.NoError(t, git.InitFull(t.Context()))
	setting.LoadDBSetting()
	setting.InitLoggersForTest()
	return deferFn
}

func availableVersions() ([]string, error) {
	migrationsDir, err := os.Open("tests/integration/migration-test")
	if err != nil {
		return nil, err
	}
	defer migrationsDir.Close()
	versionRE, err := regexp.Compile(".*-v(?P<version>.+)\\." + regexp.QuoteMeta(setting.Database.Type.String()) + "\\.sql.gz")
	if err != nil {
		return nil, err
	}

	filenames, err := migrationsDir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	versions := []string{}
	for _, filename := range filenames {
		if versionRE.MatchString(filename) {
			substrings := versionRE.FindStringSubmatch(filename)
			versions = append(versions, substrings[1])
		}
	}
	sort.Strings(versions)
	return versions, nil
}

func readSQLFromFile(version string) (string, error) {
	filename := fmt.Sprintf("tests/integration/migration-test/gitea-v%s.%s.sql.gz", version, setting.Database.Type)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		filename = fmt.Sprintf("tests/integration/migration-test/forgejo-v%s.%s.sql.gz", version, setting.Database.Type)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return "", nil
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	bytes, err := io.ReadAll(gr)
	if err != nil {
		return "", err
	}
	return string(charset.MaybeRemoveBOM(bytes, charset.ConvertOpts{})), nil
}

func restoreOldDB(t *testing.T, version string) bool {
	data, err := readSQLFromFile(version)
	require.NoError(t, err)
	if len(data) == 0 {
		tests.Printf("No db found to restore for %s version: %s\n", setting.Database.Type, version)
		return false
	}

	switch {
	case setting.Database.Type.IsSQLite3():
		util.Remove(setting.Database.Path)
		err := os.MkdirAll(path.Dir(setting.Database.Path), os.ModePerm)
		require.NoError(t, err)

		db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&mode=rwc&_busy_timeout=%d&_txlock=immediate", setting.Database.Path, setting.Database.Timeout))
		require.NoError(t, err)
		defer db.Close()

		_, err = db.Exec(data)
		require.NoError(t, err)
		db.Close()

	case setting.Database.Type.IsMySQL():
		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/",
			setting.Database.User, setting.Database.Passwd, setting.Database.Host))
		require.NoError(t, err)
		defer db.Close()

		databaseName := strings.SplitN(setting.Database.Name, "?", 2)[0]

		_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
		require.NoError(t, err)

		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", databaseName))
		require.NoError(t, err)
		db.Close()

		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s",
			setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.Name))
		require.NoError(t, err)
		defer db.Close()

		_, err = db.Exec(data)
		require.NoError(t, err)
		db.Close()

	case setting.Database.Type.IsPostgreSQL():
		var db *sql.DB
		var err error
		if setting.Database.Host[0] == '/' {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@/?sslmode=%s&host=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.SSLMode, setting.Database.Host))
			require.NoError(t, err)
		} else {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/?sslmode=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.SSLMode))
			require.NoError(t, err)
		}
		defer db.Close()

		_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", setting.Database.Name))
		require.NoError(t, err)

		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", setting.Database.Name))
		require.NoError(t, err)
		db.Close()

		// Check if we need to setup a specific schema
		if len(setting.Database.Schema) != 0 {
			if setting.Database.Host[0] == '/' {
				db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@/%s?sslmode=%s&host=%s",
					setting.Database.User, setting.Database.Passwd, setting.Database.Name, setting.Database.SSLMode, setting.Database.Host))
			} else {
				db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
					setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.Name, setting.Database.SSLMode))
			}
			require.NoError(t, err)

			defer db.Close()

			schrows, err := db.Query(fmt.Sprintf("SELECT 1 FROM information_schema.schemata WHERE schema_name = '%s'", setting.Database.Schema))
			require.NoError(t, err)
			if !assert.NotEmpty(t, schrows) {
				return false
			}

			if !schrows.Next() {
				// Create and setup a DB schema
				_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", setting.Database.Schema))
				require.NoError(t, err)
			}
			schrows.Close()

			// Make the user's default search path the created schema; this will affect new connections
			_, err = db.Exec(fmt.Sprintf(`ALTER USER "%s" SET search_path = %s`, setting.Database.User, setting.Database.Schema))
			require.NoError(t, err)

			db.Close()
		}

		if setting.Database.Host[0] == '/' {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@/%s?sslmode=%s&host=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Name, setting.Database.SSLMode, setting.Database.Host))
		} else {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.Name, setting.Database.SSLMode))
		}
		require.NoError(t, err)
		defer db.Close()

		_, err = db.Exec(data)
		require.NoError(t, err)
		db.Close()
	}
	return true
}

func wrappedMigrate(x *xorm.Engine) error {
	currentEngine = x
	return migrations.Migrate(x)
}

func doMigrationTest(t *testing.T, version string) {
	defer tests.PrintCurrentTest(t)()
	tests.Printf("Performing migration test for %s version: %s\n", setting.Database.Type, version)
	if !restoreOldDB(t, version) {
		return
	}

	setting.InitSQLLoggersForCli(log.INFO)

	err := db.InitEngineWithMigration(t.Context(), wrappedMigrate)
	require.NoError(t, err)
	currentEngine.Close()

	beans, _ := db.NamesToBean()

	err = db.InitEngineWithMigration(t.Context(), func(x *xorm.Engine) error {
		currentEngine = x
		return migrate_base.RecreateTables(beans...)(x)
	})
	require.NoError(t, err)
	currentEngine.Close()

	// We do this a second time to ensure that there is not a problem with retained indices
	err = db.InitEngineWithMigration(t.Context(), func(x *xorm.Engine) error {
		currentEngine = x
		return migrate_base.RecreateTables(beans...)(x)
	})
	require.NoError(t, err)

	currentEngine.Close()
}

func TestMigrations(t *testing.T) {
	defer initMigrationTest(t)()

	dialect := setting.Database.Type
	versions, err := availableVersions()
	require.NoError(t, err)

	if len(versions) == 0 {
		tests.Printf("No old database versions available to migration test for %s\n", dialect)
		return
	}

	tests.Printf("Preparing to test %d migrations for %s\n", len(versions), dialect)
	for _, version := range versions {
		t.Run(fmt.Sprintf("Migrate-%s-%s", dialect, version), func(t *testing.T) {
			doMigrationTest(t, version)
		})
	}
}
