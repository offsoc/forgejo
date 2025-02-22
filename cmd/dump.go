// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/util"

	"code.forgejo.org/go-chi/session"
	"github.com/mholt/archives"
	"github.com/urfave/cli/v2"
)

func addObject(archiveJobs chan archives.ArchiveAsyncJob, object fs.File, customName string, verbose bool) error {
	if verbose {
		log.Info("Adding file %s", customName)
	}

	info, err := object.Stat()
	if err != nil {
		return err
	}

	ch := make(chan error)

	archiveJobs <- archives.ArchiveAsyncJob{
		File: archives.FileInfo{
			FileInfo:      info,
			NameInArchive: customName,
			Open: func() (fs.File, error) {
				return object, nil
			},
		},
		Result: ch,
	}

	return <-ch
}

func addFile(archiveJobs chan archives.ArchiveAsyncJob, filePath, absPath string, verbose bool) error {
	file, err := os.Open(absPath) // Closed by archiver
	if err != nil {
		return err
	}

	return addObject(archiveJobs, file, filePath, verbose)
}

func isSubdir(upper, lower string) (bool, error) {
	if relPath, err := filepath.Rel(upper, lower); err != nil {
		return false, err
	} else if relPath == "." || !strings.HasPrefix(relPath, ".") {
		return true, nil
	}
	return false, nil
}

type outputType struct {
	Enum     []string
	Default  string
	selected string
}

func (o outputType) Join() string {
	return strings.Join(o.Enum, ", ")
}

func (o *outputType) Set(value string) error {
	for _, enum := range o.Enum {
		if enum == value {
			o.selected = value
			return nil
		}
	}

	return fmt.Errorf("allowed values are %s", o.Join())
}

func (o outputType) String() string {
	if o.selected == "" {
		return o.Default
	}
	return o.selected
}

var outputTypeEnum = &outputType{
	Enum:    []string{"zip", "tar", "tar.sz", "tar.gz", "tar.xz", "tar.bz2", "tar.br", "tar.lz4", "tar.zst"},
	Default: "zip",
}

func getArchiverByType(outType string) (archives.ArchiverAsync, error) {
	var archiver archives.ArchiverAsync
	switch outType {
	case "zip":
		archiver = archives.Zip{}
	case "tar":
		archiver = archives.Tar{}
	case "tar.sz":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Sz{},
		}
	case "tar.gz":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Gz{},
		}
	case "tar.xz":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Xz{},
		}
	case "tar.bz2":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Bz2{},
		}
	case "tar.br":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Brotli{},
		}
	case "tar.lz4":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Lz4{},
		}
	case "tar.zst":
		archiver = archives.CompressedArchive{
			Archival:    archives.Tar{},
			Compression: archives.Zstd{},
		}
	default:
		return nil, fmt.Errorf("unsupported output type: %s", outType)
	}
	return archiver, nil
}

// CmdDump represents the available dump sub-command.
var CmdDump = &cli.Command{
	Name:  "dump",
	Usage: "Dump Forgejo files and database",
	Description: `Dump compresses all related files and database into zip file.
It can be used for backup and capture Forgejo server image to send to maintainer`,
	Action: runDump,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "file",
			Aliases: []string{"f"},
			Value:   fmt.Sprintf("forgejo-dump-%d.zip", time.Now().Unix()),
			Usage:   "Name of the dump file which will be created. Supply '-' for stdout. See type for available types.",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"V"},
			Usage:   "Show process details",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display warnings and errors",
		},
		&cli.StringFlag{
			Name:    "tempdir",
			Aliases: []string{"t"},
			Value:   os.TempDir(),
			Usage:   "Temporary dir path",
		},
		&cli.StringFlag{
			Name:    "database",
			Aliases: []string{"d"},
			Usage:   "Specify the database SQL syntax: sqlite3, mysql, postgres",
		},
		&cli.BoolFlag{
			Name:    "skip-repository",
			Aliases: []string{"R"},
			Usage:   "Skip repositories",
		},
		&cli.BoolFlag{
			Name:    "skip-log",
			Aliases: []string{"L"},
			Usage:   "Skip logs",
		},
		&cli.BoolFlag{
			Name:  "skip-custom-dir",
			Usage: "Skip custom directory",
		},
		&cli.BoolFlag{
			Name:  "skip-lfs-data",
			Usage: "Skip LFS data",
		},
		&cli.BoolFlag{
			Name:  "skip-attachment-data",
			Usage: "Skip attachment data",
		},
		&cli.BoolFlag{
			Name:  "skip-package-data",
			Usage: "Skip package data",
		},
		&cli.BoolFlag{
			Name:  "skip-index",
			Usage: "Skip bleve index data",
		},
		&cli.BoolFlag{
			Name:  "skip-repo-archives",
			Usage: "Skip repository archives",
		},
		&cli.GenericFlag{
			Name:  "type",
			Value: outputTypeEnum,
			Usage: fmt.Sprintf("Dump output format: %s", outputTypeEnum.Join()),
		},
	},
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	log.Fatal(format, args...)
}

func runDump(ctx *cli.Context) error {
	var file *os.File
	fileName := ctx.String("file")
	outType := ctx.String("type")
	if fileName == "-" {
		file = os.Stdout
		setupConsoleLogger(log.FATAL, log.CanColorStderr, os.Stderr)
	} else {
		for _, suffix := range outputTypeEnum.Enum {
			if strings.HasSuffix(fileName, "."+suffix) {
				fileName = strings.TrimSuffix(fileName, "."+suffix)
				break
			}
		}
		fileName += "." + outType
	}
	setting.MustInstalled()

	// make sure we are logging to the console no matter what the configuration tells us do to
	// FIXME: don't use CfgProvider directly
	if _, err := setting.CfgProvider.Section("log").NewKey("MODE", "console"); err != nil {
		fatal("Setting logging mode to console failed: %v", err)
	}
	if _, err := setting.CfgProvider.Section("log.console").NewKey("STDERR", "true"); err != nil {
		fatal("Setting console logger to stderr failed: %v", err)
	}

	// Set loglevel to Warn if quiet-mode is requested
	if ctx.Bool("quiet") {
		if _, err := setting.CfgProvider.Section("log.console").NewKey("LEVEL", "Warn"); err != nil {
			fatal("Setting console log-level failed: %v", err)
		}
	}

	if !setting.InstallLock {
		log.Error("Is '%s' really the right config path?\n", setting.CustomConf)
		return fmt.Errorf("forgejo is not initialized")
	}
	setting.LoadSettings() // cannot access session settings otherwise

	verbose := ctx.Bool("verbose")
	if verbose && ctx.Bool("quiet") {
		return fmt.Errorf("--quiet and --verbose cannot both be set")
	}

	stdCtx, cancel := installSignals()
	defer cancel()

	err := db.InitEngine(stdCtx)
	if err != nil {
		return err
	}

	if err := storage.Init(); err != nil {
		return err
	}

	if file == nil {
		file, err = os.Create(fileName)
		if err != nil {
			fatal("Failed to open %s: %v", fileName, err)
		}
	}
	defer file.Close()

	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		return err
	}

	archiveJobs := make(chan archives.ArchiveAsyncJob)
	wg := sync.WaitGroup{}
	archiver, err := getArchiverByType(outType)
	if err != nil {
		fatal("Failed to get archiver for extension: %v", err)
	}

	if ctx.IsSet("skip-repository") && ctx.Bool("skip-repository") {
		log.Info("Skipping local repositories")
	} else {
		log.Info("Dumping local repositories... %s", setting.RepoRootPath)
		wg.Add(1)
		go dumpRepos(ctx, archiveJobs, &wg, absFileName, verbose)
	}

	wg.Add(1)
	go dumpDatabase(ctx, archiveJobs, &wg, verbose)

	if len(setting.CustomConf) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("Adding custom configuration file from %s", setting.CustomConf)
			if err := addFile(archiveJobs, "app.ini", setting.CustomConf, verbose); err != nil {
				fatal("Failed to include specified app.ini: %v", err)
			}
		}()
	}

	if ctx.IsSet("skip-custom-dir") && ctx.Bool("skip-custom-dir") {
		log.Info("Skipping custom directory")
	} else {
		wg.Add(1)
		go dumpCustom(archiveJobs, &wg, absFileName, verbose)
	}

	isExist, err := util.IsExist(setting.AppDataPath)
	if err != nil {
		log.Error("Failed to check if %s exists: %v", setting.AppDataPath, err)
	}
	if isExist {
		log.Info("Packing data directory...%s", setting.AppDataPath)

		wg.Add(1)
		go dumpData(ctx, archiveJobs, &wg, absFileName, verbose)
	}

	if ctx.IsSet("skip-attachment-data") && ctx.Bool("skip-attachment-data") {
		log.Info("Skipping attachment data")
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := storage.Attachments.IterateObjects("", func(objPath string, object storage.Object) error {
				return addObject(archiveJobs, object, path.Join("data", "attachments", objPath), verbose)
			}); err != nil {
				fatal("Failed to dump attachments: %v", err)
			}
		}()
	}

	if ctx.IsSet("skip-package-data") && ctx.Bool("skip-package-data") {
		log.Info("Skipping package data")
	} else if !setting.Packages.Enabled {
		log.Info("Package registry not enabled - skipping")
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := storage.Packages.IterateObjects("", func(objPath string, object storage.Object) error {
				return addObject(archiveJobs, object, path.Join("data", "packages", objPath), verbose)
			}); err != nil {
				fatal("Failed to dump packages: %v", err)
			}
		}()
	}

	// Doesn't check if LogRootPath exists before processing --skip-log intentionally,
	// ensuring that it's clear the dump is skipped whether the directory's initialized
	// yet or not.
	if ctx.IsSet("skip-log") && ctx.Bool("skip-log") {
		log.Info("Skipping log files")
	} else {
		isExist, err := util.IsExist(setting.Log.RootPath)
		if err != nil {
			log.Error("Failed to check if %s exists: %v", setting.Log.RootPath, err)
		}
		if isExist {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := addRecursiveExclude(archiveJobs, "log", setting.Log.RootPath, []string{absFileName}, verbose); err != nil {
					fatal("Failed to include log: %v", err)
				}
			}()
		}
	}

	// Wait for all jobs to finish before closing the channel
	// ArchiveAsync will only return after the channel is closed
	go func() {
		wg.Wait()
		close(archiveJobs)
	}()

	if err := archiver.ArchiveAsync(ctx.Context, file, archiveJobs); err != nil {
		_ = util.Remove(fileName)

		fatal("Archiving failed: %v", err)
	}

	if fileName != "-" {
		if err := os.Chmod(fileName, 0o600); err != nil {
			log.Info("Can't change file access permissions mask to 0600: %v", err)
		}

		log.Info("Finished dumping in file %s", fileName)
	} else {
		log.Info("Finished dumping to stdout")
	}

	return nil
}

func dumpData(ctx *cli.Context, archiveJobs chan archives.ArchiveAsyncJob, wg *sync.WaitGroup, absFileName string, verbose bool) {
	defer wg.Done()

	var excludes []string
	if setting.SessionConfig.OriginalProvider == "file" {
		var opts session.Options
		if err := json.Unmarshal([]byte(setting.SessionConfig.ProviderConfig), &opts); err != nil {
			fatal("Failed to parse session config: %v", err)
		}
		excludes = append(excludes, opts.ProviderConfig)
	}

	if ctx.IsSet("skip-index") && ctx.Bool("skip-index") {
		log.Info("Skipping bleve index data")
		excludes = append(excludes, setting.Indexer.RepoPath)
		excludes = append(excludes, setting.Indexer.IssuePath)
	}

	if ctx.IsSet("skip-repo-archives") && ctx.Bool("skip-repo-archives") {
		log.Info("Skipping repository archives data")
		excludes = append(excludes, setting.RepoArchive.Storage.Path)
	}

	excludes = append(excludes, setting.RepoRootPath)
	excludes = append(excludes, setting.LFS.Storage.Path)
	excludes = append(excludes, setting.Attachment.Storage.Path)
	excludes = append(excludes, setting.Packages.Storage.Path)
	excludes = append(excludes, setting.Log.RootPath)
	excludes = append(excludes, absFileName)
	if err := addRecursiveExclude(archiveJobs, "data", setting.AppDataPath, excludes, verbose); err != nil {
		fatal("Failed to include data directory: %v", err)
	}
}

func dumpCustom(archiveJobs chan archives.ArchiveAsyncJob, wg *sync.WaitGroup, absFileName string, verbose bool) {
	defer wg.Done()

	customDir, err := os.Stat(setting.CustomPath)
	if err == nil && customDir.IsDir() {
		if is, _ := isSubdir(setting.AppDataPath, setting.CustomPath); !is {
			if err := addRecursiveExclude(archiveJobs, "custom", setting.CustomPath, []string{absFileName}, verbose); err != nil {
				fatal("Failed to include custom: %v", err)
			}
		} else {
			log.Info("Custom dir %s is inside data dir %s, skipping", setting.CustomPath, setting.AppDataPath)
		}
	} else {
		log.Info("Custom dir %s does not exist, skipping", setting.CustomPath)
	}
}

func dumpDatabase(ctx *cli.Context, archiveJobs chan archives.ArchiveAsyncJob, wg *sync.WaitGroup, verbose bool) {
	defer wg.Done()

	tmpDir := ctx.String("tempdir")
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		fatal("Path does not exist: %s", tmpDir)
	}

	dbDump, err := os.CreateTemp(tmpDir, "forgejo-db.sql")
	if err != nil {
		fatal("Failed to create tmp file: %v", err)
	}
	defer func() {
		_ = dbDump.Close()
		if err := util.Remove(dbDump.Name()); err != nil {
			log.Warn("Failed to remove temporary file: %s: Error: %v", dbDump.Name(), err)
		}
	}()

	targetDBType := ctx.String("database")
	if len(targetDBType) > 0 && targetDBType != setting.Database.Type.String() {
		log.Info("Dumping database %s => %s...", setting.Database.Type, targetDBType)
	} else {
		log.Info("Dumping database...")
	}

	if err := db.DumpDatabase(dbDump.Name(), targetDBType); err != nil {
		fatal("Failed to dump database: %v", err)
	}

	if err := addFile(archiveJobs, "forgejo-db.sql", dbDump.Name(), verbose); err != nil {
		fatal("Failed to include forgejo-db.sql: %v", err)
	}
}

func dumpRepos(ctx *cli.Context, archiveJobs chan archives.ArchiveAsyncJob, wg *sync.WaitGroup, absFileName string, verbose bool) {
	defer wg.Done()

	if err := addRecursiveExclude(archiveJobs, "repos", setting.RepoRootPath, []string{absFileName}, verbose); err != nil {
		fatal("Failed to include repositories: %v", err)
	}

	if ctx.IsSet("skip-lfs-data") && ctx.Bool("skip-lfs-data") {
		log.Info("Skipping LFS data")
	} else if !setting.LFS.StartServer {
		log.Info("LFS not enabled - skipping")
	} else if err := storage.LFS.IterateObjects("", func(objPath string, object storage.Object) error {
		return addObject(archiveJobs, object, path.Join("data", "lfs", objPath), verbose)
	}); err != nil {
		fatal("Failed to dump LFS objects: %v", err)
	}
}

// addRecursiveExclude zips absPath to specified insidePath inside writer excluding excludeAbsPath
// archives.FilesFromDisk doesn't support excluding files, so we have to do it manually
func addRecursiveExclude(archiveJobs chan archives.ArchiveAsyncJob, insidePath, absPath string, excludeAbsPath []string, verbose bool) error {
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return err
	}
	dir, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, file := range files {
		currentAbsPath := filepath.Join(absPath, file.Name())
		currentInsidePath := path.Join(insidePath, file.Name())

		if util.SliceContainsString(excludeAbsPath, currentAbsPath) {
			log.Debug("Skipping %q (matched an excluded path)", currentAbsPath)
			continue
		}

		if file.IsDir() {
			if err := addFile(archiveJobs, currentInsidePath, currentAbsPath, false); err != nil {
				return err
			}

			if err := addRecursiveExclude(archiveJobs, currentInsidePath, currentAbsPath, excludeAbsPath, verbose); err != nil {
				return err
			}
		} else {
			// only copy regular files and symlink regular files, skip non-regular files like socket/pipe/...
			shouldAdd := file.Mode().IsRegular()
			if !shouldAdd && file.Mode()&os.ModeSymlink == os.ModeSymlink {
				target, err := filepath.EvalSymlinks(currentAbsPath)
				if err != nil {
					return err
				}
				targetStat, err := os.Stat(target)
				if err != nil {
					return err
				}
				shouldAdd = targetStat.Mode().IsRegular()
			}
			if shouldAdd {
				if err := addFile(archiveJobs, currentInsidePath, currentAbsPath, verbose); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
