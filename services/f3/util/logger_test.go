// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package util

import (
	"fmt"
	"testing"
	"time"

	forgejo_log "forgejo.org/modules/log"
	"forgejo.org/modules/test"

	"code.forgejo.org/f3/gof3/v3/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3UtilMessage(t *testing.T) {
	expected := "EXPECTED MESSAGE"
	var actual string
	logger := NewF3Logger(func(message string, args ...any) {
		actual = fmt.Sprintf(message, args...)
	}, nil)
	logger.Message("EXPECTED %s", "MESSAGE")
	assert.Equal(t, expected, actual)
}

func TestF3UtilLogger(t *testing.T) {
	for _, testCase := range []struct {
		level logger.Level
		call  func(logger.MessageInterface, string, ...any)
	}{
		{level: logger.Trace, call: func(logger logger.MessageInterface, message string, args ...any) { logger.Trace(message, args...) }},
		{level: logger.Debug, call: func(logger logger.MessageInterface, message string, args ...any) { logger.Debug(message, args...) }},
		{level: logger.Info, call: func(logger logger.MessageInterface, message string, args ...any) { logger.Info(message, args...) }},
		{level: logger.Warn, call: func(logger logger.MessageInterface, message string, args ...any) { logger.Warn(message, args...) }},
		{level: logger.Error, call: func(logger logger.MessageInterface, message string, args ...any) { logger.Error(message, args...) }},
		{level: logger.Fatal, call: func(logger logger.MessageInterface, message string, args ...any) { logger.Fatal(message, args...) }},
	} {
		t.Run(testCase.level.String(), func(t *testing.T) {
			testLoggerCase(t, testCase.level, testCase.call)
		})
	}
}

func testLoggerCase(t *testing.T, level logger.Level, loggerFunc func(logger.MessageInterface, string, ...any)) {
	lc, cleanup := test.NewLogChecker(forgejo_log.DEFAULT, f3LevelToForgejoLevel(level))
	defer cleanup()
	stopMark := "STOP"
	lc.StopMark(stopMark)
	filtered := []string{
		"MESSAGE HERE",
	}
	moreVerbose := logger.MoreVerbose(level)
	if moreVerbose != nil {
		filtered = append(filtered, "MESSAGE MORE VERBOSE")
	}
	lessVerbose := logger.LessVerbose(level)
	if lessVerbose != nil {
		filtered = append(filtered, "MESSAGE LESS VERBOSE")
	}
	lc.Filter(filtered...)

	logger := NewF3Logger(nil, forgejo_log.GetLogger(forgejo_log.DEFAULT))
	loggerFunc(logger, "MESSAGE %s", "HERE")
	if moreVerbose != nil {
		logger.Log(1, *moreVerbose, "MESSAGE %s", "MORE VERBOSE")
	}
	if lessVerbose != nil {
		logger.Log(1, *lessVerbose, "MESSAGE %s", "LESS VERBOSE")
	}
	logger.Fatal(stopMark)

	logFiltered, logStopped := lc.Check(5 * time.Second)
	assert.True(t, logStopped)
	i := 0
	assert.True(t, logFiltered[i], filtered[i])
	if moreVerbose != nil {
		i++
		require.Greater(t, len(logFiltered), i)
		assert.False(t, logFiltered[i], filtered[i])
	}
	if lessVerbose != nil {
		i++
		require.Greater(t, len(logFiltered), i)
		assert.True(t, logFiltered[i], filtered[i])
	}
}
