// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package log

import (
	"bytes"
	"strings"

	"forgejo.org/modules/json"
)

// Level is the level of the logger
type Level int

const (
	UNDEFINED Level = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	NONE
)

const CRITICAL = ERROR // most logger frameworks doesn't support CRITICAL, and it doesn't seem useful

var toString = map[Level]string{
	UNDEFINED: "undefined",

	TRACE: "trace",
	DEBUG: "debug",
	INFO:  "info",
	WARN:  "warn",
	ERROR: "error",

	FATAL: "fatal",
	NONE:  "none",
}

// Machine-readable log level prefixes as defined in sd-daemon(3).
//
// "If a systemd service definition file is configured with StandardError=journal
// or StandardError=kmsg (and similar with StandardOutput=), these prefixes can
// be used to encode a log level in lines printed. <...> To use these prefixes
// simply prefix every line with one of these strings. A line that is not prefixed
// will be logged at the default log level SD_INFO."
var toJournalPrefix = map[Level]string{
	TRACE: "<7>", // SD_DEBUG
	DEBUG: "<6>", // SD_INFO
	INFO:  "<5>", // SD_NOTICE
	WARN:  "<4>", // SD_WARNING
	ERROR: "<3>", // SD_ERR
	FATAL: "<2>", // SD_CRIT
}

var toLevel = map[string]Level{
	"undefined": UNDEFINED,

	"trace":   TRACE,
	"debug":   DEBUG,
	"info":    INFO,
	"warn":    WARN,
	"warning": WARN,
	"error":   ERROR,

	"fatal": FATAL,
	"none":  NONE,
}

var levelToColor = map[Level][]ColorAttribute{
	TRACE: {Bold, FgCyan},
	DEBUG: {Bold, FgBlue},
	INFO:  {Bold, FgGreen},
	WARN:  {Bold, FgYellow},
	ERROR: {Bold, FgRed},
	FATAL: {Bold, BgRed},
	NONE:  {Reset},
}

func (l Level) String() string {
	s, ok := toString[l]
	if ok {
		return s
	}
	return "info"
}

func (l Level) JournalPrefix() string {
	return toJournalPrefix[l]
}

func (l Level) ColorAttributes() []ColorAttribute {
	color, ok := levelToColor[l]
	if ok {
		return color
	}
	none := levelToColor[NONE]
	return none
}

// MarshalJSON takes a Level and turns it into text
func (l Level) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(toString[l])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON takes text and turns it into a Level
func (l *Level) UnmarshalJSON(b []byte) error {
	var tmp any
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	switch v := tmp.(type) {
	case string:
		*l = LevelFromString(v)
	case int:
		*l = LevelFromString(Level(v).String())
	default:
		*l = INFO
	}
	return nil
}

// LevelFromString takes a level string and returns a Level
func LevelFromString(level string) Level {
	if l, ok := toLevel[strings.ToLower(level)]; ok {
		return l
	}
	return INFO
}
