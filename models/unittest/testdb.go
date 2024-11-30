// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package unittest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/system"
	"code.gitea.io/gitea/modules/auth/password/hash"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/setting/config"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/util"

	"github.com/stretchr/testify/require"
	//	"xorm.io/xorm"
	//"xorm.io/xorm/names"
)

// giteaRoot a path to the gitea root
var (
	giteaRoot   string
	fixturesDir string
)

// FixturesDir returns the fixture directory
func FixturesDir() string {
	return fixturesDir
}

func fatalTestError(fmtStr string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, fmtStr, args...)
	os.Exit(1)
}

// InitSettings initializes config provider and load common settings for tests
func InitSettings() {
	if setting.CustomConf == "" {
		setting.CustomConf = filepath.Join(setting.CustomPath, "conf/app-unittest-tmp.ini")
		_ = os.Remove(setting.CustomConf)
	}
	setting.InitCfgProvider(setting.CustomConf)
	setting.LoadCommonSettings()

	if err := setting.PrepareAppDataPath(); err != nil {
		log.Fatalf("Can not prepare APP_DATA_PATH: %v", err)
	}
	// register the dummy hash algorithm function used in the test fixtures
	_ = hash.Register("dummy", hash.NewDummyHasher)

	setting.PasswordHashAlgo, _ = hash.SetDefaultPasswordHashAlgorithm("dummy")
}

// TestOptions represents test options
type TestOptions struct {
	FixtureFiles []string
	SetUp        func() error // SetUp will be executed before all tests in this package
	TearDown     func() error // TearDown will be executed after all tests in this package
}

// MainTest a reusable TestMain(..) function for unit tests that need to use a
// test database. Creates the test database, and sets necessary settings.
func MainTest(m *testing.M, testOpts ...*TestOptions) {
	searchDir, _ := os.Getwd()
	for searchDir != "" {
		if _, err := os.Stat(filepath.Join(searchDir, "go.mod")); err == nil {
			break // The "go.mod" should be the one for Gitea repository
		}
		if dir := filepath.Dir(searchDir); dir == searchDir {
			searchDir = "" // reaches the root of filesystem
		} else {
			searchDir = dir
		}
	}
	if searchDir == "" {
		panic("The tests should run in a Gitea repository, there should be a 'go.mod' in the root")
	}

	giteaRoot = searchDir
	setting.CustomPath = filepath.Join(giteaRoot, "custom")
	InitSettings()

	fixturesDir = filepath.Join(giteaRoot, "models", "fixtures")
	var opts FixturesOptions
	if len(testOpts) == 0 || len(testOpts[0].FixtureFiles) == 0 {
		opts.Dir = fixturesDir
	} else {
		for _, f := range testOpts[0].FixtureFiles {
			if len(f) != 0 {
				opts.Files = append(opts.Files, filepath.Join(fixturesDir, f))
			}
		}
	}

	setting.AppURL = "https://try.gitea.io/"
	setting.RunUser = "runuser"
	setting.SSH.User = "sshuser"
	setting.SSH.BuiltinServerUser = "builtinuser"
	setting.SSH.Port = 3000
	setting.SSH.Domain = "try.gitea.io"
	// setting.Database.Type = "sqlite3"
	setting.Database.Type = "postgres"
	setting.Database.Host = "172.23.0.3:5432"
	// setting.Database.Host = "pgsql:5432"
	setting.Database.Name = "testgitea"
	setting.Database.User = "postgres"
	setting.Database.Passwd = "postgres"
	setting.Database.Schema = "gtestschema"
	setting.Database.SSLMode = "disable"

	switch {
	case setting.Database.Type.IsMySQL():
		connType := "tcp"
		if len(setting.Database.Host) > 0 && setting.Database.Host[0] == '/' { // looks like a unix socket
			connType = "unix"
		}

		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s(%s)/",
			setting.Database.User, setting.Database.Passwd, connType, setting.Database.Host))
		defer db.Close()
		if err != nil {
			log.Fatal("sql.Open: %v", err)
		}
		if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", strings.SplitN(setting.Database.Name, "?", 2)[0])); err != nil {
			log.Fatal("db.Exec: %v", err)
		}
	case setting.Database.Type.IsPostgreSQL():
		var db *sql.DB
		var err error
		if setting.Database.Host[0] == '/' {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@/%s?sslmode=%s&host=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Name, setting.Database.SSLMode, setting.Database.Host))
		} else {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s?sslmode=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.SSLMode))
		}

		defer db.Close()
		if err != nil {
			log.Fatal("sql.Open: %v", err)
		}
		dbrows, err := db.Query(fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", setting.Database.Name))
		if err != nil {
			log.Fatalf("db.Query: %v", err)
		}
		defer dbrows.Close()

		if !dbrows.Next() {
			if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", setting.Database.Name)); err != nil {
				log.Fatal("db.Exec: CREATE DATABASE: %v", err)
			}
		}
		// Check if we need to setup a specific schema
		if len(setting.Database.Schema) == 0 {
			break
		}
		db.Close()

		if setting.Database.Host[0] == '/' {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@/%s?sslmode=%s&host=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Name, setting.Database.SSLMode, setting.Database.Host))
		} else {
			db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
				setting.Database.User, setting.Database.Passwd, setting.Database.Host, setting.Database.Name, setting.Database.SSLMode))
		}
		// This is a different db object; requires a different Close()
		defer db.Close()
		if err != nil {
			log.Fatal("sql.Open: %v", err)
		}
		schrows, err := db.Query(fmt.Sprintf("SELECT 1 FROM information_schema.schemata WHERE schema_name = '%s'", setting.Database.Schema))
		if err != nil {
			log.Fatal("db.Query: %v", err)
		}
		defer schrows.Close()

		if !schrows.Next() {
			// Create and setup a DB schema
			if _, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", setting.Database.Schema)); err != nil {
				log.Fatal("db.Exec: CREATE SCHEMA: %v", err)
			}
		}

	}

	if err := CreateTestEngine(opts); err != nil {
		fatalTestError("Error creating test engine: %v\n", err)
	}
	setting.Repository.DefaultBranch = "master" // many test code still assume that default branch is called "master"
	repoRootPath, err := os.MkdirTemp(os.TempDir(), "repos")
	if err != nil {
		fatalTestError("TempDir: %v\n", err)
	}
	setting.RepoRootPath = repoRootPath
	appDataPath, err := os.MkdirTemp(os.TempDir(), "appdata")
	if err != nil {
		fatalTestError("TempDir: %v\n", err)
	}
	setting.AppDataPath = appDataPath
	setting.AppWorkPath = giteaRoot
	setting.StaticRootPath = giteaRoot
	setting.GravatarSource = "https://secure.gravatar.com/avatar/"

	setting.Attachment.Storage.Path = filepath.Join(setting.AppDataPath, "attachments")

	setting.LFS.Storage.Path = filepath.Join(setting.AppDataPath, "lfs")

	setting.Avatar.Storage.Path = filepath.Join(setting.AppDataPath, "avatars")

	setting.RepoAvatar.Storage.Path = filepath.Join(setting.AppDataPath, "repo-avatars")

	setting.RepoArchive.Storage.Path = filepath.Join(setting.AppDataPath, "repo-archive")

	setting.Packages.Storage.Path = filepath.Join(setting.AppDataPath, "packages")

	setting.Actions.LogStorage.Path = filepath.Join(setting.AppDataPath, "actions_log")

	setting.Git.HomePath = filepath.Join(setting.AppDataPath, "home")

	setting.IncomingEmail.ReplyToAddress = "incoming+%{token}@localhost"

	config.SetDynGetter(system.NewDatabaseDynKeyGetter())

	if err = storage.Init(); err != nil {
		fatalTestError("storage.Init: %v\n", err)
	}
	if err = util.RemoveAll(repoRootPath); err != nil {
		fatalTestError("util.RemoveAll: %v\n", err)
	}
	if err = CopyDir(filepath.Join(giteaRoot, "tests", "gitea-repositories-meta"), setting.RepoRootPath); err != nil {
		fatalTestError("util.CopyDir: %v\n", err)
	}

	if err = git.InitFull(context.Background()); err != nil {
		fatalTestError("git.Init: %v\n", err)
	}
	ownerDirs, err := os.ReadDir(setting.RepoRootPath)
	if err != nil {
		fatalTestError("unable to read the new repo root: %v\n", err)
	}
	for _, ownerDir := range ownerDirs {
		if !ownerDir.Type().IsDir() {
			continue
		}
		repoDirs, err := os.ReadDir(filepath.Join(setting.RepoRootPath, ownerDir.Name()))
		if err != nil {
			fatalTestError("unable to read the new repo root: %v\n", err)
		}
		for _, repoDir := range repoDirs {
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "objects", "pack"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "objects", "info"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "refs", "heads"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "refs", "tag"), 0o755)
		}
	}

	if len(testOpts) > 0 && testOpts[0].SetUp != nil {
		if err := testOpts[0].SetUp(); err != nil {
			fatalTestError("set up failed: %v\n", err)
		}
	}

	exitStatus := m.Run()

	if len(testOpts) > 0 && testOpts[0].TearDown != nil {
		if err := testOpts[0].TearDown(); err != nil {
			fatalTestError("tear down failed: %v\n", err)
		}
	}

	if err = util.RemoveAll(repoRootPath); err != nil {
		fatalTestError("util.RemoveAll: %v\n", err)
	}
	if err = util.RemoveAll(appDataPath); err != nil {
		fatalTestError("util.RemoveAll: %v\n", err)
	}
	os.Exit(exitStatus)
}

// FixturesOptions fixtures needs to be loaded options
type FixturesOptions struct {
	Dir   string
	Files []string
	Dirs  []string
	Base  string
}

// CreateTestEngine creates a memory database and loads the fixture data from fixturesDir
func CreateTestEngine(opts FixturesOptions) error {
	err := db.InitEngine(context.Background())
	if err != nil {
		return err
	}

	if err = db.SyncAllTables(); err != nil {
		return err
	}

	return InitFixtures(opts)
}

// PrepareTestDatabase load test fixtures into test database
func PrepareTestDatabase() error {
	return LoadFixtures()
}

// PrepareTestEnv prepares the environment for unit tests. Can only be called
// by tests that use the above MainTest(..) function.
func PrepareTestEnv(t testing.TB) {
	require.NoError(t, PrepareTestDatabase())
	require.NoError(t, util.RemoveAll(setting.RepoRootPath))
	metaPath := filepath.Join(giteaRoot, "tests", "gitea-repositories-meta")
	require.NoError(t, CopyDir(metaPath, setting.RepoRootPath))
	ownerDirs, err := os.ReadDir(setting.RepoRootPath)
	require.NoError(t, err)
	for _, ownerDir := range ownerDirs {
		if !ownerDir.Type().IsDir() {
			continue
		}
		repoDirs, err := os.ReadDir(filepath.Join(setting.RepoRootPath, ownerDir.Name()))
		require.NoError(t, err)
		for _, repoDir := range repoDirs {
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "objects", "pack"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "objects", "info"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "refs", "heads"), 0o755)
			_ = os.MkdirAll(filepath.Join(setting.RepoRootPath, ownerDir.Name(), repoDir.Name(), "refs", "tag"), 0o755)
		}
	}

	base.SetupGiteaRoot() // Makes sure GITEA_ROOT is set
}
