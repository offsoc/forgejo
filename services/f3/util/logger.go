// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package util

import (
	"fmt"

	forgejo_log "forgejo.org/modules/log"
	"forgejo.org/modules/migration"

	"code.forgejo.org/f3/gof3/v3/logger"
)

type f3Logger struct {
	m migration.Messenger
	l forgejo_log.Logger
}

func (o *f3Logger) Message(message string, args ...any) {
	if o.m != nil {
		o.m(message, args...)
	}
}

func (o *f3Logger) SetLevel(level logger.Level) {
}

func forgejoLevelToF3Level(level forgejo_log.Level) logger.Level {
	switch level {
	case forgejo_log.TRACE:
		return logger.Trace
	case forgejo_log.DEBUG:
		return logger.Debug
	case forgejo_log.INFO:
		return logger.Info
	case forgejo_log.WARN:
		return logger.Warn
	case forgejo_log.ERROR:
		return logger.Error
	case forgejo_log.FATAL:
		return logger.Fatal
	default:
		panic(fmt.Errorf("unexpected level %d", level))
	}
}

func f3LevelToForgejoLevel(level logger.Level) forgejo_log.Level {
	switch level {
	case logger.Trace:
		return forgejo_log.TRACE
	case logger.Debug:
		return forgejo_log.DEBUG
	case logger.Info:
		return forgejo_log.INFO
	case logger.Warn:
		return forgejo_log.WARN
	case logger.Error:
		return forgejo_log.ERROR
	case logger.Fatal:
		return forgejo_log.FATAL
	default:
		panic(fmt.Errorf("unexpected level %d", level))
	}
}

func (o *f3Logger) GetLevel() logger.Level {
	return forgejoLevelToF3Level(o.l.GetLevel())
}

func (o *f3Logger) Log(skip int, level logger.Level, format string, args ...any) {
	o.l.Log(skip+1, f3LevelToForgejoLevel(level), format, args...)
}

func (o *f3Logger) Trace(message string, args ...any) {
	o.l.Log(1, forgejo_log.TRACE, message, args...)
}

func (o *f3Logger) Debug(message string, args ...any) {
	o.l.Log(1, forgejo_log.DEBUG, message, args...)
}
func (o *f3Logger) Info(message string, args ...any) { o.l.Log(1, forgejo_log.INFO, message, args...) }
func (o *f3Logger) Warn(message string, args ...any) { o.l.Log(1, forgejo_log.WARN, message, args...) }
func (o *f3Logger) Error(message string, args ...any) {
	o.l.Log(1, forgejo_log.ERROR, message, args...)
}

func (o *f3Logger) Fatal(message string, args ...any) {
	o.l.Log(1, forgejo_log.FATAL, message, args...)
}

func NewF3Logger(messenger migration.Messenger, logger forgejo_log.Logger) logger.Interface {
	return &f3Logger{
		m: messenger,
		l: logger,
	}
}
