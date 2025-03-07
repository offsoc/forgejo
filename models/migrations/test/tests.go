// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//nolint:forbidigo
package test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/testlogger"
	"code.gitea.io/gitea/modules/util"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// PrepareTestEnv prepares the test environment and reset the database. The skip parameter should usually be 0.
// Provide models to be sync'd with the database - in particular any models you expect fixtures to be loaded from.
//
// fixtures in `models/migrations/fixtures/<TestName>` will be loaded automatically
func PrepareTestEnv(t *testing.T, skip int, syncModels ...any) (*xorm.Engine, func()) {
	t.Helper()
	ourSkip := 2
	ourSkip += skip
	deferFn := testlogger.PrintCurrentTest(t, ourSkip)
	require.NoError(t, os.RemoveAll(setting.RepoRootPath))
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

	if err := deleteDB(); err != nil {
		t.Errorf("unable to reset database: %v", err)
		return nil, deferFn
	}

	x, err := newXORMEngine()
	require.NoError(t, err)
	if x != nil {
		oldDefer := deferFn
		deferFn = func() {
			oldDefer()
			if err := x.Close(); err != nil {
				t.Errorf("error during close: %v", err)
			}
			if err := deleteDB(); err != nil {
				t.Errorf("unable to reset database: %v", err)
			}
		}
	}
	if err != nil {
		return x, deferFn
	}

	if len(syncModels) > 0 {
		if err := x.Sync(syncModels...); err != nil {
			t.Errorf("error during sync: %v", err)
			return x, deferFn
		}
	}

	fixturesDir := filepath.Join(filepath.Dir(setting.AppPath), "models", "migrations", "fixtures", t.Name())

	if _, err := os.Stat(fixturesDir); err == nil {
		t.Logf("initializing fixtures from: %s", fixturesDir)
		if err := unittest.InitFixtures(
			unittest.FixturesOptions{
				Dir: fixturesDir,
			}, x); err != nil {
			t.Errorf("error whilst initializing fixtures from %s: %v", fixturesDir, err)
			return x, deferFn
		}
		if err := unittest.LoadFixtures(x); err != nil {
			t.Errorf("error whilst loading fixtures from %s: %v", fixturesDir, err)
			return x, deferFn
		}
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error whilst checking for existence of fixtures: %v", err)
	} else {
		t.Logf("no fixtures found in: %s", fixturesDir)
	}

	return x, deferFn
}

func MainTest(m *testing.M) {
	log.RegisterEventWriter("test", testlogger.NewTestLoggerWriter)

	giteaRoot := base.SetupGiteaRoot()
	if giteaRoot == "" {
		fmt.Println("Environment variable $GITEA_ROOT not set")
		os.Exit(1)
	}
	giteaBinary := "gitea"
	setting.AppPath = path.Join(giteaRoot, giteaBinary)
	if _, err := os.Stat(setting.AppPath); err != nil {
		fmt.Printf("Could not find gitea binary at %s\n", setting.AppPath)
		os.Exit(1)
	}

	giteaConf := os.Getenv("GITEA_CONF")
	if giteaConf == "" {
		giteaConf = path.Join(filepath.Dir(setting.AppPath), "tests/sqlite.ini")
		fmt.Printf("Environment variable $GITEA_CONF not set - defaulting to %s\n", giteaConf)
	}

	if !path.IsAbs(giteaConf) {
		setting.CustomConf = path.Join(giteaRoot, giteaConf)
	} else {
		setting.CustomConf = giteaConf
	}

	tmpDataPath, err := os.MkdirTemp("", "data")
	if err != nil {
		fmt.Printf("Unable to create temporary data path %v\n", err)
		os.Exit(1)
	}

	setting.CustomPath = filepath.Join(setting.AppWorkPath, "custom")
	setting.AppDataPath = tmpDataPath

	unittest.InitSettings()
	if err = git.InitFull(context.Background()); err != nil {
		fmt.Printf("Unable to InitFull: %v\n", err)
		os.Exit(1)
	}
	setting.LoadDBSetting()
	setting.InitLoggersForTest()

	exitStatus := m.Run()

	if err := testlogger.WriterCloser.Reset(); err != nil && exitStatus == 0 {
		fmt.Printf("testlogger.WriterCloser.Reset: error ignored: %v\n", err)
	}
	if err := removeAllWithRetry(setting.RepoRootPath); err != nil {
		fmt.Fprintf(os.Stderr, "os.RemoveAll: %v\n", err)
	}
	if err := removeAllWithRetry(tmpDataPath); err != nil {
		fmt.Fprintf(os.Stderr, "os.RemoveAll: %v\n", err)
	}
	os.Exit(exitStatus)
}

func newXORMEngine() (*xorm.Engine, error) {
	if err := db.InitEngine(context.Background()); err != nil {
		return nil, err
	}
	x := unittest.GetXORMEngine()
	return x, nil
}

func deleteDB() error {
	switch {
	case setting.Database.Type.IsSQLite3():
		if err := util.Remove(setting.Database.Path); err != nil {
			return err
		}
		return os.MkdirAll(path.Dir(setting.Database.Path), os.ModePerm)

	case setting.Database.Type.IsMySQL():
		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/",
			setting.Database.User, setting.Database.Passwd, setting.Database.Host))
		if err != nil {
			return err
		}
		defer db.Close()

		databaseName := strings.SplitN(setting.Database.Name, "?", 2)[0]

		if _, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName)); err != nil {
			return err
		}

		if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", databaseName)); err != nil {
			return err
		}
		return nil
	case setting.Database.Type.IsPostgreSQL():
		db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/?sslmode=%s",
			setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.SSLMode))
		if err != nil {
			return err
		}
		defer db.Close()

		if _, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", setting.Database.Name)); err != nil {
			return err
		}

		if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", setting.Database.Name)); err != nil {
			return err
		}
		db.Close()

		// Check if we need to setup a specific schema
		if len(setting.Database.Schema) != 0 {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.Name, setting.Database.SSLMode))
			if err != nil {
				return err
			}
			defer db.Close()

			schrows, err := db.Query(fmt.Sprintf("SELECT 1 FROM information_schema.schemata WHERE schema_name = '%s'", setting.Database.Schema))
			if err != nil {
				return err
			}
			defer schrows.Close()

			if !schrows.Next() {
				// Create and setup a DB schema
				_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", setting.Database.Schema))
				if err != nil {
					return err
				}
			}

			// Make the user's default search path the created schema; this will affect new connections
			_, err = db.Exec(fmt.Sprintf(`ALTER USER "%s" SET search_path = %s`, setting.Database.User, setting.Database.Schema))
			if err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func removeAllWithRetry(dir string) error {
	var err error
	for i := 0; i < 20; i++ {
		err = os.RemoveAll(dir)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return err
}
