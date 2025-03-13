// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/mattn/go-isatty"
)

func journaldDevIno() (uint64, uint64, bool) {
	journaldStream := os.Getenv("JOURNAL_STREAM")
	if len(journaldStream) == 0 {
		return 0, 0, false
	}
	deviceStr, inodeStr, ok := strings.Cut(journaldStream, ":")
	device, err1 := strconv.ParseUint(deviceStr, 10, 64)
	inode, err2 := strconv.ParseUint(inodeStr, 10, 64)
	if !ok || err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return device, inode, true
}

func fileStatDevIno(file *os.File) (uint64, uint64, bool) {
	info, err := file.Stat()
	if err != nil {
		return 0, 0, false
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}

	// Do a type conversion to uint64, because Dev isn't always uint64
	// on every operating system + architecture combination.
	return uint64(stat.Dev), stat.Ino, true //nolint:unconvert
}

func fileIsDevIno(file *os.File, dev, ino uint64) bool {
	fileDev, fileIno, ok := fileStatDevIno(file)
	return ok && dev == fileDev && ino == fileIno
}

func init() {
	// When forgejo is running under service supervisor (e.g. systemd) with logging
	// set to console, the output streams are typically captured into some logging
	// system (e.g. journald or syslog) instead of going to the terminal. Disable
	// usage of ANSI escape sequences if that's the case to avoid spamming
	// the journal or syslog with garbled mess e.g. `#033[0m#033[32mcmd/web.go:102:#033[32m`.
	CanColorStdout = isatty.IsTerminal(os.Stdout.Fd())
	CanColorStderr = isatty.IsTerminal(os.Stderr.Fd())

	// Furthermore, check if we are running under journald specifically so that
	// further output adjustments can be applied. Specifically, this changes
	// the console logger defaults to disable duplication of date/time info and
	// enable emission of special control sequences understood by journald
	// instead of ANSI colors.
	journalDev, journalIno, ok := journaldDevIno()
	JournaldOnStdout = ok && !CanColorStdout && fileIsDevIno(os.Stdout, journalDev, journalIno)
	JournaldOnStderr = ok && !CanColorStderr && fileIsDevIno(os.Stderr, journalDev, journalIno)
}
