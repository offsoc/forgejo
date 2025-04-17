// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package log_test

import (
	"testing"

	"forgejo.org/modules/log"

	"github.com/stretchr/testify/assert"
)

func TestBufferLogger(t *testing.T) {
	prefix := "TestPrefix "
	level := log.INFO
	expected := "something"

	bufferWriter := log.NewEventWriterBuffer("test-buffer", log.WriterMode{
		Level:      level,
		Prefix:     prefix,
		Expression: expected,
	})

	logger := log.NewLoggerWithWriters(t.Context(), "test", bufferWriter)

	logger.SendLogEvent(&log.Event{
		Level:         log.INFO,
		MsgSimpleText: expected,
	})
	logger.Close()
	assert.Contains(t, bufferWriter.Buffer.String(), expected)
}
