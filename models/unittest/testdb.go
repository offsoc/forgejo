// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package unittest

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/system"
	"forgejo.org/modules/auth/password/hash"
	"forgejo.org/modules/base"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/setting/config"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/util"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
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
	setting.InitGiteaEnvVars()

	// Avoid loading the git's system config.
	// On macOS, system config sets the osxkeychain credential helper, which will cause tests to freeze with a dialog.
	// But we do not set it in production at the moment, because it might be a "breaking" change,
	// more details are in "modules/git.commonBaseEnvs".
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "true")
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

	if err := CreateTestEngine(opts); err != nil {
		fatalTestError("Error creating test engine: %v\n", err)
	}

	setting.AppURL = "https://try.gitea.io/"
	setting.RunUser = "runuser"
	setting.SSH.User = "sshuser"
	setting.SSH.BuiltinServerUser = "builtinuser"
	setting.SSH.Port = 3000
	setting.SSH.Domain = "try.gitea.io"
	setting.Database.Type = "sqlite3"
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
	x, err := xorm.NewEngine("sqlite3", "file::memory:?cache=shared&_txlock=immediate")
	if err != nil {
		if strings.Contains(err.Error(), "unknown driver") {
			return fmt.Errorf(`sqlite3 requires: import _ "github.com/mattn/go-sqlite3" or -tags sqlite,sqlite_unlock_notify%s%w`, "\n", err)
		}
		return err
	}
	x.SetMapper(names.GonicMapper{})
	db.SetDefaultEngine(context.Background(), x)

	if err = db.SyncAllTables(); err != nil {
		return err
	}
	switch os.Getenv("GITEA_UNIT_TESTS_LOG_SQL") {
	case "true", "1":
		x.ShowSQL(true)
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
