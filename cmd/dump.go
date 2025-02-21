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

func addObject(object fs.File, customName string, verbose bool) (archives.FileInfo, error) {
	if verbose {
		log.Info("Adding object %s", customName)
	}

	info, err := object.Stat()
	if err != nil {
		return archives.FileInfo{}, err
	}

	return archives.FileInfo{
		FileInfo:      info,
		NameInArchive: customName,
		Open: func() (fs.File, error) {
			return object, nil
		},
	}, nil
}

func addFile(filePath, absPath string, verbose bool) (archives.FileInfo, error) {
	if verbose {
		log.Info("Adding file %s", filePath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return archives.FileInfo{}, err
	}

	return archives.FileInfo{
		FileInfo:      info,
		NameInArchive: filePath,
		Open: func() (fs.File, error) {
			// Only open the file when it's needed
			return os.Open(absPath)
		},
	}, nil
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

func getArchiverByType(outType string) (archives.Archiver, error) {
	var archiver archives.Archiver
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

	var files []archives.FileInfo
	archiver, err := getArchiverByType(outType)
	if err != nil {
		fatal("Failed to get archiver for extension: %v", err)
	}

	if ctx.IsSet("skip-repository") && ctx.Bool("skip-repository") {
		log.Info("Skipping local repositories")
	} else {
		log.Info("Dumping local repositories... %s", setting.RepoRootPath)
		l, err := addRecursiveExclude("repos", setting.RepoRootPath, []string{absFileName}, verbose)
		if err != nil {
			fatal("Failed to include repositories: %v", err)
		}
		files = append(files, l...)

		if ctx.IsSet("skip-lfs-data") && ctx.Bool("skip-lfs-data") {
			log.Info("Skipping LFS data")
		} else if !setting.LFS.StartServer {
			log.Info("LFS not enabled - skipping")
		} else if err := storage.LFS.IterateObjects("", func(objPath string, object storage.Object) error {
			f, err := addObject(object, path.Join("data", "lfs", objPath), verbose)
			if err != nil {
				return err
			}

			files = append(files, f)
			return nil
		}); err != nil {
			fatal("Failed to dump LFS objects: %v", err)
		}
	}

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

	f, err := addFile("forgejo-db.sql", dbDump.Name(), verbose)
	if err != nil {
		fatal("Failed to include forgejo-db.sql: %v", err)
	}
	files = append(files, f)

	if len(setting.CustomConf) > 0 {
		log.Info("Adding custom configuration file from %s", setting.CustomConf)
		f, err := addFile("app.ini", setting.CustomConf, verbose)
		if err != nil {
			fatal("Failed to include specified app.ini: %v", err)
		}
		files = append(files, f)
	}

	if ctx.IsSet("skip-custom-dir") && ctx.Bool("skip-custom-dir") {
		log.Info("Skipping custom directory")
	} else {
		customDir, err := os.Stat(setting.CustomPath)
		if err == nil && customDir.IsDir() {
			if is, _ := isSubdir(setting.AppDataPath, setting.CustomPath); !is {
				l, err := addRecursiveExclude("custom", setting.CustomPath, []string{absFileName}, verbose)
				if err != nil {
					fatal("Failed to include custom: %v", err)
				}
				files = append(files, l...)
			} else {
				log.Info("Custom dir %s is inside data dir %s, skipping", setting.CustomPath, setting.AppDataPath)
			}
		} else {
			log.Info("Custom dir %s does not exist, skipping", setting.CustomPath)
		}
	}

	isExist, err := util.IsExist(setting.AppDataPath)
	if err != nil {
		log.Error("Failed to check if %s exists: %v", setting.AppDataPath, err)
	}
	if isExist {
		log.Info("Packing data directory...%s", setting.AppDataPath)

		var excludes []string
		if setting.SessionConfig.OriginalProvider == "file" {
			var opts session.Options
			if err = json.Unmarshal([]byte(setting.SessionConfig.ProviderConfig), &opts); err != nil {
				return err
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
		l, err := addRecursiveExclude("data", setting.AppDataPath, excludes, verbose)
		if err != nil {
			fatal("Failed to include data directory: %v", err)
		}
		files = append(files, l...)
	}

	if ctx.IsSet("skip-attachment-data") && ctx.Bool("skip-attachment-data") {
		log.Info("Skipping attachment data")
	} else if err := storage.Attachments.IterateObjects("", func(objPath string, object storage.Object) error {
		f, err := addObject(object, path.Join("data", "attachments", objPath), verbose)
		if err != nil {
			return err
		}

		files = append(files, f)
		return nil
	}); err != nil {
		fatal("Failed to dump attachments: %v", err)
	}

	if ctx.IsSet("skip-package-data") && ctx.Bool("skip-package-data") {
		log.Info("Skipping package data")
	} else if !setting.Packages.Enabled {
		log.Info("Package registry not enabled - skipping")
	} else if err := storage.Packages.IterateObjects("", func(objPath string, object storage.Object) error {
		f, err := addObject(object, path.Join("data", "packages", objPath), verbose)
		if err != nil {
			return err
		}

		files = append(files, f)
		return nil
	}); err != nil {
		fatal("Failed to dump packages: %v", err)
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
			l, err := addRecursiveExclude("log", setting.Log.RootPath, []string{absFileName}, verbose)
			if err != nil {
				fatal("Failed to include log: %v", err)
			}
			files = append(files, l...)
		}
	}

	if err := archiver.Archive(ctx.Context, file, files); err != nil {
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

// addRecursiveExclude zips absPath to specified insidePath inside writer excluding excludeAbsPath
// archives.FilesFromDisk doesn't support excluding files, so we have to do it manually
func addRecursiveExclude(insidePath, absPath string, excludeAbsPath []string, verbose bool) ([]archives.FileInfo, error) {
	var list []archives.FileInfo
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}
	dir, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		currentAbsPath := filepath.Join(absPath, file.Name())
		currentInsidePath := path.Join(insidePath, file.Name())

		if util.SliceContainsString(excludeAbsPath, currentAbsPath) {
			log.Debug("Skipping %q (matched an excluded path)", currentAbsPath)
			continue
		}

		if file.IsDir() {
			f, err := addFile(currentInsidePath, currentAbsPath, false)
			if err != nil {
				return nil, err
			}
			list = append(list, f)

			l, err := addRecursiveExclude(currentInsidePath, currentAbsPath, excludeAbsPath, verbose)
			if err != nil {
				return nil, err
			}
			list = append(list, l...)
		} else {
			// only copy regular files and symlink regular files, skip non-regular files like socket/pipe/...
			shouldAdd := file.Mode().IsRegular()
			if !shouldAdd && file.Mode()&os.ModeSymlink == os.ModeSymlink {
				target, err := filepath.EvalSymlinks(currentAbsPath)
				if err != nil {
					return nil, err
				}
				targetStat, err := os.Stat(target)
				if err != nil {
					return nil, err
				}
				shouldAdd = targetStat.Mode().IsRegular()
			}
			if shouldAdd {
				f, err := addFile(currentInsidePath, currentAbsPath, verbose)
				if err != nil {
					return nil, err
				}
				list = append(list, f)
			}
		}
	}
	return list, nil
}
