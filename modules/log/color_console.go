// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package log

// CanColorStdout reports if we can use ANSI escape sequences on stdout
var CanColorStdout = true

// CanColorStderr reports if we can use ANSI escape sequences on stderr
var CanColorStderr = true

// JournaldOnStdout reports whether stdout is attached to journald
var JournaldOnStdout = false

// JournaldOnStderr reports whether stderr is attached to journald
var JournaldOnStderr = false
